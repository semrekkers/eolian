package module

import "github.com/mitchellh/mapstructure"

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
	IO
	in, cutoff, resonance *In
	filter                *filter
	reads                 int
	lp, bp, hp            Frame
}

func newSVFilter(poles int) (*svFilter, error) {
	m := &svFilter{
		in:        &In{Name: "input", Source: zero},
		cutoff:    &In{Name: "cutoff", Source: NewBuffer(Frequency(1000))},
		resonance: &In{Name: "resonance", Source: NewBuffer(Value(1))},
		filter:    &filter{poles: poles},
		lp:        make(Frame, FrameSize),
		bp:        make(Frame, FrameSize),
		hp:        make(Frame, FrameSize),
	}
	return m, m.Expose(
		"Filter",
		[]*In{m.in, m.cutoff, m.resonance},
		[]*Out{
			{Name: "lowpass", Provider: m.out(lowpass)},
			{Name: "highpass", Provider: m.out(highpass)},
			{Name: "bandpass", Provider: m.out(bandpass)},
		},
	)
}

func (f *svFilter) out(m filterMode) ReaderProvider {
	return Provide(&svFilterOut{svFilter: f, mode: m})
}

func (f *svFilter) read(out Frame) {
	if f.reads == 0 {
		f.in.Read(out)
		cutoff := f.cutoff.ReadFrame()
		resonance := f.resonance.ReadFrame()

		for i := range out {
			f.filter.cutoff = cutoff[i]
			f.filter.resonance = resonance[i]
			f.lp[i], f.bp[i], f.hp[i] = f.filter.Tick(out[i])
		}
	}
	if count := f.OutputsActive(); count > 0 {
		f.reads = (f.reads + 1) % count
	}
}

type svFilterOut struct {
	*svFilter
	mode filterMode
}

func (f *svFilterOut) Read(out Frame) {
	f.read(out)
	for i := range out {
		switch f.mode {
		case lowpass:
			out[i] = f.lp[i]
		case highpass:
			out[i] = f.hp[i]
		case bandpass:
			out[i] = f.bp[i]
		}
	}
}

type filterMode int

const (
	lowpass filterMode = iota
	highpass
	bandpass
)

type filter struct {
	poles              int
	cutoff, lastCutoff Value
	resonance          Value
	g, state1, state2  Value
}

func (f *filter) Tick(in Value) (lp, bp, hp Value) {
	if f.cutoff != f.lastCutoff {
		f.g = tanValue(f.cutoff)
	}
	f.lastCutoff = f.cutoff

	r := Value(1 / f.resonance)
	h := 1 / (1 + r*f.g + f.g*f.g)

	for j := 0; j < f.poles; j++ {
		hp = h * (in - r*f.state1 - f.g*f.state1 - f.state2)
		bp = f.g*hp + f.state1
		lp = f.g*bp + f.state2

		f.state1 = f.g*hp + bp
		f.state2 = f.g*bp + lp
	}
	return
}
