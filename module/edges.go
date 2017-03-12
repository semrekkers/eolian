package module

func init() {
	Register("Edges", func(c Config) (Patcher, error) { return newEdges() })
}

type edges struct {
	multiOutIO
	in                *In
	endRise, endCycle Frame
	lastIn            Value
}

func newEdges() (*edges, error) {
	m := &edges{
		in:       &In{Name: "input", Source: NewBuffer(zero)},
		endRise:  make(Frame, FrameSize),
		endCycle: make(Frame, FrameSize),
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

func (e *edges) Read(out Frame) {
	e.incrRead(func() {
		e.in.Read(out)

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
