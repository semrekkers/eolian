package module

import "buddin.us/eolian/dsp"

func init() {
	Register("Edges", func(c Config) (Patcher, error) { return newEdges() })
}

type edges struct {
	multiOutIO
	in                *In
	endRise, endCycle dsp.Frame
	lastIn            dsp.Float64
}

func newEdges() (*edges, error) {
	m := &edges{
		in:       NewIn("input", dsp.Float64(0)),
		endRise:  dsp.NewFrame(),
		endCycle: dsp.NewFrame(),
	}
	return m, m.Expose(
		"Edges",
		[]*In{m.in},
		[]*Out{
			{Name: "endRise", Provider: provideCopyOut(m, &m.endRise)},
			{Name: "endCycle", Provider: provideCopyOut(m, &m.endCycle)},
		},
	)
}

func (e *edges) Process(out dsp.Frame) {
	e.incrRead(func() {
		e.in.Process(out)

		for i := range out {
			if e.lastIn < 0 && out[i] > 0 {
				e.endRise[i] = 1
				e.endCycle[i] = -1
			}
			if e.lastIn > 0 && out[i] < 0 {
				e.endCycle[i] = 1
				e.endRise[i] = -1
			} else {
				e.endCycle[i] = -1
				e.endRise[i] = -1
			}

			e.lastIn = out[i]
		}
	})
}
