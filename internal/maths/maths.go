package maths

import (
	"fmt"
	"math/big"
	"strconv"
)

// CalculateMarkPrice calculates the mark price based on base price, power price,
// normalization factor, and scale factor. The function returns the calculated mark price as a float64.
// It handles the conversion of string inputs to big.Float for precise calculations, especially
// important for financial computations. If any of the string inputs are invalid or if there's an
// inexact conversion to float64, the function returns an error.
//
// Parameters:
// - basePrice: The base asset price as a string.
// - powerPrice: The power asset price as a string.
// - normalizationFactor: A string representing the normalization factor to adjust the price.
// - scaleFactor: An integer representing the scale factor to be applied.
//
// Returns:
// - The calculated mark price as float64.
// - An error if there is an issue with input parsing or inexact float conversion.
func CalculateMarkPrice(basePrice, powerPrice, normalizationFactor string, scaleFactor int) (float64, error) {
	if normalizationFactor == "" {
		return 0, fmt.Errorf("normalization factor is empty")
	}

	// Convert strings to big.Float
	basePriceBig, ok := big.NewFloat(0).SetString(basePrice)
	if !ok {
		return 0, fmt.Errorf("invalid base price: %s", basePrice)
	}
	powerPriceBig, ok := big.NewFloat(0).SetString(powerPrice)
	if !ok {
		return 0, fmt.Errorf("invalid power price: %s", powerPrice)
	}
	normalizationFactorBig, ok := big.NewFloat(0).SetString(normalizationFactor)
	if !ok {
		return 0, fmt.Errorf("invalid normalization factor: %s", normalizationFactor)
	}

	scaleFactorBig := big.NewFloat(float64(scaleFactor))

	// Perform the calculation: (basePrice / powerPrice / normalizationFactor) * scaleFactor
	normalizedPrice := new(big.Float).Quo(basePriceBig, powerPriceBig)
	normalizedPrice.Quo(normalizedPrice, normalizationFactorBig)
	normalizedPrice.Mul(normalizedPrice, scaleFactorBig)

	// Convert big.Float to float64
	result, accuracy := normalizedPrice.Float64()
	if accuracy == big.Exact {
		return result, nil
	}
	return result, fmt.Errorf("inexact conversion to float64")
}

func CalculateTargetPrice(basePrice, normalizationFactor string, scaleFactor int) (float64, error) {
	if normalizationFactor == "" {
		return 0, fmt.Errorf("normalization factor is empty")
	}

	// Convert strings to big.Float
	basePriceBig, ok := big.NewFloat(0).SetString(basePrice)
	if !ok {
		return 0, fmt.Errorf("invalid base price: %s", basePrice)
	}
	normalizationFactorBig, ok := big.NewFloat(0).SetString(normalizationFactor)
	if !ok {
		return 0, fmt.Errorf("invalid normalization factor: %s", normalizationFactor)
	}

	scaleFactorBig := big.NewFloat(float64(scaleFactor))

	// Perform the calculation: (basePrice * scaleFactor) / (basePrice^2 * normalizationFactor)
	basePriceSquared := new(big.Float).Mul(basePriceBig, basePriceBig)
	numerator := new(big.Float).Mul(basePriceBig, scaleFactorBig)
	denominator := new(big.Float).Mul(basePriceSquared, normalizationFactorBig)

	targetPrice := new(big.Float).Quo(numerator, denominator)

	// Convert big.Float to float64
	result, accuracy := targetPrice.Float64()
	if accuracy == big.Exact {
		return result, nil
	}
	return result, fmt.Errorf("inexact conversion to float64")
}

// calculatePremium computes the premium based on markPrice and indexPrice.
func CalculatePremium(markPrice, indexPrice float64) float64 {
	if indexPrice == 0 {
		return 0
	}
	premium := ((markPrice - indexPrice) / indexPrice)
	return premium
}

func CalculateIndexPrice(baseSpotPrice string) (float64, error) {
	sp, err := strconv.ParseFloat(baseSpotPrice, 64)
	if err != nil {
		return 0, err
	}

	return sp * sp, nil
}
