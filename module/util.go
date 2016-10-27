package module

import (
	"math"
	"math/rand"
)

var epsilon = math.Nextafter(1, 2) - 1

func clampValue(s, min, max Value) Value {
	if s > max {
		s = max
	} else if s < min {
		s = min
	}
	return s
}

func randValue() Value {
	return Value(rand.Float64())
}

func absValue(v Value) Value {
	return Value(math.Abs(float64(v)))
}

func exp(v Value) Value {
	return Value(math.Exp(float64(v)))
}
