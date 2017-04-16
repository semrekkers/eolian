package dsp

import (
	"math"
	"math/rand"
)

// Clamp limits a value to a specific range
func Clamp(s, min, max Float64) Float64 {
	if s > max {
		s = max
	} else if s < min {
		s = min
	}
	return s
}

// Tan is math.Tan()
func Tan(v Float64) Float64 {
	return Float64(math.Tan(float64(v)))
}

// Rand is rand.Float64()
func Rand() Float64 {
	return Float64(rand.Float64())
}

// RandRange returns random values between a specified range
func RandRange(min, max Float64) Float64 {
	return Rand()*(max-min) + min
}

// Abs is math.Abs()
func Abs(v Float64) Float64 {
	return Float64(math.Abs(float64(v)))
}

// Floor is math.Floor()
func Floor(v Float64) Float64 {
	return Float64(math.Floor(float64(v)))
}

// Ceil is math.Ceil()
func Ceil(v Float64) Float64 {
	return Float64(math.Ceil(float64(v)))
}

// Max is math.Max()
func Max(a, b Float64) Float64 {
	return Float64(math.Max(float64(a), float64(b)))
}

// ExpRatio produces an (inverse-)exponential curve that's inflection can be controlled by a specific ratio
func ExpRatio(ratio, speed Float64) Float64 {
	return Float64(math.Exp(-math.Log(float64((1+ratio)/ratio)) / float64(speed)))
}

// AttenSum adds two inputs together, but allows you to attenuate either (mutually exclusive) with a bias input. -1 will
// be 100% signal a and none of signal b. 1 will be none of signal a and 100% signal b. 0 they will be equal (100%).
func AttenSum(bias, a, b Float64) Float64 {
	if bias > 0 {
		return (1-bias)*a + b
	} else if bias < 0 {
		return a + (1+bias)*b
	}
	return a + b
}
