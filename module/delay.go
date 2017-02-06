package module

import "github.com/mitchellh/mapstructure"

func init() {
	Register("Delay", func(c Config) (Patcher, error) {
		var config struct{ Size int }
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Size == 0 {
			config.Size = 10000
		}
		return newDelay(DurationInt(config.Size))
	})
}

type delay struct {
	IO
	in, duration *In
	line         *delayline
}

func newDelay(size MS) (*delay, error) {
	m := &delay{
		in:       &In{Name: "input", Source: zero},
		duration: &In{Name: "duration", Source: NewBuffer(Duration(1000))},
		line:     newDelayLine(size),
	}

	err := m.Expose(
		"delay",
		[]*In{m.in, m.duration},
		[]*Out{
			{Name: "output", Provider: Provide(m)},
		},
	)
	return m, err
}

func (c *delay) Read(out Frame) {
	c.in.Read(out)
	duration := c.duration.ReadFrame()
	for i := range out {
		out[i] = c.line.TickDuration(out[i], duration[i])
	}
}

type delayline struct {
	buffer       Frame
	size, offset int
}

func newDelayLine(size MS) *delayline {
	v := int(size.Value())
	return &delayline{
		size:   v,
		buffer: make(Frame, v),
	}
}

func (d *delayline) TickDuration(v, duration Value) Value {
	if d.offset >= int(duration) || d.offset >= d.size {
		d.offset = 0
	}
	v, d.buffer[d.offset] = d.buffer[d.offset], v
	d.offset++
	return v
}

func (d *delayline) Tick(v Value) Value {
	return d.TickDuration(v, 1)
}

func (l *delayline) Read(out Frame) {
	for i := range out {
		out[i] = l.Tick(out[i])
	}
}
