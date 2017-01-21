package module

func init() {
	Register("FilteredFBComb", func(Config) (Patcher, error) { return NewFilteredFBComb(Duration(10000)) })
}

type FilteredFBComb struct {
	IO
	in, duration, gain, cutoff, resonance *In

	line   *DelayLine
	filter *FourPole
	last   Value
}

func NewFilteredFBComb(size MS) (*FilteredFBComb, error) {
	m := &FilteredFBComb{
		in:        &In{Name: "input", Source: zero},
		duration:  &In{Name: "duration", Source: NewBuffer(Duration(1000))},
		gain:      &In{Name: "gain", Source: NewBuffer(Value(0.98))},
		cutoff:    &In{Name: "cutoff", Source: NewBuffer(Frequency(1000))},
		resonance: &In{Name: "resonance", Source: NewBuffer(zero)},

		filter: &FourPole{kind: LowPass},
		line:   NewDelayLine(size),
	}
	err := m.Expose(
		[]*In{m.in, m.duration, m.gain, m.cutoff, m.resonance},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *FilteredFBComb) Read(out Frame) {
	reader.in.Read(out)
	gain := reader.gain.ReadFrame()
	duration := reader.duration.ReadFrame()
	cutoff := reader.cutoff.ReadFrame()
	resonance := reader.resonance.ReadFrame()
	for i := range out {
		out[i] += reader.last
		reader.filter.cutoff = cutoff[i]
		reader.filter.resonance = resonance[i]
		reader.last = gain[i] * reader.filter.Tick(reader.line.TickDuration(out[i], duration[i]))
	}
}

type DelayLine struct {
	buffer       Frame
	size, offset int
}

func NewDelayLine(size MS) *DelayLine {
	v := int(size.Value())
	return &DelayLine{
		size:   v,
		buffer: make(Frame, v),
	}
}

func (d *DelayLine) TickDuration(v, duration Value) Value {
	if d.offset >= int(duration) || d.offset >= d.size {
		d.offset = 0
	}
	v, d.buffer[d.offset] = d.buffer[d.offset], v
	d.offset++
	return v
}

func (d *DelayLine) Tick(v Value) Value {
	return d.TickDuration(v, 1)
}

func (reader *DelayLine) Read(out Frame) {
	for i := range out {
		out[i] = reader.Tick(out[i])
	}
}
