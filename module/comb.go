package module

import "github.com/mitchellh/mapstructure"

func init() {
	setup := func(f func(s MS, c Config) (Patcher, error)) func(Config) (Patcher, error) {
		return func(c Config) (Patcher, error) {
			var config struct {
				Size int
			}
			if err := mapstructure.Decode(c, &config); err != nil {
				return nil, err
			}
			if config.Size == 0 {
				config.Size = 10000
			}
			return f(DurationInt(config.Size), c)
		}
	}

	Register("FFComb", setup(func(s MS, c Config) (Patcher, error) { return NewFFComb(s) }))
	Register("FBComb", setup(func(s MS, c Config) (Patcher, error) { return NewFBComb(s) }))
	Register("Allpass", setup(func(s MS, c Config) (Patcher, error) { return NewAllPass(s) }))
	Register("FilteredFBComb", setup(func(s MS, c Config) (Patcher, error) { return NewFilteredFBComb(s) }))
}

type FFComb struct {
	IO
	in, duration, gain *In

	Config
	line *DelayLine
}

func NewFFComb(size MS) (*FFComb, error) {
	m := &FFComb{
		in:       &In{Name: "input", Source: zero},
		duration: &In{Name: "duration", Source: NewBuffer(Duration(1000))},
		gain:     &In{Name: "gain", Source: NewBuffer(Value(0.9))},
		line:     NewDelayLine(size),
	}
	err := m.Expose(
		[]*In{m.in, m.duration, m.gain},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (c *FFComb) Read(out Frame) {
	c.in.Read(out)
	gain := c.gain.ReadFrame()
	duration := c.duration.ReadFrame()
	for i := range out {
		out[i] += gain[i] * c.line.TickDuration(out[i], duration[i])
	}
}

type FBComb struct {
	IO
	in, duration, gain *In

	line *DelayLine
	last Value
}

func NewFBComb(size MS) (*FBComb, error) {
	m := &FBComb{
		in:       &In{Name: "input", Source: zero},
		duration: &In{Name: "duration", Source: NewBuffer(Duration(1000))},
		gain:     &In{Name: "gain", Source: NewBuffer(Value(0.9))},
		line:     NewDelayLine(size),
	}
	err := m.Expose(
		[]*In{m.in, m.duration, m.gain},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (c *FBComb) Read(out Frame) {
	c.in.Read(out)
	gain := c.gain.ReadFrame()
	duration := c.duration.ReadFrame()
	for i := range out {
		out[i] += c.last
		c.last = gain[i] * c.line.TickDuration(out[i], duration[i])
	}
}

type AllPass struct {
	IO
	in, duration, gain *In

	line *DelayLine
	last Value
}

func NewAllPass(size MS) (*AllPass, error) {
	m := &AllPass{
		in:       &In{Name: "input", Source: zero},
		duration: &In{Name: "duration", Source: NewBuffer(Duration(1000))},
		gain:     &In{Name: "gain", Source: NewBuffer(Value(0.9))},
		line:     NewDelayLine(size),
	}
	err := m.Expose(
		[]*In{m.in, m.duration, m.gain},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (p *AllPass) Read(out Frame) {
	p.in.Read(out)
	gain := p.gain.ReadFrame()
	duration := p.duration.ReadFrame()
	for i := range out {
		gain := gain[i]
		in := out[i]
		before := in + -gain*p.last
		p.last = p.line.TickDuration(before, duration[i])
		out[i] = p.last + gain*before
	}
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

func (c *FilteredFBComb) Read(out Frame) {
	c.in.Read(out)
	gain := c.gain.ReadFrame()
	duration := c.duration.ReadFrame()
	cutoff := c.cutoff.ReadFrame()
	resonance := c.resonance.ReadFrame()
	for i := range out {
		out[i] += c.last
		c.filter.cutoff = cutoff[i]
		c.filter.resonance = resonance[i]
		c.last = gain[i] * c.filter.Tick(c.line.TickDuration(out[i], duration[i]))
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

func (l *DelayLine) Read(out Frame) {
	for i := range out {
		out[i] = l.Tick(out[i])
	}
}
