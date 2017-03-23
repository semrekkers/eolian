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
	a, b, duration, gain *In
	aDelay, bDelay       *delayline
	aOut, bOut           Frame
	aLast, bLast         Value
	size                 MS
}

func newPingPongDelay(size MS) (*pingPongDelay, error) {
	m := &pingPongDelay{
		a:        &In{Name: "a", Source: NewBuffer(zero)},
		b:        &In{Name: "b", Source: NewBuffer(zero)},
		duration: &In{Name: "duration", Source: NewBuffer(Duration(1000))},
		gain:     &In{Name: "gain", Source: NewBuffer(Value(0.5))},
		aDelay:   newDelayLine(size),
		bDelay:   newDelayLine(size),
		aOut:     make(Frame, FrameSize),
		bOut:     make(Frame, FrameSize),
		size:     size,
	}

	return m, m.Expose("PingPongDelay", []*In{m.a, m.b, m.duration, m.gain}, []*Out{
		{Name: "a", Provider: provideCopyOut(m, &m.aOut)},
		{Name: "b", Provider: provideCopyOut(m, &m.bOut)},
	})
}

func (p *pingPongDelay) Read(out Frame) {
	p.incrRead(func() {
		a, b := p.a.ReadFrame(), p.b.ReadFrame()
		duration := p.duration.ReadFrame()
		gain := p.gain.ReadFrame()
		for i := range out {
			d := clampValue(duration[i], 0, p.size.Value())

			p.aOut[i] = a[i] + p.bLast
			p.bOut[i] = b[i] + p.aLast

			p.aLast = gain[i] * p.aDelay.TickDuration(a[i], d)
			p.bLast = gain[i] * p.bDelay.TickDuration(b[i], d)
		}
	})
}
