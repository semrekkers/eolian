package module

import (
	"fmt"

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
		selection: &In{Name: "selection", Source: NewBuffer(zero)},
	}
	inputs := []*In{m.selection}
	for i := 0; i < size; i++ {
		in := &In{
			Name:   fmt.Sprintf("%d/input", i),
			Source: NewBuffer(zero),
		}
		m.sources = append(m.sources, in)
		inputs = append(inputs, in)
	}
	return m, m.Expose("Mux", inputs, []*Out{{Name: "output", Provider: Provide(m)}})
}

func (m *mux) Read(out Frame) {
	selection := m.selection.ReadFrame()
	for i := 0; i < len(m.sources); i++ {
		m.sources[i].ReadFrame()
	}
	for i := range out {
		s := int(clampValue(selection[i], 0, Value(len(m.sources)-1)))
		out[i] = m.sources[s].LastFrame()[i]
	}
}

type demux struct {
	multiOutIO
	in, selection *In
	outs          []Frame
}

func newDemux(size int) (*demux, error) {
	m := &demux{
		in:        &In{Name: "input", Source: zero},
		selection: &In{Name: "selection", Source: NewBuffer(zero)},
		outs:      make([]Frame, size),
	}
	inputs := []*In{m.in, m.selection}
	outputs := []*Out{}
	for i := 0; i < size; i++ {
		m.outs[i] = make(Frame, FrameSize)
		outputs = append(outputs, &Out{
			Name:     fmt.Sprintf("%d", i),
			Provider: provideCopyOut(m, &m.outs[i]),
		})
	}
	return m, m.Expose("Demux", inputs, outputs)
}

func (m *demux) Read(out Frame) {
	m.incrRead(func() {
		m.in.Read(out)
		selection := m.selection.ReadFrame()
		for i := range out {
			s := int(clampValue(selection[i], 0, Value(len(m.outs)-1)))
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
