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
	in, play, record, reset, bias *In
	organize, splice, unsplice    *In

	state     *tapeState
	stateFunc tapeStateFunc
	reads     int

	endOfSplice Frame
}

func NewTape(max int) (*Tape, error) {
	m := &Tape{
		in:          &In{Name: "input", Source: zero},
		play:        &In{Name: "play", Source: NewBuffer(Value(1))},
		record:      &In{Name: "record", Source: NewBuffer(zero)},
		reset:       &In{Name: "reset", Source: NewBuffer(zero)},
		bias:        &In{Name: "bias", Source: NewBuffer(zero)},
		organize:    &In{Name: "organize", Source: NewBuffer(zero)},
		splice:      &In{Name: "splice", Source: NewBuffer(zero)},
		unsplice:    &In{Name: "unsplice", Source: NewBuffer(zero)},
		stateFunc:   tapeIdle,
		state:       newTapeState(max * SampleRate),
		endOfSplice: make(Frame, FrameSize),
	}
	err := m.Expose(
		[]*In{m.in, m.play, m.record, m.reset, m.bias, m.splice, m.organize, m.unsplice},
		[]*Out{
			{Name: "output", Provider: Provide(&tapeOut{Tape: m})},
			{Name: "endsplice", Provider: Provide(&tapeEndOfSplice{Tape: m})},
		},
	)
	return m, err
}

func (t *Tape) read(out Frame) {
	if t.reads == 0 {
		t.in.Read(out)
		t.play.ReadFrame()
		t.record.ReadFrame()
		t.reset.ReadFrame()
		t.organize.ReadFrame()
		t.splice.ReadFrame()
		t.unsplice.ReadFrame()
		t.bias.ReadFrame()
	}

	if outs := t.OutputsActive(); outs > 0 {
		t.reads = (t.reads + 1) % outs
	}
}

type tapeOut struct {
	*Tape
}

func (o *tapeOut) Read(out Frame) {
	o.read(out)
	for i := range out {
		o.state.in = out[i]
		o.state.organize = o.organize.LastFrame()[i]
		o.state.play = o.play.LastFrame()[i]
		o.state.record = o.record.LastFrame()[i]
		o.state.reset = o.reset.LastFrame()[i]
		o.state.splice = o.splice.LastFrame()[i]
		o.state.unsplice = o.unsplice.LastFrame()[i]
		o.state.atSpliceEnd = false

		o.stateFunc = o.stateFunc(o.state)
		bias := o.bias.LastFrame()

		if bias[i] > 0 {
			out[i] = (1-bias[i])*out[i] + o.state.out
		} else if bias[i] < 0 {
			out[i] = out[i] + (1+bias[i])*o.state.out
		} else {
			out[i] = out[i] + o.state.out
		}

		o.state.lastPlay = o.state.play
		o.state.lastRecord = o.state.record
		o.state.lastReset = o.state.reset
		o.state.lastSplice = o.state.splice
		o.state.lastUnsplice = o.state.unsplice
		if o.state.atSpliceEnd {
			o.endOfSplice[i] = 1
		} else {
			o.endOfSplice[i] = -1
		}
	}
}

type tapeEndOfSplice struct {
	*Tape
}

func (o *tapeEndOfSplice) Read(out Frame) {
	for i := range out {
		out[i] = o.endOfSplice[i]
	}
}

type tapeState struct {
	in, out, play, organize, reset, record, splice, unsplice  Value
	lastPlay, lastRecord, lastReset, lastSplice, lastUnsplice Value

	markers *markers
	memory  []Value

	offset, recordingEnd   int
	unspliceHold           int
	spliceStart, spliceEnd int
	atSpliceEnd            bool
}

func newTapeState(max int) *tapeState {
	return &tapeState{
		markers:      newMarkers(),
		spliceStart:  0,
		memory:       make([]Value, max),
		lastPlay:     -1,
		lastRecord:   -1,
		lastReset:    -1,
		lastSplice:   -1,
		lastUnsplice: -1,
	}
}

func (s *tapeState) mark() {
	s.markers.Create(s.offset)
	s.spliceStart, s.spliceEnd = s.markers.GetRange(s.organize)
	s.offset = s.markers.At(s.spliceStart)
}

func (s *tapeState) removeMark() {
	s.unspliceHold = 0
	s.markers.Erase(s.spliceEnd)
	s.spliceStart, s.spliceEnd = s.markers.GetRange(s.organize)
}

func (s *tapeState) clearMarkers() {
	s.markers = newMarkers()
	s.markers.Create(s.recordingEnd)
	s.offset, s.spliceStart, s.spliceEnd = 0, 0, 1
}

func (s *tapeState) erase() {
	s.markers = newMarkers()
	s.memory = make([]Value, len(s.memory))
	s.offset, s.spliceStart, s.spliceEnd, s.recordingEnd = 0, 0, 0, 0
}

func (s *tapeState) resetPlayhead() {
	s.spliceStart, s.spliceEnd = s.markers.GetRange(s.organize)
	s.offset = s.markers.At(s.spliceStart)
}

func (s *tapeState) playback() {
	s.out = s.memory[s.offset]
	s.offset++
	if s.offset >= s.markers.At(s.spliceEnd) {
		s.resetPlayhead()
		s.atSpliceEnd = true
	}
}

type tapeStateFunc func(*tapeState) tapeStateFunc

func tapeIdle(s *tapeState) tapeStateFunc {
	handleUnsplice(s)
	handleReset(s)
	if next := handleRecord(s); next != nil {
		return next
	}
	if s.recordingEnd != 0 && s.play > 0 {
		s.resetPlayhead()
		return tapePlay
	}
	return tapeIdle
}

func leaveRecord(s *tapeState) tapeStateFunc {
	if s.play > 0 {
		return tapePlay
	}
	return tapeIdle
}

func tapeRecord(s *tapeState) tapeStateFunc {
	if s.lastRecord < 0 && s.record > 0 {
		if s.markers.Count() == 1 {
			s.recordingEnd = s.offset
			s.markers.Create(s.offset)
			s.spliceEnd = 1
		}
		s.offset = s.spliceStart
		return leaveRecord(s)
	}
	s.memory[s.offset] = s.in
	s.offset++

	if s.markers.Count() == 1 {
		if s.offset >= len(s.memory) {
			s.offset = 0
		}
	} else if s.offset >= s.markers.At(s.spliceEnd) {
		s.offset = s.markers.At(s.spliceStart)
		return leaveRecord(s)
	}
	return tapeRecord
}

func tapePlay(s *tapeState) tapeStateFunc {
	handleUnsplice(s)
	if next := handleRecord(s); next != nil {
		return next
	}
	if next := handleReset(s); next != nil {
		return next
	}
	if s.lastSplice < 0 && s.splice > 0 {
		s.mark()
	}
	s.playback()
	if s.atSpliceEnd && s.play < 0 {
		return tapeIdle
	}
	return tapePlay
}

func handleReset(s *tapeState) tapeStateFunc {
	if s.lastReset < 0 && s.reset > 0 {
		s.erase()
		return tapeIdle
	}
	return nil
}

func handleUnsplice(s *tapeState) {
	if s.unsplice > 0 {
		if s.unspliceHold > 2*SampleRate {
			s.unspliceHold = 0
			s.clearMarkers()
		}
		s.unspliceHold++
	}
	if s.lastUnsplice > 0 && s.unsplice < 0 {
		s.unspliceHold = 0
		s.removeMark()
	}
}

func handleRecord(s *tapeState) tapeStateFunc {
	if s.lastRecord < 0 && s.record > 0 {
		s.offset = s.markers.At(s.spliceStart)
		return tapeRecord
	}
	return nil
}

func newMarkers() *markers {
	return &markers{
		indexes: []int{0},
	}
}

type markers struct {
	indexes []int
}

func (b *markers) Create(i int) {
	b.indexes = append(b.indexes, i)
	sort.Sort(&indexSorter{b.indexes})
}

func (b *markers) Count() int {
	return len(b.indexes)
}

func (b *markers) At(i int) int {
	return b.indexes[i]
}

func (b *markers) Erase(end int) {
	if end == len(b.indexes)-1 {
		return
	}
	b.indexes = append(b.indexes[:end], b.indexes[end+1:]...)
}

func (b *markers) GetRange(organize Value) (int, int) {
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
