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
	in      *In
	pitches []*In

	frames []Frame
}

func NewQuantize(size int) (*Quantize, error) {
	m := &Quantize{
		in:      &In{Name: "input", Source: zero},
		pitches: make([]*In, size),
		frames:  make([]Frame, size),
	}

	inputs := []*In{m.in}
	for i := 0; i < size; i++ {
		in := &In{
			Name:   fmt.Sprintf("%d/pitch", i),
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

func (q *Quantize) Read(out Frame) {
	q.in.Read(out)
	for i, p := range q.pitches {
		q.frames[i] = p.ReadFrame()
	}
	for i := range out {
		n := float64(len(q.pitches))
		idx := math.Floor(n*float64(out[i]) + 0.5)
		idx = math.Min(idx, n-1)
		idx = math.Max(idx, 0)
		out[i] = q.frames[int(idx)][i]
	}
}
