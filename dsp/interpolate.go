package dsp

func Lerp(in, min, max Float64) Float64 {
	return in*(max-min) + min
}

type RollingAverage struct {
	Window int
	value  Float64
}

func (a *RollingAverage) Tick(in Float64) Float64 {
	a.value -= a.value / Float64(a.Window)
	a.value += in / Float64(a.Window)
	return a.value
}
