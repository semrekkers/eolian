package module

import (
	"buddin.us/eolian/dsp"
	"buddin.us/eolian/wav"
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

var minSpliceSize = int(dsp.Duration(10).Value())

type tape struct {
	multiOutIO

	in, play, record, reset, splice, unsplice,
	speed, bias, organize, zoom, slide, layers *In

	state                 *tapeState
	stateFunc             tapeStateFunc
	mainOut, endSpliceOut dsp.Frame
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
		in:           NewIn("input", dsp.Float64(0)),
		speed:        NewInBuffer("speed", dsp.Float64(1)),
		play:         NewInBuffer("play", dsp.Float64(1)),
		record:       NewInBuffer("record", dsp.Float64(0)),
		reset:        NewInBuffer("reset", dsp.Float64(0)),
		bias:         NewInBuffer("bias", dsp.Float64(0)),
		organize:     NewInBuffer("organize", dsp.Float64(0)),
		splice:       NewInBuffer("splice", dsp.Float64(0)),
		unsplice:     NewInBuffer("unsplice", dsp.Float64(0)),
		zoom:         NewInBuffer("zoom", dsp.Float64(0)),
		slide:        NewInBuffer("slide", dsp.Float64(0)),
		layers:       NewInBuffer("layers", dsp.Float64(1)),
		stateFunc:    tapeIdle,
		mainOut:      dsp.NewFrame(),
		endSpliceOut: dsp.NewFrame(),
	}

	if w != nil {
		samples, err := w.ReadAll()
		if err != nil {
			return nil, err
		}
		ratio := int(dsp.SampleRate / float64(w.SampleRate))
		size := len(samples) * tapeOversample * ratio
		if size > max {
			max = size
		}
		m.state = newTapeState(max)
		for _, s := range samples {
			m.state.writeToMemory(dsp.Float64(s), tapeOversample*ratio-1)
		}
		m.state.createFirstMarker()
		m.state.spliceStart = 0
		m.stateFunc = tapePlay
	} else {
		m.state = newTapeState(max * dsp.SampleRate * tapeOversample)
	}

	return m, m.Expose(
		"Tape",
		[]*In{m.in, m.speed, m.play, m.record, m.reset, m.bias, m.splice, m.organize, m.unsplice, m.zoom, m.slide, m.layers},
		[]*Out{
			{Name: "output", Provider: provideCopyOut(m, &m.mainOut)},
			{Name: "endsplice", Provider: provideCopyOut(m, &m.endSpliceOut)},
		},
	)
}

func (t *tape) Process(out dsp.Frame) {
	t.incrRead(func() {

		t.in.Process(out)

		var (
			speed    = t.speed.ProcessFrame()
			play     = t.play.ProcessFrame()
			record   = t.record.ProcessFrame()
			reset    = t.reset.ProcessFrame()
			organize = t.organize.ProcessFrame()
			splice   = t.splice.ProcessFrame()
			unsplice = t.unsplice.ProcessFrame()
			bias     = t.bias.ProcessFrame()
			zoom     = t.zoom.ProcessFrame()
			slide    = t.slide.ProcessFrame()
			layers   = t.layers.ProcessFrame()
		)

		for i := range out {
			t.state.in = out[i]
			t.state.bias = bias[i]
			t.state.organize = organize[i]
			t.state.speed = speed[i]
			t.state.play = play[i]
			t.state.record = record[i]
			t.state.reset = reset[i]
			t.state.splice = splice[i]
			t.state.layers = dsp.Clamp(layers[i], 0, 8)
			t.state.unsplice = unsplice[i]
			t.state.atSpliceEnd = false

			t.state.zoom = zoom[i]
			t.state.slide = slide[i]

			t.stateFunc = t.stateFunc(t.state)
			t.mainOut[i] = t.state.out

			t.state.lastPlay = t.state.play
			t.state.lastRecord = t.state.record
			t.state.lastReset = t.state.reset
			t.state.lastSplice = t.state.splice
			t.state.lastUnsplice = t.state.unsplice

			if t.state.atSpliceEnd {
				t.endSpliceOut[i] = 1
			} else {
				t.endSpliceOut[i] = -1
			}
		}
	})
}

type tapeState struct {
	in, out, speed, play, organize, reset,
	record, splice, unsplice, bias, layers dsp.Float64

	zoom, slide dsp.Float64

	lastPlay, lastRecord, lastReset,
	lastSplice, lastUnsplice dsp.Float64

	markers *markers
	memory  []dsp.Float64

	offset, recordingEnd   int
	unspliceHold           int
	spliceStart, spliceEnd int
	atSpliceEnd            bool
}

func newTapeState(size int) *tapeState {
	return &tapeState{
		markers:      newMarkers(),
		spliceStart:  0,
		memory:       make([]dsp.Float64, size),
		lastPlay:     -1,
		lastRecord:   -1,
		lastReset:    -1,
		lastSplice:   -1,
		lastUnsplice: -1,
	}
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
	s.memory = make([]dsp.Float64, len(s.memory))
	s.offset, s.spliceStart, s.spliceEnd, s.recordingEnd = 0, 0, 0, 0
}

func (s *tapeState) captureSpliceRange() {
	s.spliceStart, s.spliceEnd = s.markers.GetRange(s.organize)
}

func (s *tapeState) playheadTo(offset int) {
	s.captureSpliceRange()
	s.offset = offset
	s.atSpliceEnd = true
}

func (s *tapeState) playbackSpeed() dsp.Float64 {
	return dsp.Float64(tapeOversample) * s.speed
}

func (s *tapeState) recordInput() {
	in := dsp.AttenSum(s.bias, s.in, s.memory[s.offset])
	s.writeToMemory(in, int(s.playbackSpeed()))
	s.out = in
}

func (s *tapeState) writeToMemory(in dsp.Float64, oversample int) {
	for i := 0; i < oversample; i++ {
		s.memory[s.offset+i] = in
	}
	s.offset += oversample
}

func (s *tapeState) playback() {
	var (
		start      = s.markers.At(s.spliceStart)
		end        = s.markers.At(s.spliceEnd)
		grains     = 1024 >> uint(10-(s.zoom*10))
		size       = (end - start) / grains
		slide      = size * int(0.9*s.slide*dsp.Float64(grains))
		grainStart = start + slide
		grainEnd   = start + slide + size
		memoryOut  = s.memory[s.offset]
	)

	if s.zoom > 0 {
		for i := 0; i < int(s.layers); i++ {
			var (
				offset      = s.offset - grainStart
				grainOffset = size * (i + 1)
				future      = grainStart + grainOffset + offset
				atten       = 1 - 0.1*dsp.Float64(i+1)
			)
			if future < len(s.memory) && future >= 0 {
				memoryOut += s.memory[future] * atten
			}
		}
	}

	s.out = dsp.AttenSum(s.bias, s.in, memoryOut)
	s.offset += int(s.playbackSpeed())

	if s.zoom > 0 && s.offset >= grainEnd {
		s.playheadTo(grainStart)
	} else if s.zoom > 0 && s.offset <= grainStart {
		s.playheadTo(grainEnd)
	} else if s.offset >= end {
		s.playheadTo(start)
	} else if s.offset <= start {
		s.playheadTo(end)
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
		s.playheadTo(s.markers.At(s.spliceStart))
		return tapePlay
	}
	s.out = dsp.AttenSum(s.bias, s.in, 0)
	return tapeIdle
}

func leaveRecord(s *tapeState) tapeStateFunc {
	if s.play > 0 {
		return tapePlay
	}
	return tapeIdle
}

func tapeRecord(s *tapeState) tapeStateFunc {
	if s.speed <= 0 {
		return leaveRecord(s)
	}
	if next := handleReset(s); next != nil {
		return next
	}

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
		if s.unspliceHold > 2*dsp.SampleRate {
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
