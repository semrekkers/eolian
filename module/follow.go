package module

import "math"

func init() {
	Register("Follow", func(Config) (Patcher, error) { return newFollow() })
}

type follow struct {
	IO
	in, attack, release *In

	envelope Value
}

func newFollow() (*follow, error) {
	m := &follow{
		in:      &In{Name: "input", Source: zero},
		attack:  &In{Name: "attack", Source: NewBuffer(Duration(10))},
		release: &In{Name: "release", Source: NewBuffer(Duration(500))},
	}
	err := m.Expose(
		"Follow",
		[]*In{m.in, m.attack, m.release},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

func (c *follow) Read(out Frame) {
	c.in.Read(out)
	attack, release := c.attack.ReadFrame(), c.release.ReadFrame()
	for i := range out {
		in := absValue(out[i])
		side := release[i]
		if in > c.envelope {
			side = attack[i]
		}
		factor := math.Pow(0.01, float64(1.0/side))
		c.envelope = Value(factor)*(c.envelope-in) + in
		out[i] = c.envelope
	}
}
