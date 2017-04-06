package module

import (
	"buddin.us/eolian/dsp"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("FBPingPongDelay", func(c Config) (Patcher, error) {
		var config struct{ Size int }
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Size == 0 {
			config.Size = 10000
		}
		return newPingPongDelay(dsp.DurationInt(config.Size))
	})
}

type pingPongDelay struct {
	multiOutIO
	a, b, duration, gain *In
	aDelay, bDelay       *dsp.DelayLine
	aOut, bOut           dsp.Frame
	aLast, bLast         dsp.Float64
	size                 dsp.MS
}

func newPingPongDelay(size dsp.MS) (*pingPongDelay, error) {
	m := &pingPongDelay{
		a:        NewInBuffer("a", dsp.Float64(0)),
		b:        NewInBuffer("b", dsp.Float64(0)),
		duration: NewInBuffer("duration", dsp.Duration(1000)),
		gain:     NewInBuffer("gain", dsp.Float64(0.5)),
		aDelay:   dsp.NewDelayLine(size),
		bDelay:   dsp.NewDelayLine(size),
		aOut:     dsp.NewFrame(),
		bOut:     dsp.NewFrame(),
		size:     size,
	}

	return m, m.Expose("PingPongDelay", []*In{m.a, m.b, m.duration, m.gain}, []*Out{
		{Name: "a", Provider: provideCopyOut(m, &m.aOut)},
		{Name: "b", Provider: provideCopyOut(m, &m.bOut)},
	})
}

func (p *pingPongDelay) Process(out dsp.Frame) {
	p.incrRead(func() {
		a, b := p.a.ProcessFrame(), p.b.ProcessFrame()
		duration := p.duration.ProcessFrame()
		gain := p.gain.ProcessFrame()
		for i := range out {
			d := dsp.Clamp(duration[i], 0, p.size.Value())

			p.aOut[i] = a[i] + p.bLast
			p.bOut[i] = b[i] + p.aLast

			p.aLast = gain[i] * p.aDelay.TickDuration(a[i], d)
			p.bLast = gain[i] * p.bDelay.TickDuration(b[i], d)
		}
	})
}
