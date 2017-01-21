package module

import (
	"fmt"
	"strconv"

	"github.com/brettbuddin/musictheory"
)

const (
	FrameSize  = 256
	SampleRate = 44100.0

	zero = Value(0)
)

type Value float64

func (v Value) Read(out Frame) {
	for i := range out {
		out[i] = v
	}
}

func (valuer Value) Value() Value {
	return valuer
}

type Valuer interface {
	Value() Value
}

type Frame []Value

type Hz struct {
	Valuer
	Raw float64
}

func Frequency(v float64) Hz {
	return Hz{Raw: v, Valuer: Value(v / SampleRate)}
}

func (raw Hz) String() string {
	return fmt.Sprintf("%.2fHz", raw.Raw)
}

func (hz Hz) Read(out Frame) {
	hz.Value().Read(out)
}

func ParsePitch(v string) (Pitch, error) {
	p, err := musictheory.ParsePitch(v)
	if err != nil {
		return Pitch{}, err
	}
	return Pitch{Raw: v, Valuer: Frequency(p.Freq())}, nil
}

type Pitch struct {
	Valuer
	Raw string
}

func (raw Pitch) String() string {
	return raw.Raw
}

func (p Pitch) Read(out Frame) {
	p.Value().Read(out)
}

type MS struct {
	Valuer
	Raw float64
}

func DurationInt(v int) MS {
	return Duration(float64(v))
}

func Duration(v float64) MS {
	return MS{Raw: v, Valuer: Value(v * SampleRate * 0.001)}
}

func (raw MS) String() string {
	return fmt.Sprintf("%dms", int(raw.Raw))
}

func (ms MS) Read(out Frame) {
	ms.Value().Read(out)
}

func ParseValueString(value string) (Valuer, error) {
	if v, err := strconv.ParseInt(value, 10, 64); err == nil {
		return Value(v), nil
	} else if v, err := strconv.ParseFloat(value, 32); err == nil {
		return Value(v), nil
	} else {
		v, err := ParsePitch(value)
		if err != nil {
			return nil, fmt.Errorf("%s: %s", err, value)
		}
		return v, nil
	}
}
