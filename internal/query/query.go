package query

import (
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	clquery "github.com/osmosis-labs/osmosis/v21/x/concentrated-liquidity/client/queryproto"
	pmquery "github.com/osmosis-labs/osmosis/v21/x/poolmanager/client/queryproto"
	"google.golang.org/grpc"

	"github.com/cosmos/cosmos-sdk/x/authz"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// Client is a wrapper with Cosmos and Osmosis grpc query clients
type Client struct {
	// cosmos-sdk query clients
	Authz authz.QueryClient
	Bank  banktypes.QueryClient

	// wasmd query clients
	Wasm wasmtypes.QueryClient

	// osmosis query clients
	ConcentratedLiquidity clquery.QueryClient
	PoolManager           pmquery.QueryClient
}

// NewQueryClient creates a new QueryClient and initializes all the module query clients
func NewQueryClient(conn *grpc.ClientConn) (*Client, error) {
	client := &Client{
		Authz:                 authz.NewQueryClient(conn),
		Bank:                  banktypes.NewQueryClient(conn),
		Wasm:                  wasmtypes.NewQueryClient(conn),
		ConcentratedLiquidity: clquery.NewQueryClient(conn),
		PoolManager:           pmquery.NewQueryClient(conn),
	}
	return client, nil
}
