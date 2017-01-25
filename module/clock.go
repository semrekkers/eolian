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
		"ClockDivide",
		[]*In{m.in, m.divisor},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (d *Divider) Read(out Frame) {
	d.in.Read(out)
	divisor := d.divisor.ReadFrame()
	for i := range out {
		in := out[i]
		if d.tick == 0 || Value(d.tick) >= divisor[i] {
			out[i] = 1
			d.tick = 0
		} else {
			out[i] = -1
		}
		if d.last < 0 && in > 0 {
			d.tick++
		}
		d.last = in
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
		"ClockMultiply",
		[]*In{m.in, m.multiplier},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (m *Multiplier) Read(out Frame) {
	m.in.Read(out)
	multiplier := m.multiplier.ReadFrame()
	for i := range out {
		in := out[i]
		if m.learn.last < 0 && in > 0 {
			m.rate = m.learn.rate
			m.learn.rate = 0
		}
		m.learn.rate++
		m.learn.last = in

		if Value(m.tick) < Value(m.rate)/multiplier[i] {
			out[i] = -1
		} else {
			out[i] = 1
			m.tick = 0
		}

		m.tick++
	}
}
