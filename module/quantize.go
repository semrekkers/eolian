package module

import (
	"fmt"
	"math"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("Quantize", func(c Config) (Patcher, error) {
		var config struct {
			Size int
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Size == 0 {
			config.Size = 10
		}
		return NewQuantize(config.Size)
	})
}

type Quantize struct {
	IO
	in, max *In
	pitches []*In

	frames []Frame
}

func NewQuantize(size int) (*Quantize, error) {
	m := &Quantize{
		in:      &In{Name: "input", Source: zero},
		max:     &In{Name: "max", Source: NewBuffer(zero)},
		pitches: make([]*In, size),
		frames:  make([]Frame, size),
	}

	inputs := []*In{m.in, m.max}
	for i := 0; i < size; i++ {
		in := &In{
			Name:   fmt.Sprintf("%d.pitch", i),
			Source: NewBuffer(zero),
		}
		m.pitches[i] = in
		m.frames[i] = make(Frame, FrameSize)
		inputs = append(inputs, in)
	}

	err := m.Expose(
		inputs,
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *Quantize) Read(out Frame) {
	reader.in.Read(out)
	for i, p := range reader.pitches {
		reader.frames[i] = p.ReadFrame()
	}
	for i := range out {
		idx := math.Floor(10*float64(out[i]) + 0.5)
		idx = math.Min(idx, float64(len(reader.pitches)-1))
		idx = math.Max(idx, 0)
		out[i] = reader.frames[int(idx)][i]
	}
}