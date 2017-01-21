package module

import "github.com/mitchellh/mapstructure"

func init() {
	setup := func(f func(s MS, c Config) (Patcher, error)) func(Config) (Patcher, error) {
		return func(c Config) (Patcher, error) {
			var config struct {
				Size int
			}
			if err := mapstructure.Decode(c, &config); err != nil {
				return nil, err
			}
			if config.Size == 0 {
				config.Size = 10000
			}
			return f(DurationInt(config.Size), c)
		}
	}

	Register("FFComb", setup(func(s MS, c Config) (Patcher, error) { return NewFFComb(s) }))
	Register("FBComb", setup(func(s MS, c Config) (Patcher, error) { return NewFBComb(s) }))
	Register("Allpass", setup(func(s MS, c Config) (Patcher, error) { return NewAllPass(s) }))
}

type FFComb struct {
	IO
	in, duration, gain *In

	Config
	line *DelayLine
}

func NewFFComb(size MS) (*FFComb, error) {
	m := &FFComb{
		in:       &In{Name: "input", Source: zero},
		duration: &In{Name: "duration", Source: NewBuffer(Duration(1000))},
		gain:     &In{Name: "gain", Source: NewBuffer(Value(0.9))},
		line:     NewDelayLine(size),
	}
	err := m.Expose(
		[]*In{m.in, m.duration, m.gain},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *FFComb) Read(out Frame) {
	reader.in.Read(out)
	gain := reader.gain.ReadFrame()
	duration := reader.duration.ReadFrame()
	for i := range out {
		out[i] += gain[i] * reader.line.TickDuration(out[i], duration[i])
	}
}

type FBComb struct {
	IO
	in, duration, gain *In

	line *DelayLine
	last Value
}

func NewFBComb(size MS) (*FBComb, error) {
	m := &FBComb{
		in:       &In{Name: "input", Source: zero},
		duration: &In{Name: "duration", Source: NewBuffer(Duration(1000))},
		gain:     &In{Name: "gain", Source: NewBuffer(Value(0.9))},
		line:     NewDelayLine(size),
	}
	err := m.Expose(
		[]*In{m.in, m.duration, m.gain},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *FBComb) Read(out Frame) {
	reader.in.Read(out)
	gain := reader.gain.ReadFrame()
	duration := reader.duration.ReadFrame()
	for i := range out {
		out[i] += reader.last
		reader.last = gain[i] * reader.line.TickDuration(out[i], duration[i])
	}
}

type AllPass struct {
	IO
	in, duration, gain *In

	line *DelayLine
	last Value
}

func NewAllPass(size MS) (*AllPass, error) {
	m := &AllPass{
		in:       &In{Name: "input", Source: zero},
		duration: &In{Name: "duration", Source: NewBuffer(Duration(1000))},
		gain:     &In{Name: "gain", Source: NewBuffer(Value(0.9))},
		line:     NewDelayLine(size),
	}
	err := m.Expose(
		[]*In{m.in, m.duration, m.gain},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *AllPass) Read(out Frame) {
	reader.in.Read(out)
	gain := reader.gain.ReadFrame()
	duration := reader.duration.ReadFrame()
	for i := range out {
		gain := gain[i]
		in := out[i]
		before := in + -gain*reader.last
		reader.last = reader.line.TickDuration(before, duration[i])
		out[i] = reader.last + gain*before
	}
}
