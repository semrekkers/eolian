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
	return m, m.Expose("mux", inputs, []*Out{{Name: "output", Provider: Provide(m)}})
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
	IO
	in, selection *In
	outs          []Frame
	readTracker   manyReadTracker
}

func newDemux(size int) (*demux, error) {
	m := &demux{
		in:        &In{Name: "input", Source: zero},
		selection: &In{Name: "selection", Source: NewBuffer(zero)},
		outs:      make([]Frame, size),
	}
	m.readTracker = manyReadTracker{counter: m}
	inputs := []*In{m.in, m.selection}
	outputs := []*Out{}
	for i := 0; i < size; i++ {
		m.outs[i] = make(Frame, FrameSize)
		outputs = append(outputs, &Out{
			Name:     fmt.Sprintf("%d", i),
			Provider: m.out(&m.outs[i]),
		})
	}
	return m, m.Expose("mux", inputs, outputs)
}

func (m *demux) out(cache *Frame) ReaderProvider {
	return ReaderProviderFunc(func() Reader {
		return &manyOut{reader: m, cache: cache}
	})
}

func (m *demux) readMany(out Frame) {
	if m.readTracker.count() > 0 {
		m.readTracker.incr()
		return
	}

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

	m.readTracker.incr()
}
