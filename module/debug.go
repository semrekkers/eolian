package module

import (
	"fmt"
	"io"
	"os"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("Debug", func(c Config) (Patcher, error) {
		var config struct {
			RateDivisor int
			Output      io.Writer
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.RateDivisor == 0 {
			config.RateDivisor = 10
		}
		if config.Output == nil {
			config.Output = os.Stdout
		}
		return newDebug(config.Output, config.RateDivisor)
	})
}

type debug struct {
	IO
	in     *In
	rate   int
	tick   int
	output io.Writer
}

func newDebug(w io.Writer, rate int) (*debug, error) {
	m := &debug{
		in:     &In{Name: "input", Source: zero},
		rate:   SampleRate / rate,
		output: w,
	}
	return m, m.Expose("Debug", []*In{m.in}, []*Out{{Name: "output", Provider: Provide(m)}})
}

func (d *debug) Read(out Frame) {
	d.in.Read(out)
	for i := range out {
		if d.tick == 0 {
			fmt.Fprintf(d.output, "%v\n", out[i])
		}
		d.tick = (d.tick + 1) % d.rate
	}
}
