package module

import "math"

func init() {
	f := func(Config) (Patcher, error) { return NewOscillator() }
	Register("Osc", f)
	Register("Oscillator", f)
}

const (
	Pulse WaveShape = iota
	Saw
	Sine
	Triangle
)

type Oscillator struct {
	IO
	pitch, pitchMod, pitchModAmount *In
	detune, amp, offset, sync       *In

	state *oscStateFrames
	reads int

	phases map[string]float64
}

type oscStateFrames struct {
	pitch, pitchMod, pitchModAmount Frame
	detune, amp, offset, sync       Frame
}

func NewOscillator() (*Oscillator, error) {
	m := &Oscillator{
		pitch:          &In{Name: "pitch", Source: NewBuffer(zero)},
		pitchMod:       &In{Name: "pitchMod", Source: NewBuffer(zero)},
		pitchModAmount: &In{Name: "pitchModAmount", Source: NewBuffer(Value(1))},
		amp:            &In{Name: "amp", Source: NewBuffer(Value(1))},
		detune:         &In{Name: "detune", Source: NewBuffer(zero)},
		offset:         &In{Name: "offset", Source: NewBuffer(zero)},
		sync:           &In{Name: "sync", Source: NewBuffer(zero)},
		state:          &oscStateFrames{},
		phases: map[string]float64{
			"sine":     0,
			"saw":      0,
			"pulse":    0,
			"triangle": 0,
			"sub":      0,
		},
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
			{Name: "pulse", Provider: m.out("pulse", Pulse, 1)},
			{Name: "saw", Provider: m.out("saw", Saw, 1)},
			{Name: "sine", Provider: m.out("sine", Sine, 1)},
			{Name: "triangle", Provider: m.out("triangle", Triangle, 1)},
			{Name: "sub", Provider: m.out("sub", Pulse, 0.5)},
		},
	)

	return m, err
}

func (o *Oscillator) out(name string, shape WaveShape, multiplier float64) ReaderProvider {
	return Provide(&oscOut{
		Oscillator: o,
		shape:      shape,
		name:       name,
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
	name       string
	shape      WaveShape
	multiplier float64
	last       Value
}

func (reader *oscOut) Read(out Frame) {
	reader.read(out)
	for i := range out {
		phase := reader.phases[reader.name]
		bPhase := phase / (2 * math.Pi)
		pitch := reader.state.pitch[i] * Value(reader.multiplier)
		delta := float64(pitch + reader.state.detune[i] + reader.state.pitchMod[i]*(reader.state.pitchModAmount[i]/10))
		next := blepSample(reader.shape, phase)*reader.state.amp[i] + reader.state.offset[i]

		switch reader.shape {
		case Sine:
		case Saw:
			next -= blep(bPhase, delta)
		case Pulse:
			next += blep(bPhase, delta)
			next -= blep(math.Mod(bPhase+0.5, 1), delta)
		case Triangle:
			next += blep(bPhase, delta)
			next -= blep(math.Mod(bPhase+0.5, 1), delta)
			next = pitch*next + (1-pitch)*reader.last
		default:
		}

		phase += float64(delta) * 2 * math.Pi
		if phase >= 2*math.Pi {
			phase -= 2 * math.Pi
		}
		if reader.state.sync[i] > 0 {
			phase = 0
		}
		reader.phases[reader.name] = phase
		out[i] = next
		reader.last = next
	}
}

type WaveShape int

func blepSample(shape WaveShape, phase float64) Value {
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
