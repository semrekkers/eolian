package module

import (
	"strconv"

	"buddin.us/eolian/dsp"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("GateMix", func(c Config) (Patcher, error) {
		var config struct{ Size int }
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Size == 0 {
			config.Size = 4
		}
		return newGateMix(config.Size)
	})
}

type gateMix struct {
	IO
	sources []*In
}

func newGateMix(size int) (*gateMix, error) {
	m := &gateMix{
		sources: make([]*In, size),
	}
	for i := range m.sources {
		m.sources[i] = NewInBuffer(strconv.Itoa(i), dsp.Float64(0))
	}
	return m, m.Expose("GateMix", m.sources, []*Out{{Name: "output", Provider: dsp.Provide(m)}})
}

func (m *gateMix) Process(out dsp.Frame) {
	for _, s := range m.sources {
		s.ProcessFrame()
	}

	for i := range out {
		out[i] = -1
		for _, s := range m.sources {
			if v := s.LastFrame()[i]; v > 0 {
				out[i] = 1
				break
			}
		}
	}
}
