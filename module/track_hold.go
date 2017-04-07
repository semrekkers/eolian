package module

import "buddin.us/eolian/dsp"

func init() {
	Register("TrackHold", func(Config) (Patcher, error) { return newTrackHold() })
}

type trackHold struct {
	IO
	in, hang  *In
	lastValue dsp.Float64
}

func newTrackHold() (*trackHold, error) {
	m := &trackHold{
		in:   NewIn("input", dsp.Float64(0)),
		hang: NewInBuffer("hang", dsp.Float64(0)),
	}
	return m, m.Expose(
		"TrackHold",
		[]*In{m.in, m.hang},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
}

func (th *trackHold) Process(out dsp.Frame) {
	th.in.Process(out)
	hang := th.hang.ProcessFrame()
	for i := range out {
		if hang[i] > 0 {
			out[i] = th.lastValue
		}
		th.lastValue = out[i]
	}
}
