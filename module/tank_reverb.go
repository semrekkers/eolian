package module

import (
	"buddin.us/eolian/dsp"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("TankReverb", func(c Config) (Patcher, error) {
		var config struct {
			PolesA int `mapstructure:"polesA"`
			PolesB int `mapstructure:"polesB"`
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.PolesA == 0 {
			config.PolesA = 4
		}
		if config.PolesB == 0 {
			config.PolesB = 4
		}
		return newTankReverb(config.PolesA, config.PolesB)
	})
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

func newTankReverb(polesA, polesB int) (*tankReverb, error) {
	m := &tankReverb{
		a:       NewInBuffer("a", dsp.Float64(0)),
		b:       NewInBuffer("b", dsp.Float64(0)),
		defuse:  NewInBuffer("defuse", dsp.Float64(0.5)),
		cutoff:  NewInBuffer("cutoff", dsp.Frequency(800)),
		decay:   NewInBuffer("decay", dsp.Float64(0.5)),
		bias:    NewInBuffer("bias", dsp.Float64(0)),
		aFilter: &dsp.SVFilter{Poles: polesA, Resonance: 1},
		bFilter: &dsp.SVFilter{Poles: polesB, Resonance: 1},
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
		defuseIn := m.defuse.ProcessFrame()
		cutoff := m.cutoff.ProcessFrame()
		biasIn := m.bias.ProcessFrame()
		decayIn := m.decay.ProcessFrame()

		for i := range out {
			defuse := dsp.Clamp(defuseIn[i], 0.4, 0.6)
			decay := dsp.Clamp(decayIn[i], 0, 0.9)
			bias := dsp.Clamp(biasIn[i], -1, 1)

			d := m.ap[0].Tick(a[i]+b[i], defuse+0.25)
			d = m.ap[1].Tick(d, defuse+0.25)
			d = m.ap[2].Tick(d, defuse+0.125)
			d = m.ap[3].Tick(d, defuse+0.125)

			var (
				aOut, bOut dsp.Float64
			)

			aSig := d + (m.bLast * decay)
			m.aFilter.Cutoff = cutoff[i]
			aSig, _, _ = m.aFilter.Tick(aSig)
			aOut += aSig * 0.5
			aSig = m.aAP[0].Tick(aSig, -defuse-0.2)
			aOut += aSig * 0.5
			aSig = m.aAP[1].Tick(aSig*decay, defuse)

			aTaps := m.aDL.Tick(aSig)
			aOut -= aTaps[0]
			aOut += aTaps[1]
			m.aLast = aTaps[1]

			bSig := d + (m.aLast * decay)
			m.bFilter.Cutoff = cutoff[i]
			bSig, _, _ = m.bFilter.Tick(bSig)
			bOut += bSig * 0.5
			bSig = m.bAP[0].Tick(bSig, -defuse-0.2)
			bOut += bSig * 0.5
			bSig = m.bAP[1].Tick(bSig*decay, defuse)

			bTaps := m.bDL.Tick(bSig)
			bOut -= bTaps[0]
			bOut += bTaps[1]
			m.bLast = bTaps[1]

			m.aOut[i] = dsp.AttenSum(bias, a[i], aOut)
			m.bOut[i] = dsp.AttenSum(bias, b[i], bOut)
		}
	})
}
