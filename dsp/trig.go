package dsp

import "math"

const (
	sineLength = 1024
	sineStep   = sineLength / (2 * math.Pi)
)

var sineTable, sineDiff []Float64

func init() {
	sineTable = make([]Float64, sineLength)
	sineDiff = make([]Float64, sineLength)

	for i := 0; i < sineLength; i++ {
		sineTable[i] = Float64(math.Sin(float64(i) * (1 / sineStep)))
	}
	for i := 0; i < sineLength; i++ {
		next := sineTable[(i+1)%sineLength]
		sineDiff[i] = Float64(next - sineTable[i])
	}
}

// Sin is a lookup table version of math.Sin
func Sin(x Float64) Float64 {
	step := x * sineStep
	if x < 0 {
		step = -step
	}

	trunc := int(step)
	i := trunc % sineLength
	out := sineTable[i] + sineDiff[i]*(step-Float64(trunc))

	if x < 0 {
		return -out
	}
	return out
}

// Tan is a lookup table version of math.Tan
func Tan(x Float64) Float64 {
	return Sin(x) / Sin(x+0.5*math.Pi)
}
