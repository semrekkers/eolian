package module

import (
	"fmt"

	"buddin.us/eolian/dsp"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("Switch", func(c Config) (Patcher, error) {
		var config struct {
			Size int
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Size == 0 {
			config.Size = 4
		}
		return newSeqSwitch(config.Size)
	})
}

type seqSwitch struct {
	IO
	clock, reset *In
	sources      []*In

	step                 int
	lastClock, lastReset dsp.Float64
}

func newSeqSwitch(size int) (*seqSwitch, error) {
	m := &seqSwitch{
		clock:     NewInBuffer("clock", dsp.Float64(0)),
		reset:     NewInBuffer("reset", dsp.Float64(0)),
		lastClock: -1,
	}
	inputs := []*In{m.clock, m.reset}
	for i := 0; i < size; i++ {
		in := NewInBuffer(fmt.Sprintf("%d/input", i), dsp.Float64(0))
		m.sources = append(m.sources, in)
		inputs = append(inputs, in)
	}
	return m, m.Expose("Switch", inputs, []*Out{{Name: "output", Provider: dsp.Provide(m)}})
}

func (s *seqSwitch) Process(out dsp.Frame) {
	clock := s.clock.ProcessFrame()
	reset := s.reset.ProcessFrame()
	for i := 0; i < len(s.sources); i++ {
		s.sources[i].ProcessFrame()
	}
	for i := range out {
		if s.lastReset < 0 && reset[i] > 0 {
			s.step = 0
		} else if s.lastClock < 0 && clock[i] > 0 {
			s.step = (s.step + 1) % len(s.sources)
		}
		out[i] = s.sources[s.step].LastFrame()[i]
		s.lastClock = clock[i]
	}
}
