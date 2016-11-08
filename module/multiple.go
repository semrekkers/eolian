package module

import (
	"fmt"

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
		return NewMultipleSized(config.Size)
	})
}

type Multiple struct {
	IO
	in *In

	frame Frame
	size  int
	reads int
}

func NewMultipleSized(size int) (*Multiple, error) {
	m := &Multiple{
		in:    &In{Name: "input", Source: NewBuffer(zero)},
		frame: make(Frame, FrameSize),
		size:  size,
	}

	outputs := []*Out{}
	for i := 0; i < size; i++ {
		name := fmt.Sprintf("%d", i)
		outputs = append(outputs, &Out{Name: name, Provider: Provide(&multOut{m})})
	}

	err := m.Expose([]*In{m.in}, outputs)
	return m, err
}

func (m *Multiple) read(out Frame) {
	if m.reads == 0 {
		copy(m.frame, m.in.ReadFrame())
	}
	for i := range out {
		out[i] = m.frame[i]
	}
	if outs := m.OutputsActive(); outs > 0 {
		m.reads = (m.reads + 1) % outs
	}
}

type multOut struct {
	*Multiple
}

func (reader *multOut) Read(out Frame) {
	reader.Multiple.read(out)
}
