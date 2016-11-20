package module

import (
	"sort"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("Loop", func(c Config) (Patcher, error) {
		var config struct {
			Max int
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Max == 0 {
			config.Max = 10
		}
		return NewLoop(config.Max)
	})
}

type Loop struct {
	IO
	in, trigger, reset, level *In
	organize, splice, erase   *In

	state     *loopState
	stateFunc loopStateFunc
}

func NewLoop(max int) (*Loop, error) {
	m := &Loop{
		in:        &In{Name: "input", Source: zero},
		trigger:   &In{Name: "trigger", Source: NewBuffer(zero)},
		reset:     &In{Name: "reset", Source: NewBuffer(zero)},
		level:     &In{Name: "level", Source: NewBuffer(Value(1))},
		organize:  &In{Name: "organize", Source: NewBuffer(zero)},
		splice:    &In{Name: "splice", Source: NewBuffer(zero)},
		erase:     &In{Name: "erase", Source: NewBuffer(zero)},
		stateFunc: loopIdle,
		state:     newLoopState(max * SampleRate),
	}
	err := m.Expose(
		[]*In{m.in, m.trigger, m.reset, m.level, m.splice, m.organize, m.erase},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *Loop) Read(out Frame) {
	reader.in.Read(out)
	trigger := reader.trigger.ReadFrame()
	reset := reader.reset.ReadFrame()
	organize := reader.organize.ReadFrame()
	splice := reader.splice.ReadFrame()
	erase := reader.erase.ReadFrame()
	level := reader.level.ReadFrame()

	for i := range out {
		reader.state.in = out[i]
		reader.state.organize = organize[i]
		reader.state.trigger = trigger[i]
		reader.state.reset = reset[i]
		reader.state.splice = splice[i]
		reader.state.erase = erase[i]

		reader.stateFunc = reader.stateFunc(reader.state)
		out[i] = reader.state.out*level[i] + out[i]

		reader.state.lastTrigger = trigger[i]
		reader.state.lastReset = reset[i]
		reader.state.lastSplice = splice[i]
	}
}

type loopState struct {
	in, out, organize, reset, trigger, splice, erase Value
	lastTrigger, lastReset, lastSplice, lastErase    Value

	splices            *splices
	memory             []Value
	start, end, offset int
}

func newLoopState(max int) *loopState {
	return &loopState{
		memory:      make([]Value, max),
		lastTrigger: -1,
		lastReset:   -1,
		lastSplice:  -1,
		lastErase:   -1,
	}
}

type loopStateFunc func(*loopState) loopStateFunc

func loopIdle(s *loopState) loopStateFunc {
	if s.lastTrigger < 0 && s.trigger > 0 {
		s.offset = 0
		s.splices = newSplices()
		return loopRecording
	}
	return loopIdle
}

func loopRecording(s *loopState) loopStateFunc {
	if s.lastTrigger < 0 && s.trigger > 0 {
		s.splices.Add(s.offset)
		s.start, s.end, s.offset = 0, 1, 0
		return loopPlayback
	}
	s.memory[s.offset] = s.in
	s.offset = (s.offset + 1) % len(s.memory)
	return loopRecording
}

func loopPlayback(s *loopState) loopStateFunc {
	if s.lastReset < 0 && s.reset > 0 {
		s.offset = 0
		return loopIdle
	}
	if s.lastSplice < 0 && s.splice > 0 {
		s.splices.Add(s.offset)
		s.splices.Sort()
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
	}
	return loopPlayback
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

func (b *splices) Sort() {
	sort.Sort(&indexSorter{b.indexes})
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
