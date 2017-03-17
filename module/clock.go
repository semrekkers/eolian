package module

import (
	"math"
	"strconv"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("Clock", func(Config) (Patcher, error) { return newClock() })
	Register("RotatingClockDivide", func(Config) (Patcher, error) { return newRCD(8) })
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

type clock struct {
	IO
	tempo, width, shuffle *In
	tick                  int
	second                bool
}

func newClock() (*clock, error) {
	m := &clock{
		tempo:   &In{Name: "tempo", Source: NewBuffer(BPM(120))},
		width:   &In{Name: "pulseWidth", Source: NewBuffer(Value(0.9))},
		shuffle: &In{Name: "shuffle", Source: NewBuffer(Value(0))},
	}
	return m, m.Expose(
		"Clock",
		[]*In{m.tempo, m.width, m.shuffle},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
}

func (t *clock) Read(out Frame) {
	tempo := t.tempo.ReadFrame()
	width := t.width.ReadFrame()
	shuffle := t.shuffle.ReadFrame()

	for i := range out {
		duty := math.Floor(float64(60/(tempo[i]*60*SampleRate)*SampleRate) + 0.5)

		if !t.second {
			if t.tick >= int(duty) {
				t.tick = 0
				t.second = true
			}
		} else if t.tick >= int(duty+(duty*float64(shuffle[i]))) {
			t.tick = 0
			t.second = false
		}

		if Value(t.tick) <= width[i]*Value(duty) {
			out[i] = 1
		} else {
			out[i] = -1
		}
		t.tick++
	}
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
	multiOutIO
	in, rotate, reset     *In
	rotation, maxRotation int
	ticks                 []int

	lastIn, lastRotate, lastReset Value
	outs                          []Frame
}

func newRCD(size int) (*rcd, error) {
	m := &rcd{
		in:          &In{Name: "input", Source: NewBuffer(zero)},
		rotate:      &In{Name: "rotate", Source: NewBuffer(zero)},
		reset:       &In{Name: "reset", Source: NewBuffer(zero)},
		ticks:       make([]int, size),
		outs:        make([]Frame, size),
		maxRotation: size,
		lastIn:      -1,
		lastReset:   -1,
		lastRotate:  -1,
	}

	outputs := []*Out{}
	for i := 0; i < size; i++ {
		m.outs[i] = make(Frame, FrameSize)
		outputs = append(outputs, &Out{
			Name:     strconv.Itoa(i + 1),
			Provider: provideCopyOut(m, &m.outs[i])})
	}

	return m, m.Expose(
		"RCD",
		[]*In{m.in, m.rotate, m.reset},
		outputs,
	)
}

func (d *rcd) Read(out Frame) {
	d.incrRead(func() {
		var (
			in     = d.in.ReadFrame()
			rotate = d.rotate.ReadFrame()
			reset  = d.reset.ReadFrame()
		)

		for i := range out {
			if d.lastRotate < 0 && rotate[i] > 0 {
				d.rotation = (d.rotation + 1) % d.maxRotation
			}
			if d.lastReset < 0 && reset[i] > 0 {
				d.rotation = 0
			}

			for j := 0; j < len(d.outs); j++ {
				count := j - d.rotation
				if count < 0 {
					count = d.maxRotation + count
				}
				if d.lastIn < 0 && in[i] > 0 {
					d.ticks[j] = (d.ticks[j] + 1) % (count + 1)
				}
				if d.ticks[j] == 0 {
					d.outs[j][i] = 1
				} else {
					d.outs[j][i] = -1
				}
			}

			d.lastIn = in[i]
			d.lastReset = reset[i]
			d.lastRotate = rotate[i]
		}
	})
}
