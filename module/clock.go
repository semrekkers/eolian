package module

import "github.com/mitchellh/mapstructure"

func init() {
	Register("ClockDivide", func(c Config) (Patcher, error) {
		var config struct {
			Divisor int
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Divisor == 0 {
			config.Divisor = 1
		}
		return NewDivider(config.Divisor)
	})
	Register("ClockMultiply", func(c Config) (Patcher, error) {
		var config struct {
			Multiplier int
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Multiplier == 0 {
			config.Multiplier = 1
		}
		return NewMultiplier(config.Multiplier)
	})
}

type Divider struct {
	IO
	in, divisor *In

	tick int
	last Value
}

func NewDivider(factor int) (*Divider, error) {
	m := &Divider{
		in:      &In{Name: "input", Source: zero},
		divisor: &In{Name: "divisor", Source: NewBuffer(Value(factor))},
	}
	err := m.Expose(
		[]*In{m.in, m.divisor},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *Divider) Read(out Frame) {
	reader.in.Read(out)
	divisor := reader.divisor.ReadFrame()
	for i := range out {
		in := out[i]
		if reader.tick == 0 || Value(reader.tick) >= divisor[i] {
			out[i] = 1
			reader.tick = 0
		} else {
			out[i] = -1
		}
		if reader.last < 0 && in > 0 {
			reader.tick++
		}
		reader.last = in
	}
}

type Multiplier struct {
	IO
	in, multiplier *In

	learn struct {
		rate int
		last Value
	}

	rate int
	tick int
}

func NewMultiplier(factor int) (*Multiplier, error) {
	m := &Multiplier{
		in:         &In{Name: "input", Source: zero},
		multiplier: &In{Name: "multiplier", Source: NewBuffer(Value(factor))},
	}
	err := m.Expose(
		[]*In{m.in, m.multiplier},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (reader *Multiplier) Read(out Frame) {
	reader.in.Read(out)
	multiplier := reader.multiplier.ReadFrame()
	for i := range out {
		in := out[i]
		if reader.learn.last < 0 && in > 0 {
			reader.rate = reader.learn.rate
			reader.learn.rate = 0
		}
		reader.learn.rate++
		reader.learn.last = in

		if Value(reader.tick) < Value(reader.rate)/multiplier[i] {
			out[i] = -1
		} else {
			out[i] = 1
			reader.tick = 0
		}

		reader.tick++
	}
}
