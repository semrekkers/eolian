package module

import (
	"github.com/mitchellh/mapstructure"
)

func init() {
	setup := func(f func(s MS, c Config) (Patcher, error)) func(Config) (Patcher, error) {
		return func(c Config) (Patcher, error) {
			var config struct{ Size int }
			if err := mapstructure.Decode(c, &config); err != nil {
				return nil, err
			}
			if config.Size == 0 {
				config.Size = 10000
			}
			return f(DurationInt(config.Size), c)
		}
	}

	Register("FFDelay", setup(func(s MS, c Config) (Patcher, error) { return newFFComb(s) }))
	Register("FBDelay", setup(func(s MS, c Config) (Patcher, error) { return newFBDelay(s) }))
	Register("Allpass", setup(func(s MS, c Config) (Patcher, error) { return newAllpass(s) }))
	Register("FilteredFBDelay", setup(func(s MS, c Config) (Patcher, error) { return newFilteredFBDelay(s) }))
	Register("FBLoopDelay", setup(func(s MS, c Config) (Patcher, error) { return newFBLoopDelay(s) }))
}

type ffDelay struct {
	IO
	in, duration, gain *In

	Config
	line *delayline
}

func newFFComb(size MS) (*ffDelay, error) {
	m := &ffDelay{
		in:       &In{Name: "input", Source: zero},
		duration: &In{Name: "duration", Source: NewBuffer(Duration(1000))},
		gain:     &In{Name: "gain", Source: NewBuffer(Value(0.9))},
		line:     newDelayLine(size),
	}
	err := m.Expose(
		"FFComb",
		[]*In{m.in, m.duration, m.gain},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (c *ffDelay) Read(out Frame) {
	c.in.Read(out)
	gain := c.gain.ReadFrame()
	duration := c.duration.ReadFrame()
	for i := range out {
		out[i] += gain[i] * c.line.TickDuration(out[i], duration[i])
	}
}

type fbDelay struct {
	IO
	in, duration, gain *In

	line *delayline
	last Value
}

func newFBDelay(size MS) (*fbDelay, error) {
	m := &fbDelay{
		in:       &In{Name: "input", Source: zero},
		duration: &In{Name: "duration", Source: NewBuffer(Duration(1000))},
		gain:     &In{Name: "gain", Source: NewBuffer(Value(0.9))},
		line:     newDelayLine(size),
	}
	err := m.Expose(
		"FBComb",
		[]*In{m.in, m.duration, m.gain},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (c *fbDelay) Read(out Frame) {
	c.in.Read(out)
	gain := c.gain.ReadFrame()
	duration := c.duration.ReadFrame()
	for i := range out {
		out[i] += c.last
		c.last = gain[i] * c.line.TickDuration(out[i], duration[i])
	}
}

type allpass struct {
	IO
	in, duration, gain *In

	line *delayline
	last Value
}

func newAllpass(size MS) (*allpass, error) {
	m := &allpass{
		in:       &In{Name: "input", Source: zero},
		duration: &In{Name: "duration", Source: NewBuffer(Duration(1000))},
		gain:     &In{Name: "gain", Source: NewBuffer(Value(0.9))},
		line:     newDelayLine(size),
	}
	err := m.Expose(
		"Allpass",
		[]*In{m.in, m.duration, m.gain},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (p *allpass) Read(out Frame) {
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

type filteredFBDelay struct {
	IO
	in, duration, gain, cutoff, resonance *In

	line   *delayline
	filter *filter
	last   Value
}

func newFilteredFBDelay(size MS) (*filteredFBDelay, error) {
	m := &filteredFBDelay{
		in:        &In{Name: "input", Source: zero},
		duration:  &In{Name: "duration", Source: NewBuffer(Duration(1000))},
		gain:      &In{Name: "gain", Source: NewBuffer(Value(0.98))},
		cutoff:    &In{Name: "cutoff", Source: NewBuffer(Frequency(1000))},
		resonance: &In{Name: "resonance", Source: NewBuffer(Value(1))},

		filter: &filter{poles: 4},
		line:   newDelayLine(size),
	}
	err := m.Expose(
		"FilteredFBComb",
		[]*In{m.in, m.duration, m.gain, m.cutoff, m.resonance},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (c *filteredFBDelay) Read(out Frame) {
	c.in.Read(out)
	gain := c.gain.ReadFrame()
	duration := c.duration.ReadFrame()
	cutoff := c.cutoff.ReadFrame()
	resonance := c.resonance.ReadFrame()
	for i := range out {
		out[i] += c.last
		c.filter.cutoff = cutoff[i]
		c.filter.resonance = resonance[i]
		lp, _, _ := c.filter.Tick(c.line.TickDuration(out[i], duration[i]))
		c.last = gain[i] * lp
	}
}

type fbLoopDelay struct {
	IO
	in, duration, gain, feedbackReturn *In
	feedbackSend                       *Out

	sent Frame
	line *delayline
	last Value
}

func newFBLoopDelay(size MS) (*fbLoopDelay, error) {
	m := &fbLoopDelay{
		in:       &In{Name: "input", Source: zero},
		duration: &In{Name: "duration", Source: NewBuffer(Duration(1000))},
		gain:     &In{Name: "gain", Source: NewBuffer(Value(0.98))},
		feedbackReturn: &In{
			Name:         "feedbackReturn",
			Source:       NewBuffer(zero),
			ForceSinking: true,
		},

		sent: make(Frame, FrameSize),
		line: newDelayLine(size),
	}
	m.feedbackSend = &Out{Name: "feedbackSend", Provider: Provide(&loopDelaySend{m})}

	err := m.Expose(
		"FBLoopComb",
		[]*In{m.in, m.duration, m.gain, m.feedbackReturn},
		[]*Out{
			m.feedbackSend,
			{Name: "output", Provider: Provide(m)},
		},
	)
	return m, err
}

func (c *fbLoopDelay) Read(out Frame) {
	c.in.Read(out)
	gain := c.gain.ReadFrame()
	duration := c.duration.ReadFrame()
	if c.feedbackSend.IsActive() {
		c.feedbackReturn.ReadFrame()
	}
	for i := range out {
		out[i] += c.last
		c.sent[i] = c.line.TickDuration(out[i], duration[i])
		if c.feedbackSend.IsActive() {
			c.last = gain[i] * c.feedbackReturn.LastFrame()[i]
		} else {
			c.last = gain[i] * c.sent[i]
		}
	}
}

type loopDelaySend struct {
	*fbLoopDelay
}

func (c *loopDelaySend) Read(out Frame) {
	for i := range out {
		out[i] = c.sent[i]
	}
}
