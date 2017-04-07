package module

import "buddin.us/eolian/dsp"

func init() {
	Register("Clip", func(Config) (Patcher, error) { return newClip() })
}

type clip struct {
	IO
	in, level *In
}

func newClip() (*clip, error) {
	m := &clip{
		in:    NewIn("input", dsp.Float64(0)),
		level: NewInBuffer("level", dsp.Float64(1)),
	}
	err := m.Expose(
		"Clip",
		[]*In{m.in, m.level},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

func (c *clip) Process(out dsp.Frame) {
	c.in.Process(out)
	level := c.level.ProcessFrame()
	for i := range out {
		level := level[i]
		if out[i] > level {
			out[i] = level
		} else if out[i] < -level {
			out[i] = -level
		}
	}
}
