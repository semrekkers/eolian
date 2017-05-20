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
	multiOutIO
	in     *In
	frames []dsp.Frame
}

func newMultiple(size int) (*multiple, error) {
	m := &multiple{
		in:     NewInBuffer("input", dsp.Float64(0)),
		frames: make([]dsp.Frame, size),
	}

	outputs := []*Out{}
	for i := 0; i < size; i++ {
		m.frames[i] = dsp.NewFrame()
		outputs = append(outputs, &Out{
			Name:     fmt.Sprintf("%d", i),
			Provider: provideCopyOut(m, &m.frames[i]),
		})
	}

	return m, m.Expose("Multiple", []*In{m.in}, outputs)
}

func (m *multiple) Process(_ dsp.Frame) {
	m.incrRead(func() {
		in := m.in.ProcessFrame()
		for i := range m.frames {
			copy(m.frames[i], in)
		}
	})
}
