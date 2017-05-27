package module

import (
	"buddin.us/eolian/dsp"
)

func init() {
	Register("Freeze", func(Config) (Patcher, error) { return newFreeze() })
}

type freeze struct {
	IO
	in, gate, size, rate, bits *In
	lastGate, max              dsp.Float64
	write, read                dsp.Frame
	start, offset              int
	decimate                   *dsp.Decimate
}

func newFreeze() (*freeze, error) {
	max := 3 * dsp.SampleRate

	m := &freeze{
		in:       NewInBuffer("input", dsp.Float64(0)),
		gate:     NewInBuffer("gate", dsp.Float64(-1)),
		size:     NewInBuffer("size", dsp.Duration(1)),
		rate:     NewInBuffer("rate", dsp.Float64(44100)),
		bits:     NewInBuffer("bits", dsp.Float64(24)),
		write:    make(dsp.Frame, int(max)),
		read:     make(dsp.Frame, int(max)),
		max:      dsp.Float64(max),
		lastGate: -1,
		decimate: &dsp.Decimate{},
	}

	err := m.Expose(
		"Freeze",
		[]*In{m.in, m.gate, m.size, m.rate, m.bits},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

func (f *freeze) Process(out dsp.Frame) {
	var (
		in   = f.in.ProcessFrame()
		gate = f.gate.ProcessFrame()
		size = f.size.ProcessFrame()
		rate = f.rate.ProcessFrame()
		bits = f.bits.ProcessFrame()
	)

	for i := range out {
		if f.lastGate < 0 && gate[i] > 0 {
			f.start = f.offset
			f.write, f.read = f.read, f.write
		}

		f.write[f.offset] = in[i]

		if gate[i] > 0 {
			out[i] = f.decimate.Tick(f.read[f.offset], rate[i], bits[i])
			f.offset++
			if f.offset >= f.start+int(dsp.Clamp(size[i], 1, f.max)) {
				f.offset = f.start
			}
			if f.offset >= len(f.read) {
				f.offset = 0
			}
		} else {
			out[i] = in[i]
			f.offset = (f.offset + 1) % len(f.read)
		}
		f.lastGate = gate[i]
	}
}
