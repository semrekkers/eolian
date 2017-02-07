package module

import (
	"strconv"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("RotatingClockDivide", func(Config) (Patcher, error) { return newRCD() })
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
		return newClockDivider(config.Divisor)
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
		return newClockMultiply(config.Multiplier)
	})
}

type clockDivide struct {
	IO
	in, divisor *In

	tick int
	last Value
}

func newClockDivider(factor int) (*clockDivide, error) {
	m := &clockDivide{
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

func (d *clockDivide) Read(out Frame) {
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

type clockMultiply struct {
	IO
	in, multiplier *In

	learn struct {
		rate int
		last Value
	}

	rate int
	tick int
}

func newClockMultiply(factor int) (*clockMultiply, error) {
	m := &clockMultiply{
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

func (m *clockMultiply) Read(out Frame) {
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

type rcd struct {
	IO
	in, rotate, reset            *In
	reads, rotation, maxRotation int
	ticks                        []int

	lastIn, lastRotate, lastReset Value
}

func newRCD() (*rcd, error) {
	size := 8

	m := &rcd{
		in:          &In{Name: "input", Source: NewBuffer(zero)},
		rotate:      &In{Name: "rotate", Source: NewBuffer(zero)},
		reset:       &In{Name: "reset", Source: NewBuffer(zero)},
		ticks:       make([]int, size),
		maxRotation: size,
	}

	outputs := []*Out{}
	for i := 0; i < size; i++ {
		outputs = append(outputs, &Out{
			Name:     strconv.Itoa(i + 1),
			Provider: Provide(&rcdOut{rcd: m, pos: i})})
	}

	return m, m.Expose(
		"RCD",
		[]*In{m.in, m.rotate, m.reset},
		outputs,
	)
}

func (d *rcd) read(out Frame) {
	if d.reads > 0 {
		return
	}
	d.in.ReadFrame()
	d.rotate.ReadFrame()
	d.reset.ReadFrame()
}

func (d *rcd) tick(times int, do func(int)) {
	rotate := d.rotate.LastFrame()
	reset := d.reset.LastFrame()
	for i := 0; i < times; i++ {
		if d.reads == 0 {
			if d.lastRotate < 0 && rotate[i] > 0 {
				d.rotation = (d.rotation + 1) % d.maxRotation
			}
			if d.lastReset < 0 && reset[i] > 0 {
				d.rotation = 0
			}
			d.lastReset = reset[i]
			d.lastRotate = rotate[i]
		}
		do(i)
	}
}

func (d *rcd) postRead() {
	if outs := d.OutputsActive(); outs > 0 {
		d.reads = (d.reads + 1) % outs
	}
}

type rcdOut struct {
	*rcd
	pos  int
	last Value
}

func (d *rcdOut) Read(out Frame) {
	d.read(out)
	in := d.in.LastFrame()

	count := d.pos - d.rotation
	if count < 0 {
		count = d.maxRotation + count
	}

	d.tick(len(out), func(i int) {
		if d.last < 0 && in[i] > 0 {
			d.ticks[d.pos] = (d.ticks[d.pos] + 1) % (count + 1)
		}
		d.last = in[i]
		if d.ticks[d.pos] == 0 {
			out[i] = 1
		} else {
			out[i] = -1
		}
	})
	d.postRead()
}
