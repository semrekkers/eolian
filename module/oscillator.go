package module

import (
	"math"

	"github.com/mitchellh/mapstructure"
)

func init() {
	f := func(c Config) (Patcher, error) {
		var config struct {
			Algorithm string
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Algorithm == "" {
			config.Algorithm = algBLEP
		}
		return newOscillator(config.Algorithm)
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
	IO
	pitch, pitchMod, pitchModAmount       *In
	detune, amp, offset, sync, pulseWidth *In

	algorithm string
	state     *oscStateFrames
	reads     int
	phases    []float64
}

type oscStateFrames struct {
	pitch, pitchMod, pitchModAmount       Frame
	detune, amp, offset, sync, pulseWidth Frame
}

func newOscillator(algorithm string) (*oscillator, error) {
	m := &oscillator{
		pitch:          &In{Name: "pitch", Source: NewBuffer(zero)},
		pitchMod:       &In{Name: "pitchMod", Source: NewBuffer(zero)},
		pitchModAmount: &In{Name: "pitchModAmount", Source: NewBuffer(Value(1))},
		amp:            &In{Name: "amp", Source: NewBuffer(Value(1))},
		detune:         &In{Name: "detune", Source: NewBuffer(zero)},
		offset:         &In{Name: "offset", Source: NewBuffer(zero)},
		pulseWidth:     &In{Name: "pulseWidth", Source: NewBuffer(Value(0.5))},
		sync:           &In{Name: "sync", Source: NewBuffer(zero)},
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
			{Name: "pulse", Provider: m.out(0, pulse, 1)},
			{Name: "saw", Provider: m.out(1, saw, 1)},
			{Name: "sine", Provider: m.out(2, sine, 1)},
			{Name: "triangle", Provider: m.out(3, triangle, 1)},
			{Name: "sub", Provider: m.out(4, pulse, 0.5)},
		},
	)

	return m, err
}

func (o *oscillator) out(idx int, shape int, multiplier float64) ReaderProvider {
	return Provide(&oscOut{
		oscillator: o,
		phaseIndex: idx,
		shape:      shape,
		multiplier: multiplier,
	})
}

func (o *oscillator) read(out Frame) {
	if o.reads == 0 {
		o.state.pitch = o.pitch.ReadFrame()
		o.state.pitchMod = o.pitchMod.ReadFrame()
		o.state.pitchModAmount = o.pitchModAmount.ReadFrame()
		o.state.amp = o.amp.ReadFrame()
		o.state.offset = o.offset.ReadFrame()
		o.state.detune = o.detune.ReadFrame()
		o.state.sync = o.sync.ReadFrame()
		o.state.pulseWidth = o.pulseWidth.ReadFrame()
	}
	if count := o.OutputsActive(); count > 0 {
		o.reads = (o.reads + 1) % count
	}
}

type oscOut struct {
	*oscillator
	phaseIndex int
	shape      int
	multiplier float64
	last       Value
}

func (o *oscOut) Read(out Frame) {
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

func (o *oscOut) blep(out Frame, i int) {
	var (
		phase  = o.phases[o.phaseIndex]
		bPhase = phase / (2 * math.Pi)
		pitch  = o.state.pitch[i] * Value(o.multiplier)
		delta  = float64(pitch +
			o.state.detune[i] +
			o.state.pitchMod[i]*(o.state.pitchModAmount[i]/10))
		pulseWidth = float64(clampValue(o.state.pulseWidth[i], 0.1, 0.9))
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

	phase += float64(delta) * 2 * math.Pi
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

func blepSample(shape int, phase, pulseWidth float64) Value {
	switch shape {
	case sine:
		return Value(math.Sin(phase))
	case saw:
		return Value(2.0*phase/(2*math.Pi) - 1.0)
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

func blep(p float64, delta float64) Value {
	if p < delta {
		p /= delta
		return Value(p + p - p*p - 1.0)
	} else if p > 1.0-delta {
		p = (p - 1.0) / delta
		return Value(p + p + p*p + 1.0)
	}
	return 0.0
}

func (o *oscOut) simple(out Frame, i int) {
	var (
		phase = o.phases[o.phaseIndex]
		amp   = o.state.amp[i]
		pitch = o.state.pitch[i] +
			o.state.detune[i] +
			o.state.pitchMod[i]*(o.state.pitchModAmount[i]/10)
		offset     = o.state.offset[i]
		pulseWidth = float64(clampValue(o.state.pulseWidth[i], 0.1, 0.9))
		next       Value
	)

	switch o.shape {
	case sine:
		next = Value(math.Sin(float64(phase))) * amp
	case saw:
		next = Value(1-float32(1/math.Pi*phase)) * amp
	case pulse:
		if phase < math.Pi*pulseWidth {
			next = 1 * amp
		} else {
			next = -1 * amp
		}
	case triangle:
		if phase < math.Pi {
			next = Value(-1+(2/math.Pi)*phase) * amp
		} else {
			next = Value(3+(2/math.Pi)*phase) * amp
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
