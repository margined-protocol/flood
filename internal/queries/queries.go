package queries

import (
	"context"
	"fmt"
	"sync"

	"github.com/cenkalti/backoff/v4"
	"github.com/osmosis-labs/osmosis/v24/tests/e2e/util"
	cl "github.com/osmosis-labs/osmosis/v24/x/concentrated-liquidity/client/queryproto"
	cltypes "github.com/osmosis-labs/osmosis/v24/x/concentrated-liquidity/types"
	poolmanager "github.com/osmosis-labs/osmosis/v24/x/poolmanager/client/queryproto"
	pmtypes "github.com/osmosis-labs/osmosis/v24/x/poolmanager/types"

	floodbackoff "github.com/margined-protocol/flood/internal/backoff"
	"github.com/margined-protocol/flood/internal/types"
)

func GetUserPositions(ctx context.Context, client cl.QueryClient, poolConfig types.Pool, user string) (*cl.UserPositionsResponse, error) {
	// Now lets check if we have any open CL positions for the bot
	req := cl.UserPositionsRequest{
		PoolId:  poolConfig.ID,
		Address: user,
	}

	var userPositions *cl.UserPositionsResponse
	var err error

	exponentialBackoff := floodbackoff.NewBackoff(ctx)

	retryableRequest := func() error {
		userPositions, err = client.UserPositions(ctx, &req)
		return err
	}

	if err := backoff.Retry(retryableRequest, exponentialBackoff); err != nil {
		return nil, err
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

	exponentialBackoff := floodbackoff.NewBackoff(ctx)

	var spotPriceResponse *poolmanager.SpotPriceResponse
	var err error

	retryableRequest := func() error {
		spotPriceResponse, err = client.SpotPrice(ctx, &req)
		return err
	}

	err = backoff.Retry(retryableRequest, exponentialBackoff)

	if err != nil {
		return "", err
	} else {
		return spotPriceResponse.SpotPrice, nil
	}

}

func GetCurrentTick(ctx context.Context, client poolmanager.QueryClient, poolId uint64) (int64, error) {

	poolReq := poolmanager.PoolRequest{PoolId: poolId}

	exponentialBackoff := floodbackoff.NewBackoff(ctx)

	var res *poolmanager.PoolResponse
	var err error

	retryableRequest := func() error {
		res, err = client.Pool(ctx, &poolReq)
		return err
	}

	err = backoff.Retry(retryableRequest, exponentialBackoff)
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

	req := poolmanager.TotalPoolLiquidityRequest{PoolId: poolId}

	exponentialBackoff := floodbackoff.NewBackoff(ctx)

	var totalPoolLiquidityReponse *poolmanager.TotalPoolLiquidityResponse
	var err error

	retryableRequest := func() error {
		totalPoolLiquidityReponse, err = client.TotalPoolLiquidity(ctx, &req)
		return err
	}

	err = backoff.Retry(retryableRequest, exponentialBackoff)

	if err != nil {
		return &poolmanager.TotalPoolLiquidityResponse{}, err
	} else {
		return totalPoolLiquidityReponse, nil
	}

}

func GetSpotPrices(ctx context.Context, poolManagerClient poolmanager.QueryClient, config types.GetConfigResponse) (string, string, error) {
	var baseSpotPrice, powerSpotPrice string

	// WaitGroup to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(2)

	// Channel to capture errors
	errChan := make(chan error, 2)

	// Fetch base spot price
	go func() {
		defer wg.Done()
		spotPrice, err := GetSpotPrice(ctx, poolManagerClient, config.BasePool)
		if err != nil {
			errChan <- err
			return
		}
		baseSpotPrice = spotPrice
	}()

	// Fetch power spot price
	go func() {
		defer wg.Done()
		spotPrice, err := GetSpotPrice(ctx, poolManagerClient, config.PowerPool)
		if err != nil {
			errChan <- err
			return
		}
		powerSpotPrice = spotPrice
	}()

	// Wait for goroutines to complete
	wg.Wait()
	close(errChan)

	// Check for errors
	for e := range errChan {
		if e != nil {
			return "", "", e // Return on the first error encountered
		}
	}

	return baseSpotPrice, powerSpotPrice, nil
}
