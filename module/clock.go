package module

import (
	"math"
	"strconv"

	"buddin.us/eolian/dsp"

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
		tempo:   NewInBuffer("tempo", dsp.BPM(120)),
		width:   NewInBuffer("pulseWidth", dsp.Float64(0.9)),
		shuffle: NewInBuffer("shuffle", dsp.Float64(0)),
	}
	return m, m.Expose(
		"Clock",
		[]*In{m.tempo, m.width, m.shuffle},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
}

func (t *clock) Process(out dsp.Frame) {
	tempo := t.tempo.ProcessFrame()
	width := t.width.ProcessFrame()
	shuffle := t.shuffle.ProcessFrame()

	for i := range out {
		duty := math.Floor(float64(60/(tempo[i]*60*dsp.SampleRate)*dsp.SampleRate) + 0.5)

		if !t.second {
			if t.tick >= int(duty) {
				t.tick = 0
				t.second = true
			}
		} else if t.tick >= int(duty+(duty*float64(dsp.Clamp(shuffle[i], -0.5, 0.5)))) {
			t.tick = 0
			t.second = false
		}

		if dsp.Float64(t.tick) <= width[i]*dsp.Float64(duty) {
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
	last dsp.Float64
}

func newClockDivider(factor int) (*clockDivide, error) {
	m := &clockDivide{
		in:      NewIn("input", dsp.Float64(0)),
		divisor: NewInBuffer("divisor", dsp.Float64(factor)),
	}
	err := m.Expose(
		"ClockDivide",
		[]*In{m.in, m.divisor},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

func (d *clockDivide) Process(out dsp.Frame) {
	d.in.Process(out)
	divisor := d.divisor.ProcessFrame()
	for i := range out {
		in := out[i]
		if d.tick == 0 || dsp.Float64(d.tick) >= divisor[i] {
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
		last dsp.Float64
	}

	rate int
	tick int
}

func newClockMultiply(factor int) (*clockMultiply, error) {
	m := &clockMultiply{
		in:         NewIn("input", dsp.Float64(0)),
		multiplier: NewInBuffer("multiplier", dsp.Float64(factor)),
	}
	err := m.Expose(
		"ClockMultiply",
		[]*In{m.in, m.multiplier},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

func (m *clockMultiply) Process(out dsp.Frame) {
	m.in.Process(out)
	multiplier := m.multiplier.ProcessFrame()
	for i := range out {
		in := out[i]
		if m.learn.last < 0 && in > 0 {
			m.rate = m.learn.rate
			m.learn.rate = 0
		}
		m.learn.rate++
		m.learn.last = in

		if dsp.Float64(m.tick) < dsp.Float64(m.rate)/multiplier[i] {
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

	lastIn, lastRotate, lastReset dsp.Float64
	outs                          []dsp.Frame
}

func newRCD(size int) (*rcd, error) {
	m := &rcd{
		in:          NewInBuffer("input", dsp.Float64(0)),
		rotate:      NewInBuffer("rotate", dsp.Float64(0)),
		reset:       NewInBuffer("reset", dsp.Float64(0)),
		ticks:       make([]int, size),
		outs:        make([]dsp.Frame, size),
		maxRotation: size,
		lastIn:      -1,
		lastReset:   -1,
		lastRotate:  -1,
	}

	outputs := []*Out{}
	for i := 0; i < size; i++ {
		m.outs[i] = dsp.NewFrame()
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

func (d *rcd) Process(out dsp.Frame) {
	d.incrRead(func() {
		var (
			in     = d.in.ProcessFrame()
			rotate = d.rotate.ProcessFrame()
			reset  = d.reset.ProcessFrame()
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
				if d.ticks[j] == 0 && count == 0 {
					d.outs[j][i] = in[i]
				} else if d.ticks[j] == 0 {
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
