package module

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("StepSequence", func(c Config) (Patcher, error) {
		var config struct {
			Steps, Layers int
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Layers == 0 {
			config.Layers = 3
		} else if config.Layers > 24 {
			config.Layers = 24
		}
		if config.Steps == 0 {
			config.Steps = 8
		}
		return newStepSequence(config.Steps, config.Layers)
	})
}

type stepSequence struct {
	IO
	readTracker manyReadTracker

	clock, reset                          *In
	enables                               []*In
	steps                                 [][]step
	step, lastStep, layerCount, stepCount int
	pong                                  bool

	lastClock, lastReset Value

	pitchesOut []Frame
	gatesOut   [][]Frame
}

type step struct {
	pitch *In
	gate  *Out
}

func newStepSequence(steps, layers int) (*stepSequence, error) {
	m := &stepSequence{
		clock:      &In{Name: "clock", Source: NewBuffer(zero)},
		reset:      &In{Name: "reset", Source: NewBuffer(zero)},
		enables:    make([]*In, steps),
		pitchesOut: make([]Frame, layers),
		gatesOut:   make([][]Frame, layers),
		steps:      make([][]step, layers),
		lastClock:  -1,
		lastReset:  -1,
		lastStep:   -1,
		stepCount:  steps,
		layerCount: layers,
	}
	m.readTracker = manyReadTracker{counter: m}

	var (
		inputs  = []*In{m.clock, m.reset}
		outputs = []*Out{}
	)

	for i := 0; i < layers; i++ {
		m.gatesOut[i] = make([]Frame, steps)
		m.steps[i] = make([]step, steps)

		m.pitchesOut[i] = make(Frame, FrameSize)

		outputs = append(outputs, &Out{
			Name:     fmt.Sprintf("%c/pitch", alphaSeries[i]),
			Provider: m.out(&m.pitchesOut[i]),
		})

		for j := 0; j < steps; j++ {
			m.gatesOut[i][j] = make(Frame, FrameSize)
			m.steps[i][j] = step{
				pitch: &In{
					Name:   fmt.Sprintf("%c/%d/pitch", alphaSeries[i], j),
					Source: NewBuffer(zero),
				},
				gate: &Out{
					Name:     fmt.Sprintf("%c/%d/gate", alphaSeries[i], j),
					Provider: m.out(&m.gatesOut[i][j]),
				},
			}
			inputs = append(inputs, m.steps[i][j].pitch)
			outputs = append(outputs, m.steps[i][j].gate)
		}
	}

	for i := 0; i < steps; i++ {
		m.enables[i] = &In{
			Name:   fmt.Sprintf("%d/enabled", i),
			Source: NewBuffer(Value(1)),
		}
		inputs = append(inputs, m.enables[i])
	}

	return m, m.Expose("StepSequence", inputs, outputs)
}

func (s *stepSequence) out(cache *Frame) ReaderProvider {
	return ReaderProviderFunc(func() Reader {
		return &manyOut{reader: s, cache: cache}
	})
}

func (s *stepSequence) readMany(out Frame) {
	if s.readTracker.count() > 0 {
		s.readTracker.incr()
		return
	}

	clock := s.clock.ReadFrame()
	reset := s.reset.ReadFrame()
	for l, layer := range s.steps {
		for i := range layer {
			s.steps[l][i].pitch.ReadFrame()
		}
	}
	for i := 0; i < s.stepCount; i++ {
		s.enables[i].ReadFrame()
	}

	for i := range out {
		if s.lastClock < 0 && clock[i] > 0 {
			s.step = (s.step + 1) % s.stepCount
		}
		if s.lastReset < 0 && reset[i] > 0 {
			s.step = 0
		}
		if s.enables[s.step].LastFrame()[i] <= 0 {
			s.step = 0
		}

		s.fillPitches(i)
		s.fillGates(i, clock[i])

		s.lastClock = clock[i]
		s.lastReset = reset[i]
	}
	s.readTracker.incr()
}

func (s *stepSequence) fillPitches(i int) {
	for l := range s.steps {
		s.pitchesOut[l][i] = s.steps[l][s.step].pitch.LastFrame()[i]
	}
}

func (s *stepSequence) fillGates(i int, clock Value) {
	for l, layer := range s.gatesOut {
		for step := range layer {
			if clock > 0 && step == s.step {
				s.gatesOut[l][step][i] = 1
			} else {
				s.gatesOut[l][step][i] = -1
			}
		}
	}
}
