package module

import (
	"sort"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("Tape", func(c Config) (Patcher, error) {
		var config struct {
			Max int
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Max == 0 {
			config.Max = 10
		}
		return NewTape(config.Max)
	})
}

type Tape struct {
	IO
	in, trigger, reset, bias *In
	organize, splice, erase  *In

	state     *tapeState
	stateFunc tapeStateFunc
	reads     int

	endOfSplice Frame
}

func NewTape(max int) (*Tape, error) {
	m := &Tape{
		in:          &In{Name: "input", Source: zero},
		trigger:     &In{Name: "trigger", Source: NewBuffer(zero)},
		reset:       &In{Name: "reset", Source: NewBuffer(zero)},
		bias:        &In{Name: "bias", Source: NewBuffer(zero)},
		organize:    &In{Name: "organize", Source: NewBuffer(zero)},
		splice:      &In{Name: "splice", Source: NewBuffer(zero)},
		erase:       &In{Name: "erase", Source: NewBuffer(zero)},
		stateFunc:   tapeIdle,
		state:       newTapeState(max * SampleRate),
		endOfSplice: make(Frame, FrameSize),
	}
	err := m.Expose(
		[]*In{m.in, m.trigger, m.reset, m.bias, m.splice, m.organize, m.erase},
		[]*Out{
			{Name: "output", Provider: Provide(&tapeOut{Tape: m})},
			{Name: "endOfSplice", Provider: Provide(&tapeEndOfSplice{Tape: m})},
		},
	)
	return m, err
}

func (t *Tape) read(out Frame) {
	if t.reads == 0 {
		t.in.Read(out)
		t.trigger.ReadFrame()
		t.reset.ReadFrame()
		t.organize.ReadFrame()
		t.splice.ReadFrame()
		t.erase.ReadFrame()
		t.bias.ReadFrame()
	}

	if outs := t.OutputsActive(); outs > 0 {
		t.reads = (t.reads + 1) % outs
	}
}

type tapeOut struct {
	*Tape
}

func (reader *tapeOut) Read(out Frame) {
	reader.read(out)
	for i := range out {
		reader.state.in = out[i]
		reader.state.organize = reader.organize.LastFrame()[i]
		reader.state.trigger = reader.trigger.LastFrame()[i]
		reader.state.reset = reader.reset.LastFrame()[i]
		reader.state.splice = reader.splice.LastFrame()[i]
		reader.state.erase = reader.erase.LastFrame()[i]
		reader.state.endOfSplice = false

		reader.stateFunc = reader.stateFunc(reader.state)
		bias := reader.bias.LastFrame()

		if bias[i] > 0 {
			out[i] = (1-bias[i])*out[i] + reader.state.out
		} else if bias[i] < 0 {
			out[i] = out[i] + (1+bias[i])*reader.state.out
		} else {
			out[i] = out[i] + reader.state.out
		}

		reader.state.lastTrigger = reader.state.trigger
		reader.state.lastReset = reader.state.reset
		reader.state.lastSplice = reader.state.splice
		reader.state.lastErase = reader.state.erase
		if reader.state.endOfSplice {
			reader.endOfSplice[i] = 1
		} else {
			reader.endOfSplice[i] = -1
		}
	}
}

type tapeEndOfSplice struct {
	*Tape
}

func (reader *tapeEndOfSplice) Read(out Frame) {
	for i := range out {
		out[i] = reader.endOfSplice[i]
	}
}

type tapeState struct {
	in, out, organize, reset, trigger, splice, erase Value
	lastTrigger, lastReset, lastSplice, lastErase    Value
	endOfSplice                                      bool

	splices            *splices
	memory             []Value
	start, end, offset int
}

func newTapeState(max int) *tapeState {
	return &tapeState{
		splices:     newSplices(),
		start:       0,
		memory:      make([]Value, max),
		lastTrigger: -1,
		lastReset:   -1,
		lastSplice:  -1,
		lastErase:   -1,
	}
}

type tapeStateFunc func(*tapeState) tapeStateFunc

func tapeIdle(s *tapeState) tapeStateFunc {
	if fn := handleTrigger(s); fn != nil {
		return fn
	}
	return tapeIdle
}

func tapeRecording(s *tapeState) tapeStateFunc {
	if s.lastTrigger < 0 && s.trigger > 0 {
		if s.splices.Count() == 0 {
			s.splices.Add(s.offset)
			s.end = 1
		}
		s.offset = s.start
		return tapePlayback
	}
	s.memory[s.offset] = s.in

	if s.splices.Count() == 0 {
		s.offset = (s.offset + 1) % len(s.memory)
	} else {
		s.offset++
		if s.offset >= s.splices.At(s.end) {
			s.offset = s.splices.At(s.start)
			return tapePlayback
		}
	}
	return tapeRecording
}

func tapePlayback(s *tapeState) tapeStateFunc {
	if fn := handleTrigger(s); fn != nil {
		return fn
	}
	if s.lastReset < 0 && s.reset > 0 {
		s.splices = newSplices()
		s.memory = make([]Value, len(s.memory))
		s.start = 0
		return tapeIdle
	}
	if s.lastSplice < 0 && s.splice > 0 {
		s.splices.Add(s.offset)
		s.start, s.end = s.splices.GetRange(s.organize)
		s.offset = s.splices.At(s.start)
	}
	if s.lastErase < 0 && s.erase > 0 {
		s.splices.Erase(s.end)
		s.start, s.end = s.splices.GetRange(s.organize)
	}

	s.out = s.memory[s.offset]
	s.offset++
	if s.offset >= s.splices.At(s.end) {
		s.start, s.end = s.splices.GetRange(s.organize)
		s.offset = s.splices.At(s.start)
		s.endOfSplice = true
	}
	return tapePlayback
}

func handleTrigger(s *tapeState) tapeStateFunc {
	if s.lastTrigger < 0 && s.trigger > 0 {
		s.offset = s.splices.At(s.start)
		return tapeRecording
	}
	return nil
}

func newSplices() *splices {
	return &splices{
		indexes: []int{0},
	}
}

type splices struct {
	indexes []int
}

func (b *splices) Add(i int) {
	b.indexes = append(b.indexes, i)
	sort.Sort(&indexSorter{b.indexes})
}

func (b *splices) Count() int {
	return len(b.indexes) - 1
}

func (b *splices) At(i int) int {
	return b.indexes[i]
}

func (b *splices) Erase(end int) {
	if end == len(b.indexes)-1 {
		return
	}
	b.indexes = append(b.indexes[:end], b.indexes[end+1:]...)
}

func (b *splices) GetRange(organize Value) (int, int) {
	size := len(b.indexes)
	if size == 2 {
		return 0, size - 1
	}
	zoneSize := 1 / float64(size-1)
	start := minInt(size-2, int(float64(organize)/zoneSize))
	end := minInt(size-1, start+1)
	return start, end
}

type indexSorter struct {
	indexes []int
}

func (s *indexSorter) Len() int           { return len(s.indexes) }
func (s *indexSorter) Less(i, j int) bool { return s.indexes[i] < s.indexes[j] }
func (s *indexSorter) Swap(i, j int) {
	s.indexes[i], s.indexes[j] = s.indexes[j], s.indexes[i]
}
