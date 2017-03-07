package module

func init() {
	Register("TrackHold", func(Config) (Patcher, error) { return newTrackHold() })
}

type trackHold struct {
	IO
	in, hang  *In
	lastValue Value
}

func newTrackHold() (*trackHold, error) {
	m := &trackHold{
		in:   &In{Name: "input", Source: zero},
		hang: &In{Name: "hang", Source: NewBuffer(zero)},
	}
	return m, m.Expose(
		"trackHold",
		[]*In{m.in, m.hang},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
}

func (th *trackHold) Read(out Frame) {
	th.in.Read(out)
	hang := th.hang.ReadFrame()
	for i := range out {
		if hang[i] > 0 {
			out[i] = th.lastValue
		}
		th.lastValue = out[i]
	}
}
