package module

import (
	"github.com/brettbuddin/eolian/wav"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("Tape", func(c Config) (Patcher, error) {
		var config struct {
			Max  int
			File string
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Max == 0 {
			config.Max = 10
		}
		return newTape(config.Max, config.File)
	})
}

const tapeOversample = 20

var minSpliceSize = int(Duration(10).Value())

type tape struct {
	IO
	in, speed, play, record, reset, bias *In
	organize, splice, unsplice           *In

	state     *tapeState
	stateFunc tapeStateFunc
	reads     int

	endOfSplice Frame
}

func newTape(max int, file string) (*tape, error) {
	var w *wav.Wav
	if file != "" {
		var err error
		w, err = wav.Open(file)
		if err != nil {
			return nil, err
		}
		defer w.Close()
	}

	m := &tape{
		in:          &In{Name: "input", Source: zero},
		speed:       &In{Name: "speed", Source: NewBuffer(Value(1))},
		play:        &In{Name: "play", Source: NewBuffer(Value(1))},
		record:      &In{Name: "record", Source: NewBuffer(zero)},
		reset:       &In{Name: "reset", Source: NewBuffer(zero)},
		bias:        &In{Name: "bias", Source: NewBuffer(zero)},
		organize:    &In{Name: "organize", Source: NewBuffer(zero)},
		splice:      &In{Name: "splice", Source: NewBuffer(zero)},
		unsplice:    &In{Name: "unsplice", Source: NewBuffer(zero)},
		stateFunc:   tapeIdle,
		endOfSplice: make(Frame, FrameSize),
	}

	if w != nil {
		samples, err := w.ReadAll()
		if err != nil {
			return nil, err
		}
		ratio := int(SampleRate / float64(w.SampleRate))
		size := len(samples) * tapeOversample * ratio
		if size > max {
			max = size
		}
		m.state = newTapeState(max)
		for _, s := range samples {
			m.state.writeToMemory(Value(s), tapeOversample*ratio-1)
		}
		m.state.createFirstMarker()
		m.state.spliceStart = 0
		m.stateFunc = tapePlay
	} else {
		m.state = newTapeState(max * SampleRate * tapeOversample)
	}

	return m, m.Expose(
		"Tape",
		[]*In{m.in, m.speed, m.play, m.record, m.reset, m.bias, m.splice, m.organize, m.unsplice},
		[]*Out{
			{Name: "output", Provider: Provide(&tapeOut{tape: m})},
			{Name: "endsplice", Provider: Provide(&tapeEndOfSplice{tape: m})},
		},
	)
}

func (t *tape) read(out Frame) {
	if t.reads == 0 {
		t.in.Read(out)
		t.speed.ReadFrame()
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
	*tape
}

func (o *tapeOut) Read(out Frame) {
	o.read(out)
	for i := range out {
		o.state.in = out[i]
		o.state.bias = o.bias.LastFrame()[i]
		o.state.organize = o.organize.LastFrame()[i]
		o.state.speed = o.speed.LastFrame()[i]
		o.state.play = o.play.LastFrame()[i]
		o.state.record = o.record.LastFrame()[i]
		o.state.reset = o.reset.LastFrame()[i]
		o.state.splice = o.splice.LastFrame()[i]
		o.state.unsplice = o.unsplice.LastFrame()[i]
		o.state.atSpliceEnd = false

		o.stateFunc = o.stateFunc(o.state)
		out[i] = o.state.out

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
	*tape
}

func (o *tapeEndOfSplice) Read(out Frame) {
	o.read(out)
	for i := range out {
		out[i] = o.endOfSplice[i]
	}
}

type tapeState struct {
	in, out, speed, play, organize, reset, record, splice, unsplice, bias Value
	lastPlay, lastRecord, lastReset, lastSplice, lastUnsplice             Value

	markers *markers
	memory  []Value

	offset, recordingEnd   int
	unspliceHold           int
	spliceStart, spliceEnd int
	atSpliceEnd            bool
}

func newTapeState(size int) *tapeState {
	return &tapeState{
		markers:      newMarkers(),
		spliceStart:  0,
		memory:       make([]Value, size),
		lastPlay:     -1,
		lastRecord:   -1,
		lastReset:    -1,
		lastSplice:   -1,
		lastUnsplice: -1,
	}
}

func (s *tapeState) crossfade(live, record Value) Value {
	if s.bias > 0 {
		return (1-s.bias)*live + record
	} else if s.bias < 0 {
		return live + (1+s.bias)*record
	}
	return live + record
}

func (s *tapeState) mark() {
	// Prohibit creating splices less than 10ms in length
	start, end := s.markers.At(s.spliceStart), s.markers.At(s.spliceEnd)
	if s.offset-start < minSpliceSize || end-s.offset < minSpliceSize {
		return
	}
	s.markers.Create(s.offset)
	s.spliceStart, s.spliceEnd = s.markers.GetRange(s.organize)
	s.offset = s.markers.At(s.spliceStart)
}

func (s *tapeState) unmark() {
	s.unspliceHold = 0
	s.markers.Erase(s.spliceEnd)
	s.spliceStart, s.spliceEnd = s.markers.GetRange(s.organize)
}

func (s *tapeState) createFirstMarker() {
	s.recordingEnd = s.offset
	s.markers.Create(s.offset)
	s.spliceEnd = 1
}

func (s *tapeState) clearMarkers() {
	s.markers = newMarkers()
	s.markers.Create(s.recordingEnd)
	s.offset, s.spliceStart, s.spliceEnd = 0, 0, 1
}

func (s *tapeState) clearAll() {
	s.markers = newMarkers()
	s.memory = make([]Value, len(s.memory))
	s.offset, s.spliceStart, s.spliceEnd, s.recordingEnd = 0, 0, 0, 0
}

func (s *tapeState) playheadToStart() {
	s.spliceStart, s.spliceEnd = s.markers.GetRange(s.organize)
	s.offset = s.markers.At(s.spliceStart)
}

func (s *tapeState) playheadToEnd() {
	s.spliceStart, s.spliceEnd = s.markers.GetRange(s.organize)
	s.offset = s.markers.At(s.spliceEnd)
}

func (s *tapeState) playbackSpeed() Value {
	return Value(tapeOversample) * s.speed
}

func (s *tapeState) recordInput() {
	in := s.crossfade(s.in, s.memory[s.offset])
	s.writeToMemory(in, int(s.playbackSpeed()))
	s.out = in
}

func (s *tapeState) writeToMemory(in Value, oversample int) {
	for i := 0; i < oversample; i++ {
		s.memory[s.offset+i] = in
	}
	s.offset += oversample
}

func (s *tapeState) playback() {
	s.out = s.crossfade(s.in, s.memory[s.offset])
	s.offset += int(s.playbackSpeed())

	// Loop around (depending on which direction we are moving)
	if s.offset >= s.markers.At(s.spliceEnd) {
		s.playheadToStart()
		s.atSpliceEnd = true
	} else if s.offset <= s.markers.At(s.spliceStart) {
		s.playheadToEnd()
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
		s.playheadToStart()
		return tapePlay
	}
	s.out = s.crossfade(s.in, 0)
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
		// End of recording creates the first splice
		if s.markers.Count() == 1 {
			s.createFirstMarker()
		}
		s.offset = s.spliceStart
		return leaveRecord(s)
	}

	s.recordInput()

	// When we have no splices, use the end of the tape to wrap us; otherwise use the splice range
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
		s.clearAll()
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
		s.unmark()
	}
}

func handleRecord(s *tapeState) tapeStateFunc {
	if s.speed < 0 {
		return nil
	}
	if s.lastRecord < 0 && s.record > 0 {
		s.offset = s.markers.At(s.spliceStart)
		return tapeRecord
	}
	return nil
}
