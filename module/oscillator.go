package module

import (
	"math"

	"buddin.us/eolian/dsp"

	"github.com/mitchellh/mapstructure"
)

func init() {
	f := func(c Config) (Patcher, error) {
		var config struct {
			Algorithm  string
			Multiplier float64
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		switch config.Algorithm {
		case algBLEP:
		case algSimple:
		default:
			config.Algorithm = algBLEP
		}
		if config.Multiplier == 0 {
			config.Multiplier = 1
		}
		return newOscillator(config.Algorithm, config.Multiplier)
	}
	Register("Osc", f)
	Register("Oscillator", f)
}

const (
	pulse = iota
	saw
	sine
	triangle

	algSimple = "simple"
	algBLEP   = "blep"
)

type oscillator struct {
	multiOutIO
	pitch, pitchMod, pitchModAmount       *In
	detune, amp, offset, sync, pulseWidth *In

	algorithm string
	state     *oscStateFrames
	phases    []float64
}

type oscStateFrames struct {
	pitch, pitchMod, pitchModAmount       dsp.Frame
	detune, amp, offset, sync, pulseWidth dsp.Frame
}

func newOscillator(algorithm string, multiplier float64) (*oscillator, error) {
	m := &oscillator{
		pitch:          NewInBuffer("pitch", dsp.Float64(0)),
		pitchMod:       NewInBuffer("pitchMod", dsp.Float64(0)),
		pitchModAmount: NewInBuffer("pitchModAmount", dsp.Float64(1)),
		amp:            NewInBuffer("amp", dsp.Float64(1)),
		detune:         NewInBuffer("detune", dsp.Float64(0)),
		offset:         NewInBuffer("offset", dsp.Float64(0)),
		pulseWidth:     NewInBuffer("pulseWidth", dsp.Float64(1)),
		sync:           NewInBuffer("sync", dsp.Float64(0)),
		state:          &oscStateFrames{},
		phases:         make([]float64, 5),
		algorithm:      algorithm,
	}

	err := m.Expose(
		"Oscillator",
		[]*In{
			m.pitch,
			m.pitchMod,
			m.pitchModAmount,
			m.amp,
			m.detune,
			m.offset,
			m.sync,
			m.pulseWidth,
		},
		[]*Out{
			{Name: "pulse", Provider: m.out(0, pulse, multiplier)},
			{Name: "saw", Provider: m.out(1, saw, multiplier)},
			{Name: "sine", Provider: m.out(2, sine, multiplier)},
			{Name: "triangle", Provider: m.out(3, triangle, multiplier)},
			{Name: "sub", Provider: m.out(4, pulse, 0.5*multiplier)},
		},
	)

	return m, err
}

func (o *oscillator) out(idx int, shape int, multiplier float64) dsp.ProcessorProvider {
	return dsp.ProcessorProviderFunc(func() dsp.Processor {
		return &oscOut{
			oscillator: o,
			phaseIndex: idx,
			shape:      shape,
			multiplier: multiplier,
		}
	})
}

func (o *oscillator) read(out dsp.Frame) {
	o.incrRead(func() {
		o.state.pitch = o.pitch.ProcessFrame()
		o.state.pitchMod = o.pitchMod.ProcessFrame()
		o.state.pitchModAmount = o.pitchModAmount.ProcessFrame()
		o.state.amp = o.amp.ProcessFrame()
		o.state.offset = o.offset.ProcessFrame()
		o.state.detune = o.detune.ProcessFrame()
		o.state.sync = o.sync.ProcessFrame()
		o.state.pulseWidth = o.pulseWidth.ProcessFrame()
	})
}

type oscOut struct {
	*oscillator
	phaseIndex int
	shape      int
	multiplier float64
	last       dsp.Float64
}

func (o *oscOut) Process(out dsp.Frame) {
	o.read(out)
	for i := range out {
		switch o.algorithm {
		case algBLEP:
			o.blep(out, i)
		case algSimple:
			o.simple(out, i)
		}
	}
}

func (o *oscOut) blep(out dsp.Frame, i int) {
	var (
		phase  = o.phases[o.phaseIndex]
		bPhase = phase / (2 * math.Pi)
		pitch  = o.state.pitch[i] * dsp.Float64(o.multiplier)
		delta  = float64(pitch +
			o.state.detune[i] +
			o.state.pitchMod[i]*(o.state.pitchModAmount[i]/10))
		pulseWidth = float64(dsp.Clamp(o.state.pulseWidth[i], 0.1, 1))
		next       = blepSample(o.shape, phase, pulseWidth)*o.state.amp[i] + o.state.offset[i]
	)

	switch o.shape {
	case sine:
	case saw:
		next -= blep(bPhase, delta)
	case pulse:
		next += blep(bPhase, delta)
		next -= blep(math.Mod(bPhase+0.5, 1), delta)
	case triangle:
		next += blep(bPhase, delta)
		next -= blep(math.Mod(bPhase+0.5, 1), delta)
		next = pitch*next + (1-pitch)*o.last
	default:
	}

	phase += delta * 2 * math.Pi
	if phase >= 2*math.Pi {
		phase -= 2 * math.Pi
	}
	if o.state.sync[i] > 0 {
		phase = 0
	}
	o.phases[o.phaseIndex] = phase
	out[i] = next
	o.last = next
}

func blepSample(shape int, phase, pulseWidth float64) dsp.Float64 {
	switch shape {
	case sine:
		return dsp.Float64(math.Sin(phase))
	case saw:
		return dsp.Float64(2.0*phase/(2*math.Pi) - 1.0)
	case triangle:
		if phase < math.Pi {
			return 1
		}
		return -1
	case pulse:
		if phase < math.Pi*pulseWidth {
			return 1
		}
		return -1
	default:
		return 0
	}
}

func blep(p float64, delta float64) dsp.Float64 {
	if p < delta {
		p /= delta
		return dsp.Float64(p + p - p*p - 1.0)
	} else if p > 1.0-delta {
		p = (p - 1.0) / delta
		return dsp.Float64(p + p + p*p + 1.0)
	}
	return 0.0
}

func (o *oscOut) simple(out dsp.Frame, i int) {
	var (
		phase = o.phases[o.phaseIndex]
		amp   = o.state.amp[i]
		pitch = (o.state.pitch[i] * dsp.Float64(o.multiplier)) +
			o.state.detune[i] +
			o.state.pitchMod[i]*(o.state.pitchModAmount[i]/10)
		offset     = o.state.offset[i]
		pulseWidth = float64(dsp.Clamp(o.state.pulseWidth[i], 0.1, 0.9))
		next       dsp.Float64
	)

	switch o.shape {
	case sine:
		next = dsp.Float64(math.Sin(phase)) * amp
	case saw:
		next = dsp.Float64(1-float32(1/math.Pi*phase)) * amp
	case pulse:
		if phase < math.Pi*pulseWidth {
			next = 1 * amp
		} else {
			next = -1 * amp
		}
	case triangle:
		if phase < math.Pi {
			next = dsp.Float64(-1+(2/math.Pi)*phase) * amp
		} else {
			next = dsp.Float64(3+(2/math.Pi)*phase) * amp
		}
	default:
	}

	phase += float64(pitch) * 2 * math.Pi
	if phase >= 2*math.Pi {
		phase -= 2 * math.Pi
	}
	if o.state.sync[i] > 0 {
		phase = 0
	}
	o.phases[o.phaseIndex] = phase
	next += offset
	out[i] = next
	o.last = next
}
