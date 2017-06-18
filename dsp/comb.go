package dsp

// NewFBCombMS returns a new FBComb
func NewFBCombMS(ms MS) *FBComb {
	return &FBComb{dl: NewDelayLineMS(ms)}
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
func NewFilteredFBCombMS(ms MS, poles int) *FilteredFBComb {
	return &FilteredFBComb{dl: NewDelayLineMS(ms), f: NewFilter(LowPass, 4)}
}

// FilteredFBComb is a feedback comb filter
type FilteredFBComb struct {
	dl   *DelayLine
	f    *Filter
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
	c.last = gain * c.f.Tick(tick(c.dl, out, duration))

	return out
}

// NewFFComb returns a new FFComb
func NewFFCombMS(ms MS) *FFComb {
	return &FFComb{dl: NewDelayLineMS(ms)}
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

func tick(dl *DelayLine, in, duration Float64) Float64 {
	if duration < 0 {
		return dl.Tick(in)
	}
	return dl.TickDuration(in, duration)
}
