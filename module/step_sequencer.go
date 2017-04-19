package module

import (
	"fmt"
	"math/rand"

	"buddin.us/eolian/dsp"

	"github.com/mitchellh/mapstructure"
)

const alphaSeries = "abcdefghijklmnopqrstuvwxyz"

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

	clock, reset, mode                    *In
	enables                               []*In
	pitches                               [][]*In
	step, lastStep, layerCount, stepCount int
	pong                                  bool

	lastClock, lastReset dsp.Float64

	pitchesOut, gatesOut []dsp.Frame
	allGateOut           dsp.Frame
}

func newStepSequence(steps, layers int) (*stepSequence, error) {
	m := &stepSequence{
		clock:      NewInBuffer("clock", dsp.Float64(0)),
		reset:      NewInBuffer("reset", dsp.Float64(0)),
		mode:       NewInBuffer("mode", dsp.Float64(0)),
		pitches:    make([][]*In, layers),
		enables:    make([]*In, steps),
		gatesOut:   make([]dsp.Frame, steps),
		pitchesOut: make([]dsp.Frame, layers),
		allGateOut: dsp.NewFrame(),
		lastClock:  -1,
		lastReset:  -1,
		lastStep:   -1,
		stepCount:  steps,
		layerCount: layers,
	}

	var (
		inputs  = []*In{m.clock, m.reset, m.mode}
		outputs = []*Out{&Out{Name: "gate", Provider: provideCopyOut(m, &m.allGateOut)}}
	)

	for i := 0; i < layers; i++ {
		// Layer Steps
		m.pitches[i] = make([]*In, steps)
		for j := 0; j < steps; j++ {
			m.pitches[i][j] = NewInBuffer(fmt.Sprintf("%c/%d/pitch", alphaSeries[i], j), dsp.Float64(0))
			inputs = append(inputs, m.pitches[i][j])
		}

		// Layer Pitch
		m.pitchesOut[i] = dsp.NewFrame()
		outputs = append(outputs, &Out{
			Name:     fmt.Sprintf("%c/pitch", alphaSeries[i]),
			Provider: provideCopyOut(m, &m.pitchesOut[i]),
		})
	}

	for i := 0; i < steps; i++ {
		m.enables[i] = NewInBuffer(fmt.Sprintf("%d/enabled", i), dsp.Float64(1))
		inputs = append(inputs, m.enables[i])

		// Step Gate
		m.gatesOut[i] = dsp.NewFrame()
		outputs = append(outputs, &Out{
			Name:     fmt.Sprintf("%d/gate", i),
			Provider: provideCopyOut(m, &m.gatesOut[i]),
		})
	}

	return m, m.Expose("StepSequence", inputs, outputs)
}

func (s *stepSequence) Process(out dsp.Frame) {
	s.incrRead(func() {
		clock := s.clock.ProcessFrame()
		reset := s.reset.ProcessFrame()
		mode := s.mode.ProcessFrame()
		for l, layer := range s.pitches {
			for i := range layer {
				s.pitches[l][i].ProcessFrame()
			}
		}
		for i := 0; i < s.stepCount; i++ {
			s.enables[i].ProcessFrame()
		}

		for i := range out {
			if s.lastStep >= 0 && s.lastClock < 0 && clock[i] > 0 {
				switch mapPatternMode(mode[i]) {
				case patternModeSequential:
					s.step = (s.step + 1) % s.stepCount
					s.pong = false
				case patternModePingPong:
					if s.pong {
						s.step -= 1
					} else {
						s.step += 1
					}
					if s.step == s.stepCount-1 {
						s.step = s.stepCount - 1
						s.pong = true
					} else if s.step == 0 {
						s.step = 0
						s.pong = false
					}
				case patternModeRandom:
					s.step = rand.Intn(s.stepCount)
					s.pong = false
				}
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

func (s *stepSequence) fillGates(i int, clock dsp.Float64) {
	for j := range s.gatesOut {
		if clock > 0 && j == s.step {
			s.gatesOut[j][i] = 1
		} else {
			s.gatesOut[j][i] = -1
		}
		s.allGateOut[i] = dsp.Max(s.allGateOut[i], s.gatesOut[j][i])
	}
}
