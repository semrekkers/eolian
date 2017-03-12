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
	multiOutIO

	clock, reset                          *In
	enables                               []*In
	pitches                               [][]*In
	step, lastStep, layerCount, stepCount int

	lastClock, lastReset Value

	pitchesOut, gatesOut []Frame
}

func newStepSequence(steps, layers int) (*stepSequence, error) {
	m := &stepSequence{
		clock:      &In{Name: "clock", Source: NewBuffer(zero)},
		reset:      &In{Name: "reset", Source: NewBuffer(zero)},
		pitchesOut: make([]Frame, layers),
		pitches:    make([][]*In, layers),
		enables:    make([]*In, steps),
		gatesOut:   make([]Frame, steps),
		lastClock:  -1,
		lastReset:  -1,
		lastStep:   -1,
		stepCount:  steps,
		layerCount: layers,
	}

	var (
		inputs  = []*In{m.clock, m.reset}
		outputs = []*Out{}
	)

	for i := 0; i < layers; i++ {
		m.pitches[i] = make([]*In, steps)
		for j := 0; j < steps; j++ {
			m.pitches[i][j] = &In{
				Name:   fmt.Sprintf("%c/%d/pitch", alphaSeries[i], j),
				Source: NewBuffer(zero),
			}
			inputs = append(inputs, m.pitches[i][j])
		}
		m.pitchesOut[i] = make(Frame, FrameSize)
		outputs = append(outputs, &Out{
			Name:     fmt.Sprintf("%c/pitch", alphaSeries[i]),
			Provider: provideCopyOut(m, &m.pitchesOut[i]),
		})
	}

	for i := 0; i < steps; i++ {
		m.enables[i] = &In{
			Name:   fmt.Sprintf("%d/enabled", i),
			Source: NewBuffer(Value(1)),
		}
		inputs = append(inputs, m.enables[i])

		m.gatesOut[i] = make(Frame, FrameSize)
		outputs = append(outputs, &Out{
			Name:     fmt.Sprintf("%d/gate", i),
			Provider: provideCopyOut(m, &m.gatesOut[i]),
		})
	}

	return m, m.Expose("StepSequence", inputs, outputs)
}

func (s *stepSequence) Read(out Frame) {
	s.incrRead(func() {
		clock := s.clock.ReadFrame()
		reset := s.reset.ReadFrame()
		for l, layer := range s.pitches {
			for i := range layer {
				s.pitches[l][i].ReadFrame()
			}
		}
		for i := 0; i < s.stepCount; i++ {
			s.enables[i].ReadFrame()
		}

		for i := range out {
			if s.lastStep >= 0 && s.lastClock < 0 && clock[i] > 0 {
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
			s.lastStep = s.step
		}
	})
}

func (s *stepSequence) fillPitches(i int) {
	for l := range s.pitches {
		s.pitchesOut[l][i] = s.pitches[l][s.step].LastFrame()[i]
	}
}

func (s *stepSequence) fillGates(i int, clock Value) {
	for j := range s.gatesOut {
		if clock > 0 && j == s.step {
			s.gatesOut[j][i] = 1
		} else {
			s.gatesOut[j][i] = -1
		}
	}
}
