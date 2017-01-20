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
			config.Algorithm = BLEP
		}
		return NewOscillator(config.Algorithm)
	}
	Register("Osc", f)
	Register("Oscillator", f)
}

const (
	Pulse = iota
	Saw
	Sine
	Triangle

	Simple = "simple"
	BLEP   = "blep"
)

type Oscillator struct {
	IO
	pitch, pitchMod, pitchModAmount *In
	detune, amp, offset, sync       *In

	algorithm string
	state     *oscStateFrames
	reads     int
	phases    []float64
}

type oscStateFrames struct {
	pitch, pitchMod, pitchModAmount Frame
	detune, amp, offset, sync       Frame
}

func NewOscillator(algorithm string) (*Oscillator, error) {
	m := &Oscillator{
		pitch:          &In{Name: "pitch", Source: NewBuffer(zero)},
		pitchMod:       &In{Name: "pitchMod", Source: NewBuffer(zero)},
		pitchModAmount: &In{Name: "pitchModAmount", Source: NewBuffer(Value(1))},
		amp:            &In{Name: "amp", Source: NewBuffer(Value(1))},
		detune:         &In{Name: "detune", Source: NewBuffer(zero)},
		offset:         &In{Name: "offset", Source: NewBuffer(zero)},
		sync:           &In{Name: "sync", Source: NewBuffer(zero)},
		state:          &oscStateFrames{},
		phases:         make([]float64, 5),
		algorithm:      algorithm,
	}

	err := m.Expose(
		[]*In{
			m.pitch,
			m.pitchMod,
			m.pitchModAmount,
			m.amp,
			m.detune,
			m.offset,
			m.sync,
		},
		[]*Out{
			{Name: "pulse", Provider: m.out(0, Pulse, 1)},
			{Name: "saw", Provider: m.out(1, Saw, 1)},
			{Name: "sine", Provider: m.out(2, Sine, 1)},
			{Name: "triangle", Provider: m.out(3, Triangle, 1)},
			{Name: "sub", Provider: m.out(4, Pulse, 0.5)},
		},
	)

	return m, err
}

func (o *Oscillator) out(idx int, shape int, multiplier float64) ReaderProvider {
	return Provide(&oscOut{
		Oscillator: o,
		phaseIndex: idx,
		shape:      shape,
		multiplier: multiplier,
	})
}

func (o *Oscillator) read(out Frame) {
	if o.reads == 0 {
		o.state.pitch = o.pitch.ReadFrame()
		o.state.pitchMod = o.pitchMod.ReadFrame()
		o.state.pitchModAmount = o.pitchModAmount.ReadFrame()
		o.state.amp = o.amp.ReadFrame()
		o.state.offset = o.offset.ReadFrame()
		o.state.detune = o.detune.ReadFrame()
		o.state.sync = o.sync.ReadFrame()
	}
	if count := o.OutputsActive(); count > 0 {
		o.reads = (o.reads + 1) % count
	}
}

type oscOut struct {
	*Oscillator
	phaseIndex int
	shape      int
	multiplier float64
	last       Value
}

func (reader *oscOut) Read(out Frame) {
	reader.read(out)
	for i := range out {
		switch reader.algorithm {
		case BLEP:
			reader.blep(out, i)
		case Simple:
			reader.simple(out, i)
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
		next = blepSample(o.shape, phase)*o.state.amp[i] + o.state.offset[i]
	)

	switch o.shape {
	case Sine:
	case Saw:
		next -= blep(bPhase, delta)
	case Pulse:
		next += blep(bPhase, delta)
		next -= blep(math.Mod(bPhase+0.5, 1), delta)
	case Triangle:
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

func blepSample(shape int, phase float64) Value {
	switch shape {
	case Sine:
		return Value(math.Sin(phase))
	case Saw:
		return Value(2.0*phase/(2*math.Pi) - 1.0)
	case Triangle:
		fallthrough
	case Pulse:
		if phase < math.Pi {
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
		offset = o.state.offset[i]
		next   Value
	)

	switch o.shape {
	case Sine:
		next = Value(math.Sin(float64(phase))) * amp
	case Saw:
		next = Value(1-float32(1/math.Pi*phase)) * amp
	case Pulse:
		if phase < math.Pi {
			next = 1 * amp
		} else {
			next = -1 * amp
		}
	case Triangle:
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
