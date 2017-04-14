package module

import (
	"buddin.us/eolian/dsp"
	"github.com/mitchellh/mapstructure"
)

func init() {
	setup := func(f func(s dsp.MS, c Config) (Patcher, error)) func(Config) (Patcher, error) {
		return func(c Config) (Patcher, error) {
			var config struct{ Size int }
			if err := mapstructure.Decode(c, &config); err != nil {
				return nil, err
			}
			if config.Size == 0 {
				config.Size = 10000
			}
			return f(dsp.DurationInt(config.Size), c)
		}
	}

	Register("FBDelay", setup(func(s dsp.MS, c Config) (Patcher, error) { return newFBDelay(s) }))
	Register("Allpass", setup(func(s dsp.MS, c Config) (Patcher, error) { return newAllpass(s) }))
	Register("FilteredFBDelay", setup(func(s dsp.MS, c Config) (Patcher, error) { return newFilteredFBDelay(s) }))
	Register("FBLoopDelay", setup(func(s dsp.MS, c Config) (Patcher, error) { return newFBLoopDelay(s) }))
	Register("Delay", func(c Config) (Patcher, error) {
		var config struct{ Size int }
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Size == 0 {
			config.Size = 10000
		}
		return newDelay(dsp.DurationInt(config.Size))
	})
}

type delay struct {
	IO
	in, duration *In
	line         *dsp.DelayLine
}

func newDelay(size dsp.MS) (*delay, error) {
	m := &delay{
		in:       NewIn("input", dsp.Float64(0)),
		duration: NewInBuffer("duration", dsp.Duration(1000)),
		line:     dsp.NewDelayLineMS(size),
	}

	err := m.Expose(
		"Delay",
		[]*In{m.in, m.duration},
		[]*Out{
			{Name: "output", Provider: dsp.Provide(m)},
		},
	)
	return m, err
}

func (c *delay) Process(out dsp.Frame) {
	c.in.Process(out)
	duration := c.duration.ProcessFrame()
	for i := range out {
		out[i] = c.line.TickDuration(out[i], duration[i])
	}
}

type fbDelay struct {
	IO
	in, duration, gain *In
	comb               *dsp.FBComb
}

func newFBDelay(size dsp.MS) (*fbDelay, error) {
	m := &fbDelay{
		in:       NewIn("input", dsp.Float64(0)),
		duration: NewInBuffer("duration", dsp.Duration(1000)),
		gain:     NewInBuffer("gain", dsp.Float64(0.9)),
		comb:     dsp.NewFBCombMS(size),
	}
	err := m.Expose(
		"FBDelay",
		[]*In{m.in, m.duration, m.gain},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

func (c *fbDelay) Process(out dsp.Frame) {
	c.in.Process(out)
	gain := c.gain.ProcessFrame()
	duration := c.duration.ProcessFrame()
	for i := range out {
		out[i] = c.comb.TickDuration(out[i], gain[i], duration[i])
	}
}

type allpass struct {
	IO
	in, duration, gain *In
	comb               *dsp.AllPass
}

func newAllpass(size dsp.MS) (*allpass, error) {
	m := &allpass{
		in:       NewIn("input", dsp.Float64(0)),
		duration: NewInBuffer("duration", dsp.Duration(1000)),
		gain:     NewInBuffer("gain", dsp.Float64(0.9)),
		comb:     dsp.NewAllPassMS(size),
	}
	err := m.Expose(
		"Allpass",
		[]*In{m.in, m.duration, m.gain},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

func (p *allpass) Process(out dsp.Frame) {
	p.in.Process(out)
	gain := p.gain.ProcessFrame()
	duration := p.duration.ProcessFrame()
	for i := range out {
		out[i] = p.comb.TickDuration(out[i], gain[i], duration[i])
	}
}

type filteredFBDelay struct {
	IO
	in, duration, gain, cutoff, resonance *In
	comb                                  *dsp.FilteredFBComb
}

func newFilteredFBDelay(size dsp.MS) (*filteredFBDelay, error) {
	m := &filteredFBDelay{
		in:        NewIn("input", dsp.Float64(0)),
		duration:  NewInBuffer("duration", dsp.Duration(1000)),
		gain:      NewInBuffer("gain", dsp.Float64(0.98)),
		cutoff:    NewInBuffer("cutoff", dsp.Frequency(1000)),
		resonance: NewInBuffer("resonance", dsp.Float64(1)),
		comb:      dsp.NewFilteredFBCombMS(size, 4),
	}
	err := m.Expose(
		"FilteredFBDelay",
		[]*In{m.in, m.duration, m.gain, m.cutoff, m.resonance},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

func (c *filteredFBDelay) Process(out dsp.Frame) {
	c.in.Process(out)
	gain := c.gain.ProcessFrame()
	duration := c.duration.ProcessFrame()
	cutoff := c.cutoff.ProcessFrame()
	resonance := c.resonance.ProcessFrame()
	for i := range out {
		out[i] = c.comb.TickDuration(out[i], gain[i], duration[i], cutoff[i], resonance[i])
	}
}

type fbLoopDelay struct {
	IO
	in, duration, gain, feedbackReturn *In
	feedbackSend                       *Out

	sent dsp.Frame
	line *dsp.DelayLine
	last dsp.Float64
}

func newFBLoopDelay(size dsp.MS) (*fbLoopDelay, error) {
	m := &fbLoopDelay{
		in:       NewIn("input", dsp.Float64(0)),
		duration: NewInBuffer("duration", dsp.Duration(1000)),
		gain:     NewInBuffer("gain", dsp.Float64(0.98)),
		feedbackReturn: &In{
			Name:         "feedbackReturn",
			Source:       dsp.NewBuffer(dsp.Float64(0)),
			ForceSinking: true,
		},

		sent: dsp.NewFrame(),
		line: dsp.NewDelayLineMS(size),
	}
	m.feedbackSend = &Out{Name: "feedbackSend", Provider: dsp.Provide(&loopDelaySend{m})}

	err := m.Expose(
		"FBLoopDelay",
		[]*In{m.in, m.duration, m.gain, m.feedbackReturn},
		[]*Out{
			m.feedbackSend,
			{Name: "output", Provider: dsp.Provide(m)},
		},
	)
	return m, err
}

func (c *fbLoopDelay) Process(out dsp.Frame) {
	c.in.Process(out)
	gain := c.gain.ProcessFrame()
	duration := c.duration.ProcessFrame()
	if c.feedbackSend.IsActive() {
		c.feedbackReturn.ProcessFrame()
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

func (c *loopDelaySend) Process(out dsp.Frame) {
	for i := range out {
		out[i] = c.sent[i]
	}
}
