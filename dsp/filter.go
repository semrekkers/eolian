package dsp

// SVFilter is a state-variable filter that yields lowpass, bandpass and highpass outputs
type SVFilter struct {
	Poles             int
	Cutoff, Resonance Float64

	lastCutoff, g, state1, state2 Float64
}

// Tick advances the operation
func (f *SVFilter) Tick(in Float64) (lp, bp, hp Float64) {
	cutoff := Abs(f.Cutoff)
	if cutoff != f.lastCutoff {
		f.g = Tan(cutoff)
	}
	f.lastCutoff = cutoff

	r := 1 / Max(f.Resonance, 1)
	h := 1 / (1 + r*f.g + f.g*f.g)

	for j := 0; j < f.Poles; j++ {
		hp = h * (in - r*f.state1 - f.g*f.state1 - f.state2)
		bp = f.g*hp + f.state1
		lp = f.g*bp + f.state2

		f.state1 = f.g*hp + bp
		f.state2 = f.g*bp + lp
	}
	return
}
