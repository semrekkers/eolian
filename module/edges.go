package module

import (
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("Edges", func(c Config) (Patcher, error) {
		var config struct{}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		// Defaults
		return newEdges()
	})
}

type edges struct {
	IO
	in                    *In
	readTracker           manyReadTracker
	endOfRise, endOfCycle Frame
	lastIn                Value
}

func newEdges() (*edges, error) {
	m := &edges{
		in:         &In{Name: "input", Source: NewBuffer(zero)},
		endOfRise:  make(Frame, FrameSize),
		endOfCycle: make(Frame, FrameSize),
	}
	m.readTracker = manyReadTracker{counter: m}
	return m, m.Expose(
		"Edges",
		[]*In{m.in},
		[]*Out{
			{Name: "endRise", Provider: m.out(&m.endOfRise)},
			{Name: "endCycle", Provider: m.out(&m.endOfCycle)},
		},
	)
}

func (e *edges) out(cache *Frame) ReaderProvider {
	return ReaderProviderFunc(func() Reader {
		return &manyOut{reader: e, cache: cache}
	})
}

func (e *edges) readMany(out Frame) {
	if e.readTracker.count() > 0 {
		e.readTracker.incr()
		return
	}
	e.in.Read(out)

	for i := range out {
		if e.lastIn < 0 && out[i] > 0 {
			e.endOfRise[i] = 1
			e.endOfCycle[i] = -1
		}
		if e.lastIn > 0 && out[i] < 0 {
			e.endOfCycle[i] = 1
			e.endOfRise[i] = -1
		} else {
			e.endOfCycle[i] = -1
			e.endOfRise[i] = -1
		}

		e.lastIn = out[i]
	}
	e.readTracker.incr()
}
