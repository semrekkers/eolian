package dsp

func Lerp(in, min, max Float64) Float64 {
	return in*(max-min) + min
}
