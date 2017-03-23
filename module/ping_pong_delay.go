package module

import "github.com/mitchellh/mapstructure"

func init() {
	Register("PingPongDelay", func(c Config) (Patcher, error) {
		var config struct{ Size int }
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Size == 0 {
			config.Size = 10000
		}
		return newPingPongDelay(DurationInt(config.Size))
	})
}

type pingPongDelay struct {
	multiOutIO
	in, duration   *In
	aDelay, bDelay *delayline
	a, b           Frame
	size           MS
}

func newPingPongDelay(size MS) (*pingPongDelay, error) {
	m := &pingPongDelay{
		in:       &In{Name: "input", Source: NewBuffer(zero)},
		duration: &In{Name: "duration", Source: NewBuffer(Duration(1000))},
		aDelay:   newDelayLine(size),
		bDelay:   newDelayLine(size),
		a:        make(Frame, FrameSize),
		b:        make(Frame, FrameSize),
		size:     size,
	}

	return m, m.Expose("PingPongDelay", []*In{m.in, m.duration}, []*Out{
		{Name: "a", Provider: provideCopyOut(m, &m.a)},
		{Name: "b", Provider: provideCopyOut(m, &m.b)},
	})
}

func (p *pingPongDelay) Read(out Frame) {
	p.incrRead(func() {
		in := p.in.ReadFrame()
		duration := p.duration.ReadFrame()
		for i := range out {
			d := clampValue(duration[i], 0, p.size.Value())
			p.a[i] = p.aDelay.TickDuration(in[i], d)
			p.b[i] = p.bDelay.TickDuration(p.a[i], d)
		}
	})
}
