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

type TappedDelayLine struct {
	dl   []*DelayLine
	taps []Float64
}

func NewTappedDelayLine(taps []int) *TappedDelayLine {
	dl := &TappedDelayLine{
		dl:   make([]*DelayLine, len(taps)),
		taps: make([]Float64, len(taps)),
	}
	for i, t := range taps {
		dl.dl[i] = NewDelayLine(t)
	}
	return dl
}

func (d *TappedDelayLine) TapCount() int {
	return len(d.taps)
}

// Tick advances the operation using the full delay line size as duration
func (d *TappedDelayLine) Tick(v Float64) []Float64 {
	dv := v
	for i, dl := range d.dl {
		dv = dl.Tick(dv)
		d.taps[i] = dv
	}
	return d.taps
}
