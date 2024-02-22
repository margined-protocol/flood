package types

type SigningKey struct {
	AppName string `toml:"app_name"`
	Backend string `toml:"backend"`
	RootDir string `toml:"root_dir"`
}

type Asset struct {
	Decimals uint64 `toml:"decimals"`
	Denom    string `toml:"denom"`
}

type PowerPool struct {
	PoolId          uint64 `toml:"pool_id"`
	BaseAsset       string `toml:"base_asset"`
	QuoteAsset      string `toml:"quote_asset"`
	ContractAddress string `toml:"contract_address"`
}

type Position struct {
	DefaultToken0Amount int64  `toml:"default_token_0_amount"`
	DefaultToken1Amount int64  `toml:"default_token_1_amount"`
	Spread              string `toml:"spread"`
	LpSpread            string `toml:"lp_spread"`
}

type Config struct {
	AddressPrefix     string     `toml:"address_prefix"`
	Fees              string     `toml:"fees"`
	GasAdjustment     float64    `toml:"gas_adjustment"`
	Gas               string     `toml:"gas"`
	GRPCServerAddress string     `toml:"grpc_server_address"`
	Key               SigningKey `toml:"key"`
	Memo              string     `toml:"memo"`
	PowerPool         PowerPool  `toml:"power_pool"`
	RPCServerAddress  string     `toml:"rpc_server_address"`
	WebsocketPath     string     `toml:"websocket_path"`
	SignerAccount     string     `toml:"signer_account"`
	Position          Position   `toml:"position"`
}

// getVaultResponse represents the response structure for querying information about a vault.
// It includes the operator's address, the amount of collateral, and the short amount of the vault.
type GetStateResponse struct {
	IsOpen              bool   `json:"is_open"`
	IsPaused            bool   `json:"is_paused"`
	LastPause           string `json:"last_pause"`
	NormalisationFactor string `json:"normalisation_factor"`
	LastFundingUpdate   string `json:"last_funding_update"`
}

// getVaultResponse represents the response structure for querying information about a vault.
// It includes the operator's address, the amount of collateral, and the short amount of the vault.
type GetConfigResponse struct {
	QueryContract       string `json:"query_contract"`
	FeePoolContract     string `json:"fee_pool_contract"`
	FeeRate             string `json:"fee_rate"`
	PowerAsset          Asset  `json:"power_asset"`
	BaseAsset           Asset  `json:"base_asset"`
	BasePool            Pool   `json:"base_pool"`
	PowerPool           Pool   `json:"power_pool"`
	FundingPeriod       int    `json:"funding_period"`
	BaseDecimals        int    `json:"base_decimals"`
	PowerDecimals       int    `json:"power_decimals"`
	IndexScale          int    `json:"index_scale"`
	MinCollateralAmount string `json:"min_collateral_amount"`
	Version             string `json:"version"`
}

type Pool struct {
	ID         uint64 `json:"id"`
	BaseDenom  string `json:"base_denom"`
	QuoteDenom string `json:"quote_denom"`
}
