package dsp

// NewAllPassMS returns a new AllPass
func NewAllPassMS(ms MS) *AllPass {
	return &AllPass{dl: NewDelayLineMS(ms)}
}

// NewAllPass returns a new AllPass
func NewAllPass(size int) *AllPass {
	return &AllPass{dl: NewDelayLine(size)}
}

// AllPass is an allpass filter
type AllPass struct {
	dl   *DelayLine
	last Float64
}

// Tick advances the filter's operation with the default duration
func (a *AllPass) Tick(in, gain Float64) Float64 {
	return a.TickDuration(in, gain, -1)
}

// TickDuration advances the filter's operation
func (a *AllPass) TickDuration(in, gain, duration Float64) Float64 {
	before := in + -gain*a.last
	a.last = tick(a.dl, before, duration)
	return a.last + gain*before
}
