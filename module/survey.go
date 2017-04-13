package module

import (
	"buddin.us/eolian/dsp"
)

func init() {
	Register("Survey", func(c Config) (Patcher, error) { return newSurvey() })
}

var followDuration = dsp.Duration(300).Value()

type survey struct {
	multiOutIO
	a, b, survey, fade, offset,
	or1, or2, and1, and2,
	slope, crease *In

	aOut, bOut, orOut, andOut,
	slopeOut, creaseOut, follow dsp.Frame
	follower dsp.Follow
}

func newSurvey() (*survey, error) {
	m := &survey{
		a:         NewInBuffer("a", dsp.Float64(0)),
		b:         NewInBuffer("b", dsp.Float64(0)),
		survey:    NewInBuffer("survey", dsp.Float64(0)),
		fade:      NewInBuffer("fade", dsp.Float64(0)),
		offset:    NewInBuffer("offset", dsp.Float64(0)),
		or1:       NewInBuffer("or1", dsp.Float64(0)),
		or2:       NewInBuffer("or2", dsp.Float64(0)),
		and1:      NewInBuffer("and1", dsp.Float64(0)),
		and2:      NewInBuffer("and2", dsp.Float64(0)),
		slope:     NewInBuffer("slope", dsp.Float64(0)),
		crease:    NewInBuffer("crease", dsp.Float64(0)),
		aOut:      dsp.NewFrame(),
		bOut:      dsp.NewFrame(),
		orOut:     dsp.NewFrame(),
		andOut:    dsp.NewFrame(),
		slopeOut:  dsp.NewFrame(),
		creaseOut: dsp.NewFrame(),
		follow:    dsp.NewFrame(),
	}

	return m, m.Expose(
		"Survey",
		[]*In{m.a, m.b, m.survey, m.or1, m.or2, m.and1, m.and2, m.slope, m.crease, m.offset, m.fade},
		[]*Out{
			{Name: "a", Provider: provideCopyOut(m, &m.aOut)},
			{Name: "b", Provider: provideCopyOut(m, &m.bOut)},
			{Name: "or", Provider: provideCopyOut(m, &m.orOut)},
			{Name: "and", Provider: provideCopyOut(m, &m.andOut)},
			{Name: "slope", Provider: provideCopyOut(m, &m.slopeOut)},
			{Name: "crease", Provider: provideCopyOut(m, &m.creaseOut)},
			{Name: "follow", Provider: provideCopyOut(m, &m.follow)},
		},
	)
}

func (s *survey) Process(out dsp.Frame) {
	s.incrRead(func() {
		var (
			a      = s.a.ProcessFrame()
			b      = s.b.ProcessFrame()
			offset = s.offset.ProcessFrame()
			fade   = s.fade.ProcessFrame()
			or1    = s.or1.ProcessFrame()
			or2    = s.or2.ProcessFrame()
			and1   = s.and1.ProcessFrame()
			and2   = s.and2.ProcessFrame()
			survey = s.survey.ProcessFrame()
			slope  = s.slope.ProcessFrame()
			crease = s.crease.ProcessFrame()
		)

		for i := range out {
			srv := dsp.Clamp(survey[i], -1, 1)

			// A/B + Offset
			crossfade := srv
			if !isNormal(s.fade) {
				crossfade = fade[i]
			}
			if isNormal(s.a) && isNormal(s.b) {
				s.aOut[i] = dsp.AttenSum(crossfade, -1, 1)
				s.bOut[i] = dsp.AttenSum(crossfade, 1, -1)
			} else {
				s.aOut[i] = dsp.AttenSum(crossfade, a[i], 1) + dsp.AttenSum(crossfade, -1, b[i])
				s.bOut[i] = dsp.AttenSum(crossfade, b[i], -1) + dsp.AttenSum(crossfade, 1, a[i])
			}
			s.aOut[i] += offset[i]
			s.bOut[i] += offset[i]

			// OR
			if !isNormal(s.or1) && isNormal(s.or2) {
				if or1[i] >= srv {
					s.orOut[i] = or1[i]
				} else {
					s.orOut[i] = srv
				}
			} else if isNormal(s.or1) && !isNormal(s.or2) {
				if or2[i] >= srv {
					s.orOut[i] = or2[i]
				} else {
					s.orOut[i] = srv
				}
			} else if !isNormal(s.or1) && !isNormal(s.or2) {
				if or1[i] >= or2[i] {
					s.orOut[i] = or1[i]
				} else {
					s.orOut[i] = or2[i]
				}
			} else {
				s.orOut[i] = dsp.AttenSum(srv, 0, 1)
			}

			// AND
			if !isNormal(s.and1) && isNormal(s.and2) {
				if and1[i] <= srv {
					s.andOut[i] = and1[i]
				} else {
					s.andOut[i] = srv
				}
			} else if isNormal(s.and1) && !isNormal(s.and2) {
				if and2[i] <= srv {
					s.andOut[i] = and2[i]
				} else {
					s.andOut[i] = srv
				}
			} else if !isNormal(s.and1) && !isNormal(s.and2) {
				if and1[i] <= and2[i] {
					s.andOut[i] = and1[i]
				} else {
					s.andOut[i] = and2[i]
				}
			} else {
				s.andOut[i] = dsp.AttenSum(srv, -1, 0)
			}

			// Slope and Follow
			slopeFactor := dsp.Float64(1)
			if !isNormal(s.slope) {
				slopeFactor = slope[i]
			}
			s.follower.Rise = followDuration * slopeFactor
			s.follower.Fall = followDuration * slopeFactor
			s.follow[i] = s.follower.Tick(a[i] + b[i])

			if isNormal(s.slope) {
				s.slopeOut[i] = dsp.Abs(dsp.AttenSum(srv,
					dsp.AttenSum(srv, -1, 0),
					dsp.AttenSum(srv, 0, 1)))
			} else {
				s.slopeOut[i] = dsp.Abs(slope[i])
			}

			// Crease
			if isNormal(s.crease) {
				if int(10*srv)%2 == 0 {
					s.creaseOut[i] = 1
				} else {
					s.creaseOut[i] = -1
				}
			} else {
				if crease[i] > 0 {
					s.creaseOut[i] = 1 - crease[i]
				} else if crease[i] < 0 {
					s.creaseOut[i] = -1 - crease[i]
				} else {
					s.creaseOut[i] = 0
				}
			}
		}
	})
}

func isNormal(in *In) bool {
	v, ok := in.Source.(*dsp.Buffer).Processor.(dsp.Valuer)
	return ok && v == dsp.Float64(0)
}
