package dsp

import "math"

type Follow struct {
	Rise, Fall Float64
	env        Float64
}

func (f *Follow) Tick(in Float64) Float64 {
	in = Abs(in)
	if in == f.env {
		return f.env
	}
	slope := f.Fall
	if in > f.env {
		slope = f.Rise
	}
	f.env = Float64(math.Pow(0.01, float64(1.0/slope)))*(f.env-in) + in
	return f.env
}
