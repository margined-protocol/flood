package liquidity

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/osmosis-labs/osmosis/osmomath"
	clmath "github.com/osmosis-labs/osmosis/v21/x/concentrated-liquidity/math"
	model "github.com/osmosis-labs/osmosis/v21/x/concentrated-liquidity/model"
	cltypes "github.com/osmosis-labs/osmosis/v21/x/concentrated-liquidity/types"
	"go.uber.org/zap"
)

const TICK_SPACING = int64(100)

// createPositionMsg creates a new CL position message
func createPositionMsg(poolId uint64, lowerTick, upperTick int64, tokens sdk.Coins, addr string) sdk.Msg {
	// Generate the swap message
	msg := cltypes.MsgCreatePosition{
		PoolId:          poolId,
		Sender:          addr,
		LowerTick:       lowerTick,
		UpperTick:       upperTick,
		TokensProvided:  tokens,
		TokenMinAmount0: sdk.NewInt(1),
		TokenMinAmount1: sdk.NewInt(1),
	}

	return &msg
}

// removePositionMsg withdraws positions with specific ids
func removePositionMsg(position model.Position) sdk.Msg {
	// Generate the swap message
	msg := cltypes.MsgWithdrawPosition{
		PositionId:      position.PositionId,
		Sender:          position.Address,
		LiquidityAmount: position.Liquidity,
	}

	return &msg
}

// marketMake creates a market making positions
func RemovePreviousPositions(l *zap.Logger, positions []model.FullPositionBreakdown) []sdk.Msg {
	var msgs []sdk.Msg

	for _, p := range positions {
		l.Debug("position",
			zap.Uint64("positionId", p.Position.PositionId),
			zap.String("liquidity", p.Position.Liquidity.String()),
		)

		msg := removePositionMsg(p.Position)

		msgs = append(msgs, msg)
	}

	return msgs
}

// marketMake creates a market making positions
func MarketMake(l *zap.Logger, poolId uint64, spotPrice, targetPrice, spread string, token0 sdk.Coin, token1 sdk.Coin, addr string) ([]sdk.Msg, error) {
	l.Debug("inputs",
		zap.String("spotPrice", spotPrice),
		zap.String("targetPrice", targetPrice),
	)

	spotPriceAsBigDec, err := osmomath.NewBigDecFromStr(spotPrice)
	if err != nil {
		l.Error("Failed to convert spot price to big dec", zap.Error(err))
		return nil, err
	}

	targetPriceAsBigDec, err := osmomath.NewBigDecFromStr(targetPrice)
	if err != nil {
		l.Error("Failed to convert target price to big dec", zap.Error(err))
		return nil, err
	}

	spreadAsBigDec, err := osmomath.NewBigDecFromStr(spread)
	if err != nil {
		l.Error("Failed to convert spread to big dec", zap.Error(err))
		return nil, err
	}

	buyTick, lowTick, sellTick, highTick, err := calculateBuySellTicks(l, spotPriceAsBigDec, targetPriceAsBigDec, spreadAsBigDec)
	if err != nil {
		l.Error("Failed to calculate buy and sell ticks", zap.Error(err))
		return nil, err
	}

	buyPosition := createPositionMsg(poolId, lowTick, buyTick, sdk.NewCoins(token0), addr)
	sellPosition := createPositionMsg(poolId, sellTick, highTick, sdk.NewCoins(token1), addr)

	return []sdk.Msg{buyPosition, sellPosition}, nil
}

func calculateBuySellTicks(l *zap.Logger, buyPrice, sellPrice, spread osmomath.BigDec) (int64, int64, int64, int64, error) {
	// get the lower and upper bounds
	buyLowerBound := buyPrice.Mul(osmomath.OneBigDec().Sub(spread))
	sellUpperBound := sellPrice.Mul(osmomath.OneBigDec().Add(spread))

	// Calculate the buy and sell ticks
	buyPriceTick, err := calculateAndRoundPriceToTick(buyPrice)
	if err != nil {
		l.Error("Failed to calculate buy price tick", zap.Error(err))
	}

	buyLowerTick, err := calculateAndRoundPriceToTick(buyLowerBound)
	if err != nil {
		l.Error("Failed to calculate buy lower bound price tick", zap.Error(err))
	}

	sellPriceTick, err := calculateAndRoundPriceToTick(sellPrice)
	if err != nil {
		l.Error("Failed to calculate sell price tick", zap.Error(err))
	}

	sellUpperTick, err := calculateAndRoundPriceToTick(sellUpperBound)
	if err != nil {
		l.Error("Failed to calculate sell upper bound price tick", zap.Error(err))
	}

	return buyPriceTick, buyLowerTick, sellPriceTick, sellUpperTick, nil

}

func calculateAndRoundPriceToTick(price osmomath.BigDec) (int64, error) {
	priceTick, err := clmath.CalculatePriceToTick(price)
	if err != nil {
		return 0, err
	}

	priceTick, err = clmath.RoundDownTickToSpacing(priceTick, TICK_SPACING)
	if err != nil {
		return 0, err
	}

	return priceTick, nil
}
