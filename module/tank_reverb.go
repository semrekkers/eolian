package module

import (
	"buddin.us/eolian/dsp"
)

func init() {
	Register("TankReverb", func(c Config) (Patcher, error) { return newTankReverb() })
}

type tankReverb struct {
	multiOutIO

	a, b, defuse, bias, cutoff, decay *In
	aFilter                           *dsp.SVFilter
	bFilter                           *dsp.SVFilter
	ap, aAP, bAP                      []*dsp.AllPass
	aDL, bDL                          *dsp.TappedDelayLine
	aLast, bLast                      dsp.Float64
	aOut, bOut                        dsp.Frame
}

func newTankReverb() (*tankReverb, error) {
	m := &tankReverb{
		a:       NewInBuffer("a", dsp.Float64(0)),
		b:       NewInBuffer("b", dsp.Float64(0)),
		defuse:  NewInBuffer("defuse", dsp.Float64(0.5)),
		cutoff:  NewInBuffer("cutoff", dsp.Frequency(800)),
		decay:   NewInBuffer("decay", dsp.Float64(0.5)),
		bias:    NewInBuffer("bias", dsp.Float64(0)),
		aFilter: &dsp.SVFilter{Poles: 2, Resonance: 1},
		bFilter: &dsp.SVFilter{Poles: 2, Resonance: 1},
		ap:      make([]*dsp.AllPass, 4),
		aAP:     make([]*dsp.AllPass, 2),
		bAP:     make([]*dsp.AllPass, 2),
		aOut:    dsp.NewFrame(),
		bOut:    dsp.NewFrame(),
	}

	m.ap[0] = dsp.NewAllPass(113)
	m.ap[1] = dsp.NewAllPass(162)
	m.ap[2] = dsp.NewAllPass(241)
	m.ap[3] = dsp.NewAllPass(399)

	m.aAP[0] = dsp.NewAllPass(1653)
	m.aAP[1] = dsp.NewAllPass(2038)
	m.aDL = dsp.NewTappedDelayLine([]int{1913, 3411})

	m.bAP[0] = dsp.NewAllPass(1913)
	m.bAP[1] = dsp.NewAllPass(1663)
	m.bDL = dsp.NewTappedDelayLine([]int{1653, 4782})

	return m, m.Expose(
		"AllPassReverb",
		[]*In{m.a, m.b, m.defuse, m.cutoff, m.bias, m.decay},
		[]*Out{
			{Name: "a", Provider: provideCopyOut(m, &m.aOut)},
			{Name: "b", Provider: provideCopyOut(m, &m.bOut)},
		},
	)
}

func (m *tankReverb) Process(out dsp.Frame) {
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

			var (
				aOut, bOut dsp.Float64
			)

			aSig := d + (m.bLast * decay[i])
			m.aFilter.Cutoff = cutoff[i]
			aSig, _, _ = m.aFilter.Tick(aSig)
			aOut += aSig * 0.5
			aSig = m.aAP[0].Tick(aSig, -baseDefuse-0.2)
			aOut += aSig * 0.5
			aSig = m.aAP[1].Tick(aSig*decay[i], baseDefuse)

			aTaps := m.aDL.Tick(aSig)
			aOut -= aTaps[0]
			aOut += aTaps[1]
			m.aLast = aTaps[1]

			bSig := d + (m.aLast * decay[i])
			m.bFilter.Cutoff = cutoff[i]
			bSig, _, _ = m.bFilter.Tick(bSig)
			bOut += bSig * 0.5
			bSig = m.bAP[0].Tick(bSig, -baseDefuse-0.2)
			bOut += bSig * 0.5
			bSig = m.bAP[1].Tick(bSig*decay[i], baseDefuse)

			bTaps := m.bDL.Tick(bSig)
			bOut -= bTaps[0]
			bOut += bTaps[1]
			m.bLast = bTaps[1]

			m.aOut[i] = dsp.AttenSum(bias[i], a[i], aOut)
			m.bOut[i] = dsp.AttenSum(bias[i], b[i], bOut)
		}
	})
}
