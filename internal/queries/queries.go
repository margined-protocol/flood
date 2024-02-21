package queries

import (
	"context"
	"fmt"
	"sync"

	"github.com/osmosis-labs/osmosis/v21/tests/e2e/util"
	cl "github.com/osmosis-labs/osmosis/v21/x/concentrated-liquidity/client/queryproto"
	cltypes "github.com/osmosis-labs/osmosis/v21/x/concentrated-liquidity/types"
	poolmanager "github.com/osmosis-labs/osmosis/v21/x/poolmanager/client/queryproto"
	pmtypes "github.com/osmosis-labs/osmosis/v21/x/poolmanager/types"

	"github.com/margined-protocol/flood/internal/types"
)

func GetUserPositions(ctx context.Context, client cl.QueryClient, poolConfig types.Pool, user string) (*cl.UserPositionsResponse, error) {
	// Now lets check if we have any open CL positions for the bot
	req := cl.UserPositionsRequest{
		PoolId:  poolConfig.ID,
		Address: user,
	}

	userPositions, err := client.UserPositions(ctx, &req)
	if err != nil {
		return &cl.UserPositionsResponse{}, err
	}

	return userPositions, nil
}

func GetSpotPrice(ctx context.Context, client poolmanager.QueryClient, poolConfig types.Pool) (string, error) {
	req := poolmanager.SpotPriceRequest{
		PoolId:          poolConfig.ID,
		BaseAssetDenom:  poolConfig.BaseDenom,
		QuoteAssetDenom: poolConfig.QuoteDenom,
	}

	fmt.Println("Requesting spot price for pool", poolConfig.ID, "with base asset", poolConfig.BaseDenom, "and quote asset", poolConfig.QuoteDenom)

	spotPrice, err := client.SpotPrice(ctx, &req)
	if err != nil {
		return "", err
	}

	return spotPrice.SpotPrice, nil
}

func GetCurrentTick(ctx context.Context, client poolmanager.QueryClient, poolId uint64) (int64, error) {

	poolReq := poolmanager.PoolRequest{PoolId: poolId}
	res, err := client.Pool(ctx, &poolReq)
	if err != nil {
		return 0, err
	}

	var pool pmtypes.PoolI
	err = util.Cdc.UnpackAny(res.Pool, &pool)
	if err != nil {
		return 0, err
	}

	currentTick := pool.(cltypes.ConcentratedPoolExtension).GetCurrentTick()

	return currentTick, nil

}

func GetTotalPoolLiquidity(ctx context.Context, client poolmanager.QueryClient, poolId uint64) (*poolmanager.TotalPoolLiquidityResponse, error) {

	return client.TotalPoolLiquidity(ctx, &poolmanager.TotalPoolLiquidityRequest{PoolId: poolId})
}

func GetSpotPrices(ctx context.Context, poolManagerClient poolmanager.QueryClient, config types.GetConfigResponse) (string, string, error) {
	var baseSpotPrice, powerSpotPrice string
	var err error

	// WaitGroup to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(2)

	// Channel to capture errors
	errChan := make(chan error, 2)

	// Fetch base spot price
	go func() {
		defer wg.Done()
		baseSpotPrice, err = GetSpotPrice(ctx, poolManagerClient, config.BasePool)
		if err != nil {
			errChan <- err
		}
	}()

	// Fetch power spot price
	go func() {
		defer wg.Done()
		powerSpotPrice, err = GetSpotPrice(ctx, poolManagerClient, config.PowerPool)
		if err != nil {
			errChan <- err
		}
	}()

	// Wait for goroutines to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	for e := range errChan {
		if e != nil {
			return "", "", e
		}
	}

	return baseSpotPrice, powerSpotPrice, nil
}
