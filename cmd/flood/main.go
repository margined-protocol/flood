package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"go.uber.org/zap"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/margined-protocol/flood/internal/config"
	"github.com/margined-protocol/flood/internal/logger"
	"github.com/margined-protocol/flood/internal/maths"
	"github.com/margined-protocol/flood/internal/power"
	"github.com/margined-protocol/flood/internal/queries"
	"github.com/margined-protocol/flood/internal/types"

	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
	"github.com/ignite/cli/ignite/pkg/cosmosclient"
	clquery "github.com/osmosis-labs/osmosis/v21/x/concentrated-liquidity/client/queryproto"
	pmquery "github.com/osmosis-labs/osmosis/v21/x/poolmanager/client/queryproto"
)

var (
	// version and buildDate is set with -ldflags in the Makefile
	Version     string
	BuildDate   string
	configPath  *string
	showVersion *bool
)

func parseFlags() {
	configPath = flag.String("c", "config.toml", "path to config file")
	showVersion = flag.Bool("v", false, "Print the version of the program")
	flag.Parse()
}

// setup client initialises a cosmos client that maybe used to submit transactions
func setupCosmosClient(ctx context.Context, cfg *types.Config) (*cosmosclient.Client, error) {
	opts := []cosmosclient.Option{
		cosmosclient.WithNodeAddress(cfg.RPCServerAddress),
		cosmosclient.WithGas(cfg.Gas),
		cosmosclient.WithGasAdjustment(cfg.GasAdjustment),
		cosmosclient.WithAddressPrefix(cfg.AddressPrefix),
		cosmosclient.WithKeyringBackend(cosmosaccount.KeyringBackend(cfg.Key.Backend)),
		cosmosclient.WithFees(cfg.Fees),
		cosmosclient.WithKeyringDir(cfg.Key.RootDir),
		cosmosclient.WithKeyringServiceName(cfg.Key.AppName),
	}

	client, err := cosmosclient.New(ctx, opts...)
	if err != nil {
		return nil, err
	}

	return &client, nil
}

// setup GRPC connection establishes a GRPC connection
func setupGRPCConnection(address string) (*grpc.ClientConn, error) {
	return grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
}

// initialise performs the setup operations for the script
// * initialise a logger
// * load and parse config
// * initialise a cosmosclient
// * initilise a grpc connection
func initialize(ctx context.Context, configPath string) (*zap.Logger, *types.Config, *cosmosclient.Client, *grpc.ClientConn) {
	l, err := logger.Setup()
	if err != nil {
		log.Fatalf("Failed to initialize zap logger: %v", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		l.Fatal("Failed to load config", zap.Error(err))
	}

	client, err := setupCosmosClient(ctx, cfg)
	if err != nil {
		l.Fatal("Failed to initialise cosmosclient", zap.Error(err))
	}

	conn, err := setupGRPCConnection(cfg.GRPCServerAddress)
	if err != nil {
		l.Fatal("Failed to connect to GRPC server", zap.Error(err))
	}

	return l, cfg, client, conn
}

func main() {
	parseFlags()
	if *showVersion {
		fmt.Printf("Version: %s\nBuild Date: %s\n", Version, BuildDate)
		os.Exit(0)
	}

	fmt.Println("Version:", Version)
	fmt.Println("configPath:", configPath)
	fmt.Println("showVersion:", showVersion)

	ctx := context.Background()

	// Intialise logger, config, comsosclient and grpc client
	l, cfg, client, conn := initialize(ctx, *configPath)
	defer conn.Close()

	fmt.Println(client.AccountRegistry.List())

	// Get the client account
	account, err := client.Account(cfg.SignerAccount)
	if err != nil {
		l.Fatal("Error fetching signer account",
			zap.Error(err),
		)
	}

	// Get the client address
	address, err := account.Address(cfg.AddressPrefix)
	if err != nil {
		l.Fatal("Error fetching signer address",
			zap.Error(err),
		)
	}

	// Initialise a wasm query client to read state from power contract
	c := wasmtypes.NewQueryClient(client.Context())

	// Initialise a poolmanager query client
	poolmanagerClient := pmquery.NewQueryClient(client.Context())

	// Initialise a concentrated liquidity query client
	clClient := clquery.NewQueryClient(client.Context())

	// Get the power config and state
	powerConfig, powerState, err := power.GetConfigAndState(ctx, c)
	if err != nil {
		log.Fatalf("Failed to get config and state: %v", err)
	}

	// Get the spotprices for base and power
	baseSpotPrice, powerSpotPrice, err := queries.GetSpotPrices(ctx, poolmanagerClient, powerConfig)
	if err != nil {
		l.Fatal("Failed to fetch spot prices", zap.Error(err))
	}

	fmt.Println("baseSpotPrice:", powerConfig)
	// Calculate the mark price
	markPrice, err := maths.CalculateMarkPrice(baseSpotPrice, powerSpotPrice, powerState.NormalisationFactor, powerConfig.IndexScale)
	if err != nil {
		l.Fatal("Failed to calculate mark price", zap.Error(err))
	}

	// Calcuate the index price
	indexPrice, err := maths.CalculateIndexPrice(baseSpotPrice)
	if err != nil {
		l.Fatal("Failed to calculate index price", zap.Error(err))
	}

	// Calculate the target price
	targetPrice, err := maths.CalculateTargetPrice(baseSpotPrice, powerState.NormalisationFactor, powerConfig.IndexScale)
	if err != nil {
		l.Fatal("Failed to calculate target price", zap.Error(err))
	}

	// Calculate the premium
	premium := maths.CalculatePremium(markPrice, indexPrice)

	// Sanity check computations
	l.Debug("Summary data",
		zap.String("base_spot_price", baseSpotPrice),
		zap.Float64("mark_price", markPrice),
		zap.Float64("target_price", targetPrice),
		zap.String("power_price", powerSpotPrice),
		zap.Float64("index_price", indexPrice),
		zap.Float64("premium", premium),
		zap.String("normalization_factor", powerState.NormalisationFactor),
	)

	// Now lets check if we have any open CL positions for the bot
	req := clquery.UserPositionsRequest{
		PoolId:  powerConfig.PowerPool.ID,
		Address: address,
	}
	l.Debug("request -> ",
		zap.Reflect("request", req),
	)
	userPositions, err := clClient.UserPositions(ctx, &req)
	if err != nil {
		l.Fatal("Failed to find user positions", zap.Error(err))
	}
	l.Debug("liquidity in net direction -> ",
		zap.Reflect("response", userPositions),
	)

	if userPositions.Positions == nil {
		l.Info("No positions found")

	}

}
