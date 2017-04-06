package dsp

import (
	"fmt"
	"strconv"

	"buddin.us/musictheory"
)

// SampleRate is the numebr of samples computed every second
const SampleRate = 44100.0

// FrameSize is the size of the audio buffer
var FrameSize = 512

// Float64 is a real value sinked to the sound card
type Float64 float64

// Process reads a constant value into a Frame
func (v Float64) Process(out Frame) {
	for i := range out {
		out[i] = v
	}
}

// Value returns the constant value
func (v Float64) Value() Float64 {
	return v
}

func (v Float64) String() string {
	return fmt.Sprintf("%.2f", v)
}

// Valuer is the wrapper interface around the Value method; which is used in obtaining the constant value
type Valuer interface {
	Value() Float64
}

// Frame is a block of Values to be sinked to the soundcard
type Frame []Float64

// NewFrame returns a new Frame sized to the audio buffer size (FrameSize)
func NewFrame() Frame {
	return make(Frame, FrameSize)
}

// Hz represents cycles-per-second
type Hz struct {
	Valuer
	Raw float64
}

// Frequency returns a scalar value in Hz
func Frequency(v float64) Hz {
	return Hz{Raw: v, Valuer: Float64(v / SampleRate)}
}

func (hz Hz) String() string {
	return fmt.Sprintf("%.2fHz", hz.Raw)
}

// Process reads the Hz real value to a Frame
func (hz Hz) Process(out Frame) {
	hz.Value().Process(out)
}

// ParsePitch parses the scientific notation of a pitch
func ParsePitch(v string) (Pitch, error) {
	p, err := musictheory.ParsePitch(v)
	if err != nil {
		return Pitch{}, err
	}
	return Pitch{Raw: v, Valuer: Frequency(p.Freq())}, nil
}

// Pitch is a pitch that has been expressed in scientific notation
type Pitch struct {
	Valuer
	Raw string
}

func (p Pitch) String() string {
	return p.Raw
}

// Process reads the Pitch real value to a Frame
func (p Pitch) Process(out Frame) {
	p.Value().Process(out)
}

// MS is a value representation of milliseconds
type MS struct {
	Valuer
	Raw float64
}

// DurationInt returns a scalar value (int) in MS
func DurationInt(v int) MS {
	return Duration(float64(v))
}

// Duration returns a scalar value (float64) in MS
func Duration(v float64) MS {
	return MS{Raw: v, Valuer: Float64(v * SampleRate * 0.001)}
}

func (ms MS) String() string {
	return fmt.Sprintf("%.2fms", ms.Raw)
}

// Process reads the real value to a Frame
func (ms MS) Process(out Frame) {
	ms.Value().Process(out)
}

// ParseValueString parses string representations of integers, floats and Pitches
func ParseValueString(value string) (Valuer, error) {
	if v, err := strconv.ParseInt(value, 10, 64); err == nil {
		return Float64(v), nil
	} else if v, err := strconv.ParseFloat(value, 32); err == nil {
		return Float64(v), nil
	} else {
		v, err := ParsePitch(value)
		if err != nil {
			return nil, fmt.Errorf("%s: %s", err, value)
		}
		return v, nil
	}
}

// BeatsPerMin represents beats-per-minute
type BeatsPerMin struct {
	Valuer
	Raw float64
}

// BPM returns a scalar value in beats-per-minute
func BPM(v float64) BeatsPerMin {
	return BeatsPerMin{Raw: v, Valuer: Float64(v / 60 / SampleRate)}
}

func (bpm BeatsPerMin) String() string {
	return fmt.Sprintf("%.2fBPM", bpm.Raw)
}

// Process reads the BPM real value to a Frame
func (bpm BeatsPerMin) Process(out Frame) {
	bpm.Value().Process(out)
}
