package module

func init() {
	Register("FilteredDelay", func(Config) (Patcher, error) { return NewFilteredDelay(defaultDelay) })
}

const defaultDelay = 10000

type FilteredDelay struct {
	IO
	in, duration, gain, cutoff, resonance *In

	line   *DelayLine
	filter *FourPole
	last   Value
}

func NewFilteredDelay(size int) (*FilteredDelay, error) {
	m := &FilteredDelay{
		in:        &In{Name: "input", Source: zero},
		duration:  &In{Name: "duration", Source: NewBuffer(Value(0.01))},
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

func (reader *FilteredDelay) Read(out Frame) {
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

func NewDelayLine(size int) *DelayLine {
	return &DelayLine{
		size:   size,
		buffer: make(Frame, size),
	}
}

func (d *DelayLine) TickDuration(v, duration Value) Value {
	if d.offset >= int(float64(duration)*float64(d.size)) || d.offset >= d.size {
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
