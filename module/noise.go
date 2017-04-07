package module

import "buddin.us/eolian/dsp"

func init() {
	Register("Noise", func(Config) (Patcher, error) { return newNoise() })
}

type noise struct {
	IO
	in, min, max, gain *In
}

func newNoise() (*noise, error) {
	m := &noise{
		in:   NewIn("input", dsp.Float64(0)),
		min:  NewInBuffer("min", dsp.Float64(-1)),
		max:  NewInBuffer("max", dsp.Float64(1)),
		gain: NewInBuffer("gain", dsp.Float64(1)),
	}
	err := m.Expose(
		"Noise",
		[]*In{m.in, m.min, m.max, m.gain},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

func (n *noise) Process(out dsp.Frame) {
	n.in.Process(out)
	min := n.min.ProcessFrame()
	max := n.max.ProcessFrame()
	gain := n.gain.ProcessFrame()
	for i := range out {
		diff := max[i] - min[i]
		out[i] += (dsp.Rand()*diff + min[i]) * gain[i]
	}
}
