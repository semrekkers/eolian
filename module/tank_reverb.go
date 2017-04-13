package module

import (
	"buddin.us/eolian/dsp"
)

func init() {
	Register("TankReverb", func(c Config) (Patcher, error) {
		return newAllpassReverb([]int{113, 162, 241, 399})
	})
}

type allpassReverb struct {
	multiOutIO

	a, b, defuse, bias, cutoff, decay *In
	aFilter                           *dsp.SVFilter
	bFilter                           *dsp.SVFilter
	ap, aAP, bAP                      []*dsp.AllPass
	aDL, bDL                          *dsp.DelayLine
	aLast, bLast                      dsp.Float64
	aOut, bOut                        dsp.Frame
}

func newAllpassReverb(sizes []int) (*allpassReverb, error) {
	m := &allpassReverb{
		a:       NewInBuffer("a", dsp.Float64(0)),
		b:       NewInBuffer("b", dsp.Float64(0)),
		defuse:  NewInBuffer("defuse", dsp.Float64(0.5)),
		cutoff:  NewInBuffer("cutoff", dsp.Frequency(800)),
		decay:   NewInBuffer("decay", dsp.Float64(0.5)),
		bias:    NewInBuffer("bias", dsp.Float64(0)),
		aFilter: &dsp.SVFilter{Poles: 4, Resonance: 1},
		bFilter: &dsp.SVFilter{Poles: 4, Resonance: 1},
		ap:      make([]*dsp.AllPass, len(sizes)),
		aAP:     make([]*dsp.AllPass, 2),
		bAP:     make([]*dsp.AllPass, 2),
		aOut:    dsp.NewFrame(),
		bOut:    dsp.NewFrame(),
	}

	for i, s := range sizes {
		m.ap[i] = dsp.NewAllPass(s)
	}

	m.aAP[0] = dsp.NewAllPass(1653)
	m.aAP[1] = dsp.NewAllPass(2038)
	m.aDL = dsp.NewDelayLine(3411)

	m.bAP[0] = dsp.NewAllPass(1913)
	m.bAP[1] = dsp.NewAllPass(1663)
	m.bDL = dsp.NewDelayLine(4782)

	return m, m.Expose(
		"AllPassReverb",
		[]*In{m.a, m.b, m.defuse, m.cutoff, m.bias, m.decay},
		[]*Out{
			{Name: "a", Provider: provideCopyOut(m, &m.aOut)},
			{Name: "b", Provider: provideCopyOut(m, &m.bOut)},
		},
	)
}

func (m *allpassReverb) Process(out dsp.Frame) {
	m.incrRead(func() {
		a := m.a.ProcessFrame()
		b := m.b.ProcessFrame()
		defuse := m.defuse.ProcessFrame()
		cutoff := m.cutoff.ProcessFrame()
		bias := m.bias.ProcessFrame()
		decay := m.decay.ProcessFrame()

		for i := range out {
			baseDefuse := dsp.Clamp(defuse[i], 0.4, 0.6)

			d := m.ap[0].Tick(a[i]+b[i], baseDefuse+0.25)
			d = m.ap[1].Tick(d, baseDefuse+0.25)
			d = m.ap[2].Tick(d, baseDefuse+0.125)
			d = m.ap[3].Tick(d, baseDefuse+0.125)

			aOut := d + (m.bLast * decay[i])
			m.aFilter.Cutoff = cutoff[i]
			aOut, _, _ = m.aFilter.Tick(aOut)
			aOut = m.aAP[0].Tick(aOut, -baseDefuse-0.2)
			aOut = m.aAP[1].Tick(aOut, baseDefuse)
			aOut = m.aDL.Tick(aOut)
			m.aLast = aOut

			bOut := d + (m.aLast * decay[i])
			m.bFilter.Cutoff = cutoff[i]
			bOut, _, _ = m.bFilter.Tick(bOut)
			bOut = m.bAP[0].Tick(bOut, -baseDefuse-0.2)
			bOut = m.bAP[1].Tick(bOut, baseDefuse)
			bOut = m.bDL.Tick(bOut)
			m.bLast = bOut

			m.aOut[i] = dsp.AttenSum(bias[i], a[i], aOut)
			m.bOut[i] = dsp.AttenSum(bias[i], b[i], bOut)
		}
	})
}
