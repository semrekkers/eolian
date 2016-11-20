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

func expRatio(ratio, speed Value) Value {
	return Value(math.Exp(-math.Log(float64((1+ratio)/ratio)) / float64(speed)))
}

func minInt(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func maxInt(x, y int) int {
	if x > y {
		return x
	}
	return y
}
