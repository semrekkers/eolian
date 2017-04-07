package module

import "buddin.us/eolian/dsp"

func init() {
	Register("Round", func(Config) (Patcher, error) { return newUnary("Round", round, 0) })
	Register("Floor", func(Config) (Patcher, error) { return newUnary("Floor", floor, 0) })
	Register("Ceil", func(Config) (Patcher, error) { return newUnary("Ceil", ceil, 0) })
}

type unary struct {
	IO
	in *In
	op unaryOp
}

func newUnary(name string, op unaryOp, input dsp.Float64) (*unary, error) {
	m := &unary{
		in: NewIn("input", input),
		op: op,
	}
	err := m.Expose(
		name,
		[]*In{m.in},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

type unaryOp func(dsp.Float64) dsp.Float64

func (u *unary) Process(out dsp.Frame) {
	u.in.Process(out)
	for i := range out {
		out[i] = u.op(out[i])
	}
}

func round(in dsp.Float64) dsp.Float64 {
	if in < 0 {
		return dsp.Ceil(in - 0.5)
	}
	return dsp.Floor(in + 0.5)
}
func floor(in dsp.Float64) dsp.Float64 { return dsp.Floor(in) }
func ceil(in dsp.Float64) dsp.Float64  { return dsp.Ceil(in) }
