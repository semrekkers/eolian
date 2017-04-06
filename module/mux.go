package module

import (
	"fmt"

	"buddin.us/eolian/dsp"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("Mux", func(c Config) (Patcher, error) {
		var config struct{ Size int }
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Size == 0 {
			config.Size = 2
		}
		return newMux(config.Size)
	})
	Register("Demux", func(c Config) (Patcher, error) {
		var config struct{ Size int }
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Size == 0 {
			config.Size = 2
		}
		return newDemux(config.Size)
	})
}

type mux struct {
	IO
	selection *In
	sources   []*In
}

func newMux(size int) (*mux, error) {
	m := &mux{
		selection: NewInBuffer("selection", dsp.Float64(0)),
	}
	inputs := []*In{m.selection}
	for i := 0; i < size; i++ {
		in := NewInBuffer(fmt.Sprintf("%d/input", i), dsp.Float64(0))
		m.sources = append(m.sources, in)
		inputs = append(inputs, in)
	}
	return m, m.Expose("Mux", inputs, []*Out{{Name: "output", Provider: dsp.Provide(m)}})
}

func (m *mux) Process(out dsp.Frame) {
	selection := m.selection.ProcessFrame()
	for i := 0; i < len(m.sources); i++ {
		m.sources[i].ProcessFrame()
	}
	for i := range out {
		s := int(dsp.Clamp(selection[i], 0, dsp.Float64(len(m.sources)-1)))
		out[i] = m.sources[s].LastFrame()[i]
	}
}

type demux struct {
	multiOutIO
	in, selection *In
	outs          []dsp.Frame
}

func newDemux(size int) (*demux, error) {
	m := &demux{
		in:        NewIn("input", dsp.Float64(0)),
		selection: NewInBuffer("selection", dsp.Float64(0)),
		outs:      make([]dsp.Frame, size),
	}
	inputs := []*In{m.in, m.selection}
	outputs := []*Out{}
	for i := 0; i < size; i++ {
		m.outs[i] = dsp.NewFrame()
		outputs = append(outputs, &Out{
			Name:     fmt.Sprintf("%d", i),
			Provider: provideCopyOut(m, &m.outs[i]),
		})
	}
	return m, m.Expose("Demux", inputs, outputs)
}

func (m *demux) Process(out dsp.Frame) {
	m.incrRead(func() {
		m.in.Process(out)
		selection := m.selection.ProcessFrame()
		for i := range out {
			s := int(dsp.Clamp(selection[i], 0, dsp.Float64(len(m.outs)-1)))
			for j := 0; j < len(m.outs); j++ {
				if j == s {
					m.outs[j][i] = out[i]
				} else {
					m.outs[j][i] = 0
				}
			}
		}
	})
}
