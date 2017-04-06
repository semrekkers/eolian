package module

import "buddin.us/eolian/dsp"

func init() {
	Register("Coupler", func(c Config) (Patcher, error) { return newCoupler() })
}

const (
	couplerOpen int = iota
	couplerClosed
	couplerFadeOut
	couplerFadeIn

	couplerRatio = 0.01
)

type coupler struct {
	IO
	in, duration, toggle *In
	level                dsp.Float64
	state                int

	lastToggle dsp.Float64
}

func newCoupler() (*coupler, error) {
	m := &coupler{
		in:         NewInBuffer("input", dsp.Float64(0)),
		duration:   NewInBuffer("duration", dsp.Duration(300)),
		toggle:     NewInBuffer("toggle", dsp.Float64(0)),
		state:      couplerOpen,
		lastToggle: -1,
	}
	return m, m.Expose(
		"Coupler",
		[]*In{m.in, m.duration, m.toggle},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}})
}

func (h *coupler) Process(out dsp.Frame) {
	toggle := h.toggle.ProcessFrame()

	switch h.state {
	case couplerOpen:
		in := h.in.ProcessFrame()
		for i := range out {
			if h.lastToggle < 0 && toggle[i] > 0 {
				h.state = couplerFadeOut
			}
			out[i] = in[i]
			h.lastToggle = toggle[i]
		}
	case couplerClosed:
		for i := range out {
			if h.lastToggle < 0 && toggle[i] > 0 {
				h.state = couplerFadeIn
			}
			out[i] = 0
			h.lastToggle = toggle[i]
		}
	case couplerFadeOut:
		in := h.in.ProcessFrame()
		duration := h.duration.ProcessFrame()

		for i := range out {
			base, multiplier := shapeCoeffs(couplerRatio, duration[i], 0, expCurve)
			h.level = base + h.level*multiplier
			if h.level < 0 {
				h.level = 0
				h.state = couplerClosed
			}
			out[i] = in[i] * h.level
			h.lastToggle = toggle[i]
		}
	case couplerFadeIn:
		in := h.in.ProcessFrame()
		duration := h.duration.ProcessFrame()

		for i := range out {
			base, multiplier := shapeCoeffs(couplerRatio, duration[i], 1, logCurve)
			h.level = base + h.level*multiplier
			if h.level > 1 {
				h.level = 1
				h.state = couplerOpen
			}
			out[i] = in[i] * h.level
			h.lastToggle = toggle[i]
		}
	}
}
