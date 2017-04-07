package module

import (
	"fmt"

	"buddin.us/eolian/dsp"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("Multiple", func(c Config) (Patcher, error) {
		var config struct {
			Size int
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Size == 0 {
			config.Size = 4
		}
		return newMultiple(config.Size)
	})
}

type multiple struct {
	IO
	in *In

	frame dsp.Frame
	size  int
	reads int
}

func newMultiple(size int) (*multiple, error) {
	m := &multiple{
		in:    NewInBuffer("input", dsp.Float64(0)),
		frame: dsp.NewFrame(),
		size:  size,
	}

	outputs := []*Out{}
	for i := 0; i < size; i++ {
		name := fmt.Sprintf("%d", i)
		outputs = append(outputs, &Out{Name: name, Provider: dsp.Provide(&multOut{m})})
	}

	return m, m.Expose("Multiple", []*In{m.in}, outputs)
}

func (m *multiple) read(out dsp.Frame) {
	if m.reads == 0 {
		copy(m.frame, m.in.ProcessFrame())
	}
	for i := range out {
		out[i] = m.frame[i]
	}
	if outs := m.OutputsActive(true); outs > 0 {
		m.reads = (m.reads + 1) % outs
	}
}

type multOut struct {
	*multiple
}

func (o *multOut) Process(out dsp.Frame) {
	o.multiple.read(out)
}
