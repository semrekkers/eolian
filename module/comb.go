package module

import "github.com/mitchellh/mapstructure"

func init() {
	setup := func(f func(i int, c Config) (Patcher, error)) func(Config) (Patcher, error) {
		return func(c Config) (Patcher, error) {
			var config struct {
				Size int
			}
			if err := mapstructure.Decode(c, &config); err != nil {
				return nil, err
			}
			if config.Size == 0 {
				config.Size = 100
			}
			return f(int(SampleRate/1000*float64(config.Size)), c)
		}
	}

	Register("FFComb", setup(func(i int, c Config) (Patcher, error) {
		m, err := NewFFComb(i)
		if err != nil {
			return nil, err
		}
		return m, nil
	}))
	Register("FBComb", setup(func(i int, c Config) (Patcher, error) {
		m, err := NewFBComb(i)
		if err != nil {
			return nil, err
		}
		return m, nil
	}))
	Register("Allpass", setup(func(i int, c Config) (Patcher, error) {
		m, err := NewAllPass(i)
		if err != nil {
			return nil, err
		}
		return m, nil
	}))
}

type FFComb struct {
	IO
	in, duration, gain *In

	Config
	line *DelayLine
}

func NewFFComb(size int) (*FFComb, error) {
	m := &FFComb{
		in:       &In{Name: "input", Source: zero},
		duration: &In{Name: "duration", Source: NewBuffer(Value(1))},
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

func NewFBComb(size int) (*FBComb, error) {
	m := &FBComb{
		in:       &In{Name: "input", Source: zero},
		duration: &In{Name: "duration", Source: NewBuffer(Value(1))},
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

func NewAllPass(size int) (*AllPass, error) {
	m := &AllPass{
		in:       &In{Name: "input", Source: zero},
		duration: &In{Name: "duration", Source: NewBuffer(Value(1))},
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
