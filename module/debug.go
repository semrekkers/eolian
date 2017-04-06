package module

import (
	"fmt"
	"io"
	"os"

	"buddin.us/eolian/dsp"

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
		in:     NewIn("input", dsp.Float64(0)),
		rate:   dsp.SampleRate / rate,
		output: w,
	}
	return m, m.Expose("Debug", []*In{m.in}, []*Out{{Name: "output", Provider: dsp.Provide(m)}})
}

func (d *debug) Process(out dsp.Frame) {
	d.in.Process(out)
	for i := range out {
		if d.tick == 0 {
			fmt.Fprintf(d.output, "%v\n", float64(out[i]))
		}
		d.tick = (d.tick + 1) % d.rate
	}
}
