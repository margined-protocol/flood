package liquidity

import (
	"testing"

	"github.com/osmosis-labs/osmosis/osmomath"
	"go.uber.org/zap"
	"gotest.tools/assert"
)

func TestCalculateBuySellTicksBasicFunctionality(t *testing.T) {
	logger, _ := zap.NewProduction()

	// Assuming osmomath.BigDec can be constructed from strings or similar for this example
	buyPrice, _ := osmomath.NewBigDecFromStr("1.0")
	sellPrice, _ := osmomath.NewBigDecFromStr("1.0")
	spread, _ := osmomath.NewBigDecFromStr("0.1") // 10% spread

	buyPriceTick, buyLowerTick, sellPriceTick, sellUpperTick, _ := calculateBuySellTicks(logger, buyPrice, sellPrice, spread)

	// Assertions
	assert.Equal(t, int64(0), buyPriceTick, "Buy price tick should match expected value")
	assert.Equal(t, int64(-1000000), buyLowerTick, "Buy lower tick should match expected value")
	assert.Equal(t, int64(0), sellPriceTick, "Sell price tick should match expected value")
	assert.Equal(t, int64(100000), sellUpperTick, "Sell upper tick should match expected value")
}

func TestCalculateBuySellTicksMidPriceLessThanOne(t *testing.T) {
	logger, _ := zap.NewProduction()

	// Assuming osmomath.BigDec can be constructed from strings or similar for this example
	buyPrice, _ := osmomath.NewBigDecFromStr("0.9")
	sellPrice, _ := osmomath.NewBigDecFromStr("1.0")
	spread, _ := osmomath.NewBigDecFromStr("0.1") // 10% spread

	buyPriceTick, buyLowerTick, sellPriceTick, sellUpperTick, _ := calculateBuySellTicks(logger, buyPrice, sellPrice, spread)

	// Assertions
	assert.Equal(t, int64(-1000000), buyPriceTick, "Buy price tick should match expected value")
	assert.Equal(t, int64(-1900000), buyLowerTick, "Buy lower tick should match expected value")
	assert.Equal(t, int64(0), sellPriceTick, "Sell price tick should match expected value")
	assert.Equal(t, int64(100000), sellUpperTick, "Sell upper tick should match expected value")
}

func TestCalculateBuySellTicksMidPriceGreaterThanOne(t *testing.T) {
	logger, _ := zap.NewProduction()

	// Assuming osmomath.BigDec can be constructed from strings or similar for this example
	buyPrice, _ := osmomath.NewBigDecFromStr("10.1")
	sellPrice, _ := osmomath.NewBigDecFromStr("10.3")
	spread, _ := osmomath.NewBigDecFromStr("0.1") // 10% spread

	buyPriceTick, buyLowerTick, sellPriceTick, sellUpperTick, _ := calculateBuySellTicks(logger, buyPrice, sellPrice, spread)

	// Assertions
	assert.Equal(t, int64(9010000), buyPriceTick, "Buy price tick should match expected value")
	assert.Equal(t, int64(8090000), buyLowerTick, "Buy lower tick should match expected value")
	assert.Equal(t, int64(9030000), sellPriceTick, "Sell price tick should match expected value")
	assert.Equal(t, int64(9133000), sellUpperTick, "Sell upper tick should match expected value")
}

func TestCalculateBuySellTicksValueUnderZero(t *testing.T) {
	logger, _ := zap.NewProduction()

	// Assuming osmomath.BigDec can be constructed from strings or similar for this example
	buyPrice, _ := osmomath.NewBigDecFromStr("0.17")
	sellPrice, _ := osmomath.NewBigDecFromStr("0.18")
	spread, _ := osmomath.NewBigDecFromStr("0.1") // 10% spread

	buyPriceTick, buyLowerTick, sellPriceTick, sellUpperTick, _ := calculateBuySellTicks(logger, buyPrice, sellPrice, spread)

	// Assertions
	assert.Equal(t, int64(-8300000), buyPriceTick, "Buy price tick should match expected value")
	assert.Equal(t, int64(-8470000), buyLowerTick, "Buy lower tick should match expected value")
	assert.Equal(t, int64(-8200000), sellPriceTick, "Sell price tick should match expected value")
	assert.Equal(t, int64(-8020000), sellUpperTick, "Sell upper tick should match expected value")
}
