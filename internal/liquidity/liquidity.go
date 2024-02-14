package liquidity

import (
	"github.com/osmosis-labs/osmosis/osmomath"
	clmath "github.com/osmosis-labs/osmosis/v21/x/concentrated-liquidity/math"
	"go.uber.org/zap"
)

func market_make(l *zap.Logger, spotPrice, targetPrice string) {
	l.Debug("inputs",
		zap.String("spotPrice", spotPrice),
		zap.String("targetPrice", targetPrice),
	)

	spotPriceAsBigDec, err := osmomath.NewBigDecFromStr(spotPrice)
	if err != nil {
		l.Error("Failed to convert spot price to big dec", zap.Error(err))
	}

	targetPriceAsBigDec, err := osmomath.NewBigDecFromStr(targetPrice)
	if err != nil {
		l.Error("Failed to convert spot price to big dec", zap.Error(err))
		// return osmomath.OneBigDec(), err
	}

	l.Debug("inputs as big dec",
		zap.Reflect("spotPriceAsBigDec", spotPriceAsBigDec),
		zap.Reflect("targetPriceAsBigDec", targetPriceAsBigDec),
	)

	spotPriceTick, err := clmath.CalculatePriceToTick(spotPriceAsBigDec)
	if err != nil {
		l.Error("Failed to calculate start price tick", zap.Error(err))
		// return osmomath.OneBigDec(), err
	}

	targetPriceTick, err := clmath.CalculatePriceToTick(targetPriceAsBigDec)
	if err != nil {
		l.Error("Failed to calculate target price tick", zap.Error(err))
		// return osmomath.OneBigDec(), err
	}

	l.Debug("outputs",
		zap.Int64("spotPriceTick", spotPriceTick),
		zap.Int64("targetPriceTick", targetPriceTick),
	)
}

func calculate_buy_sell_ticks(l *zap.Logger, buyPrice, sellPrice, spread osmomath.BigDec) (int64, int64, int64, int64) {
	// get the lower and upper bounds
	buyLowerBound := buyPrice.Mul(osmomath.OneBigDec().Sub(spread))
	sellUpperBound := sellPrice.Mul(osmomath.OneBigDec().Add(spread))

	// Calculate the buy and sell ticks
	buyPriceTick, err := clmath.CalculatePriceToTick(buyPrice)
	if err != nil {
		l.Error("Failed to calculate buy price tick", zap.Error(err))
	}

	buyLowerTick, err := clmath.CalculatePriceToTick(buyLowerBound)
	if err != nil {
		l.Error("Failed to calculate buy lower bound price tick", zap.Error(err))
	}

	sellPriceTick, err := clmath.CalculatePriceToTick(sellPrice)
	if err != nil {
		l.Error("Failed to calculate sell price tick", zap.Error(err))
	}

	sellUpperTick, err := clmath.CalculatePriceToTick(sellUpperBound)
	if err != nil {
		l.Error("Failed to calculate sell upper bound price tick", zap.Error(err))
	}

	l.Debug("outputs",
		zap.Int64("buyPriceTick", buyPriceTick),
		zap.Int64("buyLowerTick", buyLowerTick),
		zap.Int64("sellPriceTick", sellPriceTick),
		zap.Int64("sellUpperTick", sellUpperTick),
	)

	return buyPriceTick, buyLowerTick, sellPriceTick, sellUpperTick

}
