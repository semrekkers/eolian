package module

import (
	"fmt"

	"buddin.us/eolian/dsp"

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
		master: NewInBuffer("master", dsp.Float64(1)),
	}
	inputs := []*In{m.master}
	for i := 0; i < size; i++ {
		in := NewInBuffer(fmt.Sprintf("%d/input", i), dsp.Float64(0))
		level := NewInBuffer(fmt.Sprintf("%d/level", i), dsp.Float64(1))
		m.sources = append(m.sources, in)
		m.levels = append(m.levels, level)
		inputs = append(inputs, in, level)
	}
	return m, m.Expose("Mix", inputs, []*Out{{Name: "output", Provider: dsp.Provide(m)}})
}

func (m *mix) Process(out dsp.Frame) {
	master := m.master.ProcessFrame()
	for i := 0; i < len(m.sources); i++ {
		m.sources[i].ProcessFrame()
		m.levels[i].ProcessFrame()
	}

	for i := range out {
		var sum dsp.Float64
		for j := 0; j < len(m.sources); j++ {
			sum += m.sources[j].LastFrame()[i] * m.levels[j].LastFrame()[i]
		}
		out[i] = sum * master[i]
	}
}
