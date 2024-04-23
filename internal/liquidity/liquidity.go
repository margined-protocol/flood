package liquidity

import (
	"errors"
	"fmt"

	"github.com/osmosis-labs/osmosis/osmomath"
	clmath "github.com/osmosis-labs/osmosis/v21/x/concentrated-liquidity/math"
	"github.com/osmosis-labs/osmosis/v21/x/concentrated-liquidity/model"
	cltypes "github.com/osmosis-labs/osmosis/v21/x/concentrated-liquidity/types"
	"go.uber.org/zap"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const TickSpacing = int64(100)

// createPositionMsg creates a new CL position message
// createPositionMsg creates a new CL position message
func createPositionMsg(poolID uint64, lowerTick, upperTick int64, tokens sdk.Coins, addr string) sdk.Msg {
	// Generate the swap message
	msg := cltypes.MsgCreatePosition{
		PoolId:         poolID,
		Sender:         addr,
		LowerTick:      lowerTick,
		UpperTick:      upperTick,
		TokensProvided: tokens,
		// Note: we do not place token min amount as currently we only supply
		// single sided liquidity, ergo we do not to consider and slippage during
		// creation of the position
		TokenMinAmount0: sdk.ZeroInt(),
		TokenMinAmount1: sdk.ZeroInt(),
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
func MarketMake(l *zap.Logger, poolID uint64, currentTick int64, spotPrice, targetPrice, spread string, token0 sdk.Coin, token1 sdk.Coin, addr string) ([]sdk.Msg, error) {
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

	if spotPriceAsBigDec.LT(targetPriceAsBigDec) {
		targetPriceAsBigDec, spotPriceAsBigDec = spotPriceAsBigDec, targetPriceAsBigDec
	}

	buyTick, lowTick, sellTick, highTick, err := calculateBuySellTicks(l, targetPriceAsBigDec, spotPriceAsBigDec, spreadAsBigDec)
	if err != nil {
		l.Error("Failed to calculate buy and sell ticks", zap.Error(err))
		return nil, err
	}

	lowTick, buyTick = adjustForCurrentTick(l, true, currentTick, lowTick, buyTick)
	sellTick, highTick = adjustForCurrentTick(l, false, currentTick, sellTick, highTick)

	if !(lowTick < buyTick && buyTick < sellTick && sellTick < highTick) {
		err := errors.New("ticks are in the incorrect order")
		l.Error("Failed to calculate buy and sell ticks", zap.Error(err))
		return nil, err
	}

	buyPosition := createPositionMsg(poolID, lowTick, buyTick, sdk.NewCoins(token1), addr)
	sellPosition := createPositionMsg(poolID, sellTick, highTick, sdk.NewCoins(token0), addr)

	fmt.Println("buyPosition", buyPosition)
	fmt.Println("sellPosition", sellPosition)

	return []sdk.Msg{buyPosition, sellPosition}, nil
}

func adjustForCurrentTick(l *zap.Logger, isBuy bool, currentTick, lowerTick, upperTick int64) (int64, int64) {
	fmt.Println("lowerTick", lowerTick)

	if lowerTick <= currentTick && currentTick <= upperTick {
		fmt.Println("The value is within the range.")

		if isBuy {
			upperTick = currentTick - TickSpacing
		} else {
			lowerTick = currentTick + TickSpacing
		}
	}

	fmt.Println("lowerTick", lowerTick)

	upperTick, err := clmath.RoundDownTickToSpacing(upperTick, TickSpacing)
	if err != nil {
		l.Error("Failed to calculate buy price tick", zap.Error(err))
	}

	lowerTick, err = clmath.RoundDownTickToSpacing(lowerTick, TickSpacing)
	if err != nil {
		l.Error("Failed to calculate buy price tick", zap.Error(err))
	}

	// Calculate the absolute difference
	lowerDelta := currentTick - lowerTick
	if lowerDelta < 0 {
		lowerDelta = -lowerDelta
	}

	// Check if lowerDelta is less than TICK_SPACING and adjust lowerTick if necessary
	if lowerDelta < TickSpacing {
		lowerTick += (3 * TickSpacing)
	}

	return lowerTick, upperTick
}

// calculateBuySellTicks calculates the buy and sell ticks based on the spread
func calculateBuySellTicks(l *zap.Logger, buyPrice, sellPrice, spread osmomath.BigDec) (int64, int64, int64, int64, error) {
	fmt.Println(buyPrice, sellPrice, spread)
	// get the lower and upper bounds
	buyLowerBound := buyPrice.Mul(osmomath.OneBigDec().Sub(spread))
	sellUpperBound := sellPrice.Mul(osmomath.OneBigDec().Add(spread))

	// Calculate the buy and sell ticks
	buyPriceTick, err := calculateAndRoundPriceToTick(buyPrice)
	if err != nil {
		l.Error("Failed to calculate buy price tick", zap.Error(err))
		return 0, 0, 0, 0, err
	}

	buyLowerTick, err := calculateAndRoundPriceToTick(buyLowerBound)
	if err != nil {
		l.Error("Failed to calculate buy lower bound price tick", zap.Error(err))
		return 0, 0, 0, 0, err
	}

	sellPriceTick, err := calculateAndRoundPriceToTick(sellPrice)
	if err != nil {
		l.Error("Failed to calculate sell price tick", zap.Error(err))
		return 0, 0, 0, 0, err
	}

	sellUpperTick, err := calculateAndRoundPriceToTick(sellUpperBound)
	if err != nil {
		l.Error("Failed to calculate sell upper bound price tick", zap.Error(err))
		return 0, 0, 0, 0, err
	}

	l.Info("Market Bounds for transaction",
		zap.String("buy_lower_bound_price", buyLowerBound.String()),
		zap.Float64("buy_lower_bound_tick", float64(buyLowerTick)),
		zap.String("buy_upper_bound_price", buyPrice.String()),
		zap.Float64("buy_upper_bound_tick", float64(buyPriceTick)),
		zap.String("sell_lower_bound_price", sellPrice.String()),
		zap.Float64("sell_lower_bound_tick", float64(sellPriceTick)),
		zap.String("sell_upper_bound_price", sellUpperBound.String()),
		zap.Float64("sell_upper_bound_tick", float64(sellUpperTick)))

	return buyPriceTick, buyLowerTick, sellPriceTick, sellUpperTick, nil
}

func calculateAndRoundPriceToTick(price osmomath.BigDec) (int64, error) {
	priceTick, err := clmath.CalculatePriceToTick(price)
	if err != nil {
		return 0, err
	}

	priceTick, err = clmath.RoundDownTickToSpacing(priceTick, TickSpacing)
	if err != nil {
		return 0, err
	}

	return priceTick, nil
}
