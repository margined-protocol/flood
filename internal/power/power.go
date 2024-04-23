package power

import (
	"context"
	"encoding/json"
	"sync"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/margined-protocol/flood/internal/types"
)

// querySmartContract executes a generic query to a smart contract.
func querySmartContract(ctx context.Context, pa string, c wasmtypes.QueryClient, query string, out interface{}) error {
	req := &wasmtypes.QuerySmartContractStateRequest{
		Address:   pa,
		QueryData: []byte(query),
	}

	res, err := c.SmartContractState(ctx, req)
	if err != nil {
		return err
	}

	return json.Unmarshal(res.Data, out)
}

// GetState queries the power contract state.
func getState(ctx context.Context, pa string, c wasmtypes.QueryClient) (types.GetStateResponse, error) {
	var getStatusData types.GetStateResponse
	err := querySmartContract(ctx, pa, c, `{"state": {}}`, &getStatusData)
	return getStatusData, err
}

// GetConfig queries the power contract configuration.
func getConfig(ctx context.Context, pa string, c wasmtypes.QueryClient) (types.GetConfigResponse, error) {
	var getConfigData types.GetConfigResponse
	err := querySmartContract(ctx, pa, c, `{"config": {}}`, &getConfigData)
	return getConfigData, err
}

func GetConfigAndState(ctx context.Context, wasmClient wasmtypes.QueryClient, contractAddress string) (types.GetConfigResponse, types.GetStateResponse, error) {
	var config types.GetConfigResponse
	var state types.GetStateResponse
	var err error

	// WaitGroup to wait for goroutines to finish
	var wg sync.WaitGroup
	wg.Add(2)

	// Error channel to capture errors from goroutines
	errChan := make(chan error, 2)

	// Fetch config concurrently
	go func() {
		defer wg.Done()
		config, err = getConfig(ctx, contractAddress, wasmClient)
		if err != nil {
			errChan <- err
		}
	}()

	// Fetch state concurrently
	go func() {
		defer wg.Done()
		state, err = getState(ctx, contractAddress, wasmClient)
		if err != nil {
			errChan <- err
		}
	}()

	// Wait for both goroutines to complete
	wg.Wait()
	close(errChan)

	// Check if any errors occurred
	for e := range errChan {
		if e != nil {
			return types.GetConfigResponse{}, types.GetStateResponse{}, e
		}
	}

	return config, state, nil
}
