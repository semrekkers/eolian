package dsp

// DelayLine is a simple delay line
type DelayLine struct {
	buffer       Frame
	sizeMS       MS
	size, offset int
}

// NewDelayLine returns a new DelayLine of a specific maximum size in milliseconds
func NewDelayLine(size int) *DelayLine {
	return &DelayLine{
		size:   size,
		buffer: make(Frame, size),
	}
}

// NewDelayLineMS returns a new DelayLine of a specific maximum size in milliseconds
func NewDelayLineMS(size MS) *DelayLine {
	v := int(size.Value())
	return &DelayLine{
		size:   v,
		buffer: make(Frame, v),
	}
}

// Tick advances the operation using the full delay line size as duration
func (d *DelayLine) Tick(v Float64) Float64 {
	return d.TickDuration(v, Float64(d.size))
}

// TickDuration advances the operation with a specific length in samples. The length must be less than or equal to the
// total length of the delay line.
func (d *DelayLine) TickDuration(v, duration Float64) Float64 {
	if d.offset >= int(duration) || d.offset >= d.size {
		d.offset = 0
	}
	v, d.buffer[d.offset] = d.buffer[d.offset], v
	d.offset++
	return v
}
