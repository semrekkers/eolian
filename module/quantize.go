package module

import (
	"fmt"
	"math"

	"buddin.us/eolian/dsp"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("Quantize", func(c Config) (Patcher, error) {
		var config struct{ Size int }
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Size == 0 {
			config.Size = 10
		}
		return newQuantize(config.Size)
	})
}

type quantize struct {
	IO
	in, transpose *In
	pitches       []*In

	frames []dsp.Frame
}

func newQuantize(size int) (*quantize, error) {
	m := &quantize{
		in:        NewIn("input", dsp.Float64(0)),
		transpose: NewInBuffer("transpose", dsp.Float64(1)),
		pitches:   make([]*In, size),
		frames:    make([]dsp.Frame, size),
	}

	inputs := []*In{m.in, m.transpose}
	for i := 0; i < size; i++ {
		in := NewInBuffer(fmt.Sprintf("%d/pitch", i), dsp.Float64(0))
		m.pitches[i] = in
		m.frames[i] = dsp.NewFrame()
		inputs = append(inputs, in)
	}

	return m, m.Expose(
		"FixedQuantize",
		inputs,
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
}

func (q *quantize) Process(out dsp.Frame) {
	q.in.Process(out)
	transpose := q.transpose.ProcessFrame()
	for i, p := range q.pitches {
		q.frames[i] = p.ProcessFrame()
	}
	for i := range out {
		n := float64(len(q.pitches))
		idx := math.Floor(n*float64(out[i]) + 0.5)
		idx = math.Min(idx, n-1)
		idx = math.Max(idx, 0)
		out[i] = q.frames[int(idx)][i] * transpose[i]
	}
}
