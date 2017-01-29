package module

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("Debug", func(c Config) (Patcher, error) {
		var config struct {
			RateDivisor int
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.RateDivisor == 0 {
			config.RateDivisor = 10
		}
		return newDebug(config.RateDivisor)
	})
}

type debug struct {
	IO
	in   *In
	rate int
	tick int
}

func newDebug(rate int) (*debug, error) {
	m := &debug{
		in:   &In{Name: "input", Source: zero},
		rate: SampleRate / rate,
	}
	return m, m.Expose("Debug", []*In{m.in}, []*Out{{Name: "output", Provider: Provide(m)}})
}

func (d *debug) Read(out Frame) {
	d.in.Read(out)
	for i := range out {
		if d.tick == 0 {
			fmt.Println(out[i])
		}
		d.tick = (d.tick + 1) % d.rate
	}
}
