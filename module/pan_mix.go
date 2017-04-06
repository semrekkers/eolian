package module

import (
	"fmt"

	"buddin.us/eolian/dsp"

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
	a, b                  dsp.Frame
}

func newPanMix(size int) (*panMix, error) {
	m := &panMix{
		master: NewInBuffer("master", dsp.Float64(1)),
		a:      dsp.NewFrame(),
		b:      dsp.NewFrame(),
	}
	inputs := []*In{m.master}
	for i := 0; i < size; i++ {
		in := NewInBuffer(fmt.Sprintf("%d/input", i), dsp.Float64(0))
		level := NewInBuffer(fmt.Sprintf("%d/level", i), dsp.Float64(1))
		pan := NewInBuffer(fmt.Sprintf("%d/pan", i), dsp.Float64(0))

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

func (m *panMix) Process(out dsp.Frame) {
	m.incrRead(func() {
		master := m.master.ProcessFrame()
		for i := 0; i < len(m.sources); i++ {
			m.sources[i].ProcessFrame()
			m.levels[i].ProcessFrame()
			m.pans[i].ProcessFrame()
		}

		for i := range out {
			var aSum, bSum dsp.Float64
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
