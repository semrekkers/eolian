package module

import (
	"fmt"
	"math/rand"
)

func init() {
	Register("Sequence", func(c Config) (Patcher, error) { return NewStageSequencer() })
}

const (
	gateModeHold int = iota
	gateModeRepeat
	gateModeSingle
	gateModeRest

	patternModeSequential int = iota
	patternModePingPong
	patternModeRandom
)

type StageSequencer struct {
	IO
	clock, transpose, reset, glide, mode *In
	stages                               []stage
	slew                                 *slew

	stage, lastStage, reads int
	gateMode                int
	pulse                   int
	lastClock, lastReset    Value
	pong                    bool
}

type stage struct {
	pitch, pulses, gateMode, glide *In
}

func NewStageSequencer() (*StageSequencer, error) {
	m := &StageSequencer{
		clock:     &In{Name: "clock", Source: NewBuffer(zero)},
		transpose: &In{Name: "transpose", Source: NewBuffer(Value(1))},
		reset:     &In{Name: "reset", Source: NewBuffer(zero)},
		glide:     &In{Name: "glide", Source: NewBuffer(zero)},
		mode:      &In{Name: "mode", Source: NewBuffer(zero)},
		slew:      newSlew(),
		stages:    make([]stage, 8),
		lastClock: -1,
		lastStage: -1,
	}

	inputs := []*In{m.clock, m.transpose, m.reset, m.glide, m.mode}

	for i := 0; i < 8; i++ {
		m.stages[i] = stage{
			pitch: &In{
				Name:   fmt.Sprintf("%d.pitch", i),
				Source: NewBuffer(zero),
			},
			pulses: &In{
				Name:   fmt.Sprintf("%d.pulses", i),
				Source: NewBuffer(Value(1)),
			},
			gateMode: &In{
				Name:   fmt.Sprintf("%d.mode", i),
				Source: NewBuffer(Value(1)),
			},
			glide: &In{
				Name:   fmt.Sprintf("%d.glide", i),
				Source: NewBuffer(Value(0)),
			},
		}
		inputs = append(inputs,
			m.stages[i].pitch,
			m.stages[i].pulses,
			m.stages[i].gateMode,
			m.stages[i].glide)
	}

	err := m.Expose(inputs,
		[]*Out{
			{
				Name:     "pitch",
				Provider: ReaderProviderFunc(func() Reader { return &stageSeqPitch{StageSequencer: m} }),
			},
			{
				Name:     "gate",
				Provider: ReaderProviderFunc(func() Reader { return &stageSeqGate{m} }),
			},
			{
				Name:     "sync",
				Provider: ReaderProviderFunc(func() Reader { return &stageSeqSync{m} }),
			},
		},
	)
	return m, err
}

func (s *StageSequencer) read(out Frame) {
	if s.reads == 0 {
		clock := s.clock.ReadFrame()
		reset := s.reset.ReadFrame()
		mode := s.mode.ReadFrame()
		s.transpose.ReadFrame()

		for i := 0; i < 8; i++ {
			s.stages[i].pitch.ReadFrame()
			s.stages[i].pulses.ReadFrame()
			s.stages[i].gateMode.ReadFrame()
			s.stages[i].glide.ReadFrame()
		}

		for i := range out {
			clock := clock[i]
			reset := reset[i]
			mode := mode[i]

			// Detect rising edge of clock and reset pulses
			if s.lastClock < 0 && clock > 0 {
				pulses := s.stages[s.stage].pulses.LastFrame()[i]
				gateMode := mapGateMode(s.stages[s.stage].gateMode.LastFrame()[i])

				s.pulse = (s.pulse + 1) % int(pulses)
				if s.pulse == 0 {
					s.lastStage = s.stage
					s.advanceStage(mode)
				}
				s.gateMode = gateMode
			}
			if s.lastReset < 0 && reset > 0 {
				s.pulse = 0
				s.stage = 0
			}

			s.lastClock = clock
			s.lastReset = reset
		}
	}
	if outs := s.OutputsActive(); outs > 0 {
		s.reads = (s.reads + 1) % outs
	}
}

func (s *StageSequencer) advanceStage(mode Value) {
	switch mapPatternMode(mode) {
	case patternModeSequential:
		s.stage = (s.stage + 1) % len(s.stages)
		s.pong = false
	case patternModePingPong:
		var inc = 1
		if s.pong {
			inc = -1
		}
		s.stage += inc

		if s.stage == len(s.stages)-1 {
			s.stage = len(s.stages) - 1
			s.pong = true
		} else if s.stage == 0 {
			s.stage = 0
			s.pong = false
		}
	case patternModeRandom:
		s.stage = rand.Intn(len(s.stages))
		s.pong = false
	}
}

type stageSeqPitch struct {
	*StageSequencer
}

func (reader *stageSeqPitch) Read(out Frame) {
	reader.read(out)
	glide := reader.glide.ReadFrame()
	for i := range out {
		stage := reader.stages[reader.stage]
		in := stage.pitch.LastFrame()[i] * reader.transpose.LastFrame()[i]
		var amt Value
		if stage.glide.LastFrame()[i] > 0 {
			amt = glide[i]
		}
		out[i] = reader.slew.Tick(in, amt, amt)
	}
}

type stageSeqGate struct {
	*StageSequencer
}

func (reader *stageSeqGate) Read(out Frame) {
	reader.read(out)
	for i := range out {
		clock := reader.clock.LastFrame()[i]

		if reader.stages[reader.stage].pitch.LastFrame()[i] == 0 {
			out[i] = -1
		} else {
			switch reader.gateMode {
			case gateModeHold:
				if clock > 0 {
					out[i] = 1
				} else {
					if reader.stage == reader.lastStage {
						out[i] = 1
					} else {
						out[i] = -1
					}
				}
			case gateModeRepeat:
				if clock > 0 {
					out[i] = 1
				} else {
					out[i] = -1
				}
			case gateModeSingle:
				if reader.pulse == 0 && clock > 0 {
					out[i] = 1
				} else {
					out[i] = -1
				}
			case gateModeRest:
				out[i] = -1
			}
		}
		reader.lastStage = reader.stage
	}
}

type stageSeqSync struct {
	*StageSequencer
}

func (reader *stageSeqSync) Read(out Frame) {
	reader.read(out)
	for i := range out {
		if reader.stage == 0 && reader.pulse == 0 {
			out[i] = 1
		} else {
			out[i] = -1
		}
	}
}

func mapPatternMode(v Value) int {
	switch v {
	case 0:
		return patternModeSequential
	case 1:
		return patternModePingPong
	case 2:
		return patternModeRandom
	default:
		return patternModeRandom
	}
}

func mapGateMode(v Value) int {
	switch v {
	case 0:
		return gateModeRest
	case 1:
		return gateModeSingle
	case 2:
		return gateModeRepeat
	case 3:
		return gateModeHold
	default:
		return gateModeHold
	}
}
