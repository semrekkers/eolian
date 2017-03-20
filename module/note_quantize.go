package module

import (
	"fmt"
	"math"
	"strings"

	"buddin.us/musictheory"
	"buddin.us/musictheory/intervals"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("NoteQuantize", func(c Config) (Patcher, error) {
		var config struct{ Key, Scale string }
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Key == "" {
			config.Key = "C"
		}
		if config.Scale == "" {
			config.Scale = "major"
		}
		return newNoteQuantize(config.Key, config.Scale)
	})
}

type noteQuantize struct {
	IO
	in, octave    *In
	key           musictheory.Pitch
	pitches       []Pitch
	intervals     []musictheory.Interval
	intervalsName string
	lastOctave    Value
}

func newNoteQuantize(key, intervalsName string) (*noteQuantize, error) {
	mtKey, err := musictheory.ParsePitch(key + "0")
	if err != nil {
		return nil, err
	}

	intervalsName = strings.ToLower(intervalsName)
	intervals, err := mapIntervals(intervalsName)
	if err != nil {
		return nil, err
	}

	m := &noteQuantize{
		in:            &In{Name: "input", Source: zero},
		octave:        &In{Name: "octave", Source: NewBuffer(Value(3))},
		key:           *mtKey,
		intervalsName: intervalsName,
		intervals:     intervals,
	}
	m.fillPitches(*mtKey)

	return m, m.Expose(
		"NoteQuantize",
		[]*In{m.in, m.octave},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
}

func (q *noteQuantize) fillPitches(tonic musictheory.Pitch) {
	q.pitches = make([]Pitch, len(q.intervals))
	p := tonic
	for i, interval := range q.intervals {
		q.pitches[i] = Pitch{Raw: p.Name(musictheory.AscNames), Valuer: Frequency(p.Freq())}
		p = p.Transpose(interval).(musictheory.Pitch)
	}
}

func (q *noteQuantize) LuaState() map[string]interface{} {
	keyName := q.key.Name(musictheory.AscNames)
	return map[string]interface{}{
		"key":       keyName[:len(keyName)],
		"intervals": q.intervalsName,
	}
}

func (q *noteQuantize) LuaMethods() map[string]LuaMethod {
	return map[string]LuaMethod{
		"setKey": LuaMethod{
			Func: func(v string) error {
				raw := v + "0"
				key, err := musictheory.ParsePitch(raw)
				if err != nil {
					return err
				}
				q.key = *key
				q.fillPitches(*key)
				return nil
			},
			Lock: true,
		},
		"setIntervals": LuaMethod{
			Func: func(v string) error {
				v = strings.ToLower(v)
				intervals, err := mapIntervals(v)
				if err != nil {
					return err
				}
				q.intervalsName = v
				q.intervals = intervals
				q.fillPitches(q.key)
				return nil
			},
			Lock: true,
		},
	}
}

func (q *noteQuantize) Read(out Frame) {
	q.in.Read(out)
	octave := q.octave.ReadFrame()
	for i := range out {
		if q.lastOctave != octave[i] {
			q.fillPitches(q.key.Transpose(musictheory.Octave(int(octave[i]))).(musictheory.Pitch))
		}

		n := float64(len(q.pitches))
		idx := math.Floor(n*float64(out[i]) + 0.5)
		idx = math.Min(idx, n-1)
		idx = math.Max(idx, 0)
		out[i] = q.pitches[int(idx)].Value()

		q.lastOctave = octave[i]
	}
}

func mapIntervals(v string) ([]musictheory.Interval, error) {
	switch v {
	case "major":
		return intervals.Major, nil
	case "minor":
		return intervals.Minor, nil
	case "dorian":
		return intervals.Dorian, nil
	case "lydian":
		return intervals.Lydian, nil
	case "mixolydian":
		return intervals.Mixolydian, nil
	case "minorpentatonic":
		return intervals.MinorPentatonic, nil
	case "majorpentatonic":
		return intervals.MajorPentatonic, nil
	case "aeolian":
		return intervals.Aeolian, nil
	}
	return nil, fmt.Errorf("unknown interaval set %q", v)
}
