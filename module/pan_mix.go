package module

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("PanMix", func(c Config) (Patcher, error) {
		var config struct {
			Size int
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Size == 0 {
			config.Size = 4
		}
		return newPanMix(config.Size)
	})
}

type panMix struct {
	multiOutIO
	master                *In
	sources, levels, pans []*In
	a, b                  Frame
}

func newPanMix(size int) (*panMix, error) {
	m := &panMix{
		master: &In{Name: "master", Source: NewBuffer(Value(1))},
		a:      make(Frame, FrameSize),
		b:      make(Frame, FrameSize),
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
		pan := &In{
			Name:   fmt.Sprintf("%d/pan", i),
			Source: NewBuffer(zero),
		}
		m.sources = append(m.sources, in)
		m.levels = append(m.levels, level)
		m.pans = append(m.pans, pan)
		inputs = append(inputs, in, level, pan)
	}
	return m, m.Expose("PanMix", inputs, []*Out{
		{Name: "a", Provider: provideCopyOut(m, &m.a)},
		{Name: "b", Provider: provideCopyOut(m, &m.b)},
	})
}

func (m *panMix) Read(out Frame) {
	m.incrRead(func() {
		master := m.master.ReadFrame()
		for i := 0; i < len(m.sources); i++ {
			m.sources[i].ReadFrame()
			m.levels[i].ReadFrame()
			m.pans[i].ReadFrame()
		}

		for i := range out {
			var aSum, bSum Value
			for j := 0; j < len(m.sources); j++ {
				signal := m.sources[j].LastFrame()[i] * m.levels[j].LastFrame()[i]
				bias := m.pans[j].LastFrame()[i]

				if bias > 0 {
					aSum += (1 - bias) * signal
					bSum += signal
				} else if bias < 0 {
					aSum += signal
					bSum += (1 + bias) * signal
				} else {
					aSum += signal
					bSum += signal
				}
			}
			m.a[i] = aSum * master[i]
			m.b[i] = bSum * master[i]
		}
	})
}
