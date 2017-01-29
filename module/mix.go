package module

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("Mix", func(c Config) (Patcher, error) {
		var config struct {
			Size int
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Size == 0 {
			config.Size = 4
		}
		return newMix(config.Size)
	})
}

type mix struct {
	IO
	master          *In
	sources, levels []*In
}

func newMix(size int) (*mix, error) {
	m := &mix{
		master: &In{Name: "master", Source: NewBuffer(Value(1))},
	}
	inputs := []*In{m.master}
	for i := 0; i < size; i++ {
		in := &In{
			Name:   fmt.Sprintf("%d/input", i),
			Source: NewBuffer(zero),
		}
		level := &In{
			Name:   fmt.Sprintf("%d/level", i),
			Source: NewBuffer(Value(1)),
		}
		m.sources = append(m.sources, in)
		m.levels = append(m.levels, level)
		inputs = append(inputs, in, level)
	}
	return m, m.Expose("Mix", inputs, []*Out{{Name: "output", Provider: Provide(m)}})
}

func (m *mix) Read(out Frame) {
	master := m.master.ReadFrame()
	for i := 0; i < len(m.sources); i++ {
		m.sources[i].ReadFrame()
		m.levels[i].ReadFrame()
	}

	for i := range out {
		var sum Value
		for j := 0; j < len(m.sources); j++ {
			sum += m.sources[j].LastFrame()[i] * m.levels[j].LastFrame()[i]
		}
		out[i] = sum * master[i]
	}
}
