package module

import (
	"buddin.us/eolian/dsp"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("Filter", func(c Config) (Patcher, error) {
		var config struct {
			Poles int
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}

		if config.Poles == 0 {
			config.Poles = 4
		}

		return newSVFilter(config.Poles)
	})
}

type svFilter struct {
	multiOutIO
	in, cutoff, resonance *In
	filter                *dsp.SVFilter
	lp, bp, hp            dsp.Frame
}

func newSVFilter(poles int) (*svFilter, error) {
	m := &svFilter{
		in:        NewIn("input", dsp.Float64(0)),
		cutoff:    NewInBuffer("cutoff", dsp.Frequency(1000)),
		resonance: NewInBuffer("resonance", dsp.Float64(1)),
		filter:    &dsp.SVFilter{Poles: poles},
		lp:        dsp.NewFrame(),
		bp:        dsp.NewFrame(),
		hp:        dsp.NewFrame(),
	}

	return m, m.Expose(
		"Filter",
		[]*In{m.in, m.cutoff, m.resonance},
		[]*Out{
			{Name: "lowpass", Provider: provideCopyOut(m, &m.lp)},
			{Name: "highpass", Provider: provideCopyOut(m, &m.hp)},
			{Name: "bandpass", Provider: provideCopyOut(m, &m.bp)},
		},
	)
}

func (f *svFilter) Process(out dsp.Frame) {
	f.incrRead(func() {
		f.in.Process(out)
		cutoff := f.cutoff.ProcessFrame()
		resonance := f.resonance.ProcessFrame()

		for i := range out {
			f.filter.Cutoff = cutoff[i]
			f.filter.Resonance = resonance[i]
			f.lp[i], f.bp[i], f.hp[i] = f.filter.Tick(out[i])
		}
	})
}
