package liquidity

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	clquery "github.com/osmosis-labs/osmosis/v21/x/concentrated-liquidity/client/queryproto"
	"go.uber.org/zap"

	"github.com/margined-protocol/flood/internal/types"
)

func CreateUpdatePositionMsgs(l *zap.Logger, p clquery.UserPositionsResponse, cfg *types.Config, address, powerPrice, targetPrice string) ([]sdk.Msg, error) {
	var msgs []sdk.Msg

	var token0 sdk.Coin
	var token1 sdk.Coin

	if p.Positions == nil {
		l.Info("No positions found")

		token0.Amount = sdk.NewInt(cfg.Position.DefaultToken0Amount)
		token0.Denom = cfg.PowerPool.BaseAsset

		token1.Amount = sdk.NewInt(cfg.Position.DefaultToken1Amount)
		token1.Denom = cfg.PowerPool.QuoteAsset

	}

	if len(p.Positions) == 2 {
		l.Info("Found open positions")

		l.Debug("existing positions",
			zap.Reflect("Positions", p.Positions),
		)

		removeMsgs := RemovePreviousPositions(l, p.Positions)
		msgs = append(msgs, removeMsgs...)

		l.Debug("removing positions",
			zap.Reflect("removeMsgs", removeMsgs),
		)

		amount0 := p.Positions[0].Asset0.AddAmount(p.Positions[1].Asset0.Amount)
		amount1 := p.Positions[0].Asset1.AddAmount(p.Positions[1].Asset1.Amount)

		token0 = sdk.NewCoin(p.Positions[0].Asset0.Denom, amount0.Amount)
		token1 = sdk.NewCoin(p.Positions[0].Asset1.Denom, amount1.Amount)

		l.Debug("tokens",
			zap.Int64("token0", token0.Amount.Int64()),
			zap.Int64("token1", token1.Amount.Int64()),
		)

	}

	positionMsgs, err := MarketMake(l, cfg.PowerPool.PoolId, powerPrice, targetPrice, cfg.Position.Spread, token0, token1, address)
	if err != nil {
		l.Fatal("Failed to market make", zap.Error(err))
		return nil, err
	}

	return append(msgs, positionMsgs...), nil

}
