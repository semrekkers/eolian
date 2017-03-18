package module

import (
	"math"
	"math/rand"
)

var (
	epsilon     = math.Nextafter(1, 2) - 1
	alphaSeries = "abcdefghijklmnopqrstuvwxyz"
)

func clampValue(s, min, max Value) Value {
	if s > max {
		s = max
	} else if s < min {
		s = min
	}
	return s
}

func tanValue(v Value) Value {
	return Value(math.Tan(float64(v)))
}

func randValue() Value {
	return Value(rand.Float64())
}

func absValue(v Value) Value {
	return Value(math.Abs(float64(v)))
}

func floorValue(v Value) Value {
	return Value(math.Floor(float64(v)))
}

func ceilValue(v Value) Value {
	return Value(math.Ceil(float64(v)))
}

func maxValue(v1, v2 Value) Value {
	return Value(math.Max(float64(v1), float64(v2)))
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
