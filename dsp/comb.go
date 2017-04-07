package dsp

// NewFBComb returns a new FBComb
func NewFBComb(ms MS) *FBComb {
	return &FBComb{dl: NewDelayLine(ms)}
}

// FBComb is a feedback comb filter
type FBComb struct {
	dl   *DelayLine
	last Float64
}

// Tick advances the filter's operation with the default duration
func (c *FBComb) Tick(in, gain Float64) Float64 {
	return c.TickDuration(in, gain, -1)
}

// TickDuration advances the filter's operation with a specific duration
func (c *FBComb) TickDuration(in, gain, duration Float64) Float64 {
	out := in + c.last
	c.last = gain * tick(c.dl, in, duration)
	return out
}

// NewFilteredFBComb returns a new FilteredFBComb
func NewFilteredFBComb(ms MS, poles int) *FilteredFBComb {
	return &FilteredFBComb{dl: NewDelayLine(ms), f: &SVFilter{Poles: poles}}
}

// FilteredFBComb is a feedback comb filter
type FilteredFBComb struct {
	dl   *DelayLine
	f    *SVFilter
	last Float64
}

// Tick advances the filter's operation with the default duration
func (c *FilteredFBComb) Tick(in, gain, cutoff, resonance Float64) Float64 {
	return c.TickDuration(in, gain, -1, cutoff, resonance)
}

// TickDuration advances the filter's operation with a specific duration
func (c *FilteredFBComb) TickDuration(in, gain, duration, cutoff, resonance Float64) Float64 {
	out := in + c.last
	c.f.Cutoff = cutoff
	c.f.Resonance = resonance
	lp, _, _ := c.f.Tick(tick(c.dl, out, duration))
	c.last = gain * lp

	return out
}

// NewFFComb returns a new FFComb
func NewFFComb(ms MS) *FFComb {
	return &FFComb{dl: NewDelayLine(ms)}
}

// FFComb is a feedforward comb filter
type FFComb struct {
	dl   *DelayLine
	last Float64
}

// Tick advances the filter's operation with the default duration
func (c *FFComb) Tick(in, gain Float64) Float64 {
	return c.TickDuration(in, gain, -1)
}

// TickDuration advances the filter's operation with a specific duration
func (c *FFComb) TickDuration(in, gain, duration Float64) Float64 {
	return in + gain*c.dl.TickDuration(in, duration)
}

// NewAllPass returns a new AllPass
func NewAllPass(ms MS) *AllPass {
	return &AllPass{dl: NewDelayLine(ms)}
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

// Tick advances the filter's operation
func (a *AllPass) TickDuration(in, gain, duration Float64) Float64 {
	before := in + -gain*a.last
	a.last = tick(a.dl, before, duration)
	return a.last + gain*before
}

func tick(dl *DelayLine, in, duration Float64) Float64 {
	if duration < 0 {
		return dl.Tick(in)
	}
	return dl.TickDuration(in, duration)
}
