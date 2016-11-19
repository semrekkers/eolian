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
		return NewMix(config.Size)
	})
}

type Mix struct {
	IO
	master          *In
	sources, levels []*In

	size              int
	names, levelNames []string
}

func NewMix(size int) (*Mix, error) {
	m := &Mix{
		master: &In{Name: "master", Source: NewBuffer(Value(1))},

		size:       size,
		names:      []string{},
		levelNames: []string{},
	}
	inputs := []*In{m.master}
	for i := 0; i < m.size; i++ {
		in := &In{
			Name:   fmt.Sprintf("%d.input", i),
			Source: NewBuffer(zero),
		}
		level := &In{
			Name:   fmt.Sprintf("%d.level", i),
			Source: NewBuffer(Value(1)),
		}
		m.sources = append(m.sources, in)
		m.levels = append(m.levels, level)
		inputs = append(inputs, in, level)
	}
	err := m.Expose(inputs, []*Out{{Name: "output", Provider: Provide(m)}})
	return m, err
}

func (reader *Mix) Read(out Frame) {
	master := reader.master.ReadFrame()
	for i := 0; i < reader.size; i++ {
		reader.sources[i].ReadFrame()
		reader.levels[i].ReadFrame()
	}

	for i := range out {
		var sum Value
		for j := 0; j < reader.size; j++ {
			sum += reader.sources[j].LastFrame()[i] * reader.levels[j].LastFrame()[i]
		}
		out[i] = sum * master[i]
	}
}
