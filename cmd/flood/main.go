package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"go.uber.org/zap"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/margined-protocol/flood/internal/config"
	"github.com/margined-protocol/flood/internal/liquidity"
	"github.com/margined-protocol/flood/internal/logger"
	"github.com/margined-protocol/flood/internal/maths"
	"github.com/margined-protocol/flood/internal/power"
	"github.com/margined-protocol/flood/internal/queries"
	"github.com/margined-protocol/flood/internal/types"

	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
	"github.com/ignite/cli/ignite/pkg/cosmosclient"

	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
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

func handleEvent(l *zap.Logger, cfg *types.Config, ctx context.Context, address string, account cosmosaccount.Account, clients types.BlockchainClients, event ctypes.ResultEvent) {

	// Get the power config and state
	powerConfig, powerState, err := power.GetConfigAndState(ctx, clients.WasmClient, clients.Config.PowerPool.ContractAddress)
	if err != nil {
		l.Fatal("Failed to get config and state: %v", zap.Error(err))
	}

	// Get the spotprices for base and power
	baseSpotPrice, powerSpotPrice, err := queries.GetSpotPrices(ctx, clients.PMClient, powerConfig)
	if err != nil {
		l.Fatal("Failed to fetch spot prices", zap.Error(err))
	}

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

	// get inverse target and spot prices
	floatPowerSpotPrice, err := strconv.ParseFloat(powerSpotPrice, 64)
	if err != nil {
		l.Fatal("Failed to parse power spot price", zap.Error(err))
	}

	inverseTargetPrice := 1 / targetPrice
	inversePowerPrice := 1 / floatPowerSpotPrice

	// Now lets check if we have any open CL positions for the bot
	userPositions, err := queries.GetUserPositions(ctx, clients.CLClient, powerConfig.PowerPool, address)
	if err != nil {
		l.Fatal("Failed to find user positions", zap.Error(err))
	}

	currentTick, err := queries.GetCurrentTick(ctx, clients.PMClient, powerConfig.PowerPool.ID)
	if err != nil {
		l.Fatal("Failed to get current tick", zap.Error(err))
	}

	// Sanity check computations
	l.Debug("Summary data",
		zap.Float64("mark_price", markPrice),
		zap.Float64("target_price", targetPrice),
		zap.Float64("inverse_target_price", inverseTargetPrice),
		zap.String("power_price", powerSpotPrice),
		zap.Float64("inverse_power_price", inversePowerPrice),
		zap.Float64("premium", premium),
		zap.String("normalization_factor", powerState.NormalisationFactor),
		zap.Int64("current_tick", currentTick),
	)

	powerPriceStr := fmt.Sprintf("%f", inversePowerPrice)
	targetPriceStr := fmt.Sprintf("%f", inverseTargetPrice)

	msgs, err := liquidity.CreateUpdatePositionMsgs(l, *userPositions, cfg, currentTick, address, powerPriceStr, targetPriceStr)
	if err != nil {
		l.Fatal("Failed to create update position msgs", zap.Error(err))
	}

	txResp, err := clients.CosmosClient.BroadcastTx(ctx, account, msgs...)
	if err != nil {
		l.Error("Transaction error",
			zap.Error(err),
		)
	} else {
		l.Debug("tx response",
			zap.String("transaction hash", txResp.TxHash),
		)
	}
}

func main() {
	parseFlags()
	if *showVersion {
		fmt.Printf("Version: %s\nBuild Date: %s\n", Version, BuildDate)
		os.Exit(0)
	}

	ctx := context.Background()

	// Intialise logger, config, comsosclient and grpc client
	l, cfg, client, conn := initialize(ctx, *configPath)
	defer conn.Close()

	// Get the client account
	account, err := client.Account(cfg.SignerAccount)
	if err != nil {
		l.Fatal("Error fetching signer account",
			zap.Error(err),
		)
	}

	// Get the client address
	//nolint:staticcheck
	address, err := account.Address(cfg.AddressPrefix)
	if err != nil {
		l.Fatal("Error fetching signer address",
			zap.Error(err),
		)
	}

	// Initialise a wasm query client to read state from power contract
	//nolint:staticcheck
	c := wasmtypes.NewQueryClient(client.Context())

	// Initialise a poolmanager query client
	//nolint:staticcheck
	pmClient := pmquery.NewQueryClient(client.Context())

	// Initialise a concentrated liquidity query client
	//nolint:staticcheck
	clClient := clquery.NewQueryClient(client.Context())

	// Initialise a websocket client
	wsClient, err := rpchttp.New(cfg.RPCServerAddress, cfg.WebsocketPath)
	if err != nil {
		l.Fatal("Error subscribing to websocket client", zap.Error(err))
	}

	err = wsClient.Start()
	if err != nil {
		l.Fatal("Error starting websocket client",
			zap.Error(err),
		)
	}

	// Generate the query we are listening for, in this case tokens swapped in a pool
	query := fmt.Sprintf("token_swapped.module = 'gamm' AND token_swapped.pool_id = '%d'", cfg.PowerPool.PoolId)
	// query := "token_swapped.module = 'gamm'"

	// An arbitraty string to identify the subscription needed for the client
	subscriber := "gobot"

	//nolint:staticcheck
	eventCh, err := wsClient.Subscribe(ctx, subscriber, query)
	if err != nil {
		l.Fatal("Error subscribing websocket client",
			zap.Error(err),
		)
	}

	// Wrap the numerous clients for convenience
	clients := types.BlockchainClients{
		CosmosClient:    client,
		WebsocketClient: wsClient,
		WasmClient:      c,
		PMClient:        pmClient,
		CLClient:        clClient,
		Config:          cfg,
	}

	go func() {
		for {
			event := <-eventCh
			handleEvent(l, cfg, ctx, address, account, clients, event)
		}
	}()

	// Keep the main goroutine running
	select {}

}
