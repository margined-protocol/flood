package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"

	"go.uber.org/zap"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	floodauthz "github.com/margined-protocol/flood/internal/authz"
	"github.com/margined-protocol/flood/internal/config"
	"github.com/margined-protocol/flood/internal/liquidity"
	"github.com/margined-protocol/flood/internal/logger"
	"github.com/margined-protocol/flood/internal/maths"
	"github.com/margined-protocol/flood/internal/power"
	"github.com/margined-protocol/flood/internal/queries"
	"github.com/margined-protocol/flood/internal/query"
	"github.com/margined-protocol/flood/internal/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/ignite/cli/ignite/pkg/cosmosaccount"
	"github.com/ignite/cli/ignite/pkg/cosmosclient"

	rpchttp "github.com/cometbft/cometbft/rpc/client/http"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
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
	return grpc.NewClient(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
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

func handleEvent(l *zap.Logger, cfg *types.Config, ctx context.Context, cosmosClient *cosmosclient.Client, queryClient *query.QueryClient, event ctypes.ResultEvent) {
	// Get the signer account
	account, err := cosmosClient.Account(cfg.SignerAccount)
	if err != nil {
		l.Fatal("Error fetching signer account",
			zap.Error(err),
		)
	}

	// Get the signer address
	address, err := account.Address(cfg.AddressPrefix)
	if err != nil {
		l.Fatal("Error fetching signer address",
			zap.Error(err),
		)
	}

	validGranters, err := floodauthz.GetValidGrantersWithRequiredGrants(ctx, queryClient, address, l)
	if err != nil {
		l.Fatal("Error fetching valid granters", zap.Error(err))
	}

	// Initialize a slice to hold all messages
	var allMsgs []sdk.Msg

	for _, granter := range validGranters {
		l.Debug("Granter with all required grants", zap.String("granter", granter))

		// Get the power config and state
		powerConfig, powerState, err := power.GetConfigAndState(ctx, queryClient.Wasm, cfg.PowerPool.ContractAddress)
		if err != nil {
			l.Fatal("Failed to get config and state: %v", zap.Error(err))
		}

		// Get the spotprices for base and power
		baseSpotPrice, powerSpotPrice, err := queries.GetSpotPrices(ctx, queryClient.PoolManager, powerConfig)
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

		// Now lets check if we have any open CL positions for the granter
		userPositions, err := queries.GetUserPositions(ctx, queryClient.ConcentratedLiquidity, powerConfig.PowerPool, granter)
		if err != nil {
			l.Fatal("Failed to find user positions", zap.Error(err))
		}

		currentTick, err := queries.GetCurrentTick(ctx, queryClient.PoolManager, powerConfig.PowerPool.ID)
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

		msgs, err := liquidity.CreateUpdatePositionMsgs(l, *userPositions, cfg, currentTick, granter, powerPriceStr, targetPriceStr)
		if err != nil {
			l.Fatal("Failed to create update position msgs", zap.Error(err))
		}

		allMsgs = append(allMsgs, msgs...)
	}

	grantee, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		l.Error("Failed to generate AccAddress", zap.String("address", address))
	}

	msgExec := authz.NewMsgExec(grantee, allMsgs)

	txResp, err := cosmosClient.BroadcastTx(ctx, account, &msgExec)
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
	l, cfg, cosmosClient, conn := initialize(ctx, *configPath)
	defer conn.Close()

	// Initialise a grpc query client
	queryClient, err := query.NewQueryClient(conn)
	if err != nil {
		l.Fatal("Error intitialising query client",
			zap.Error(err),
		)
	}

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

	eventCh, err := wsClient.Subscribe(ctx, subscriber, query)
	if err != nil {
		l.Fatal("Error subscribing websocket client",
			zap.Error(err),
		)
	}

	go func() {
		for {
			event := <-eventCh
			handleEvent(l, cfg, ctx, cosmosClient, queryClient, event)
		}
	}()

	// Keep the main goroutine running
	select {}

}
