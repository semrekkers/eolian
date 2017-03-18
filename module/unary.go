package module

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

func newUnary(name string, op unaryOp, input Value) (*unary, error) {
	m := &unary{
		in: &In{Name: "input", Source: input},
		op: op,
	}
	err := m.Expose(
		name,
		[]*In{m.in},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

type unaryOp func(Value) Value

func (u *unary) Read(out Frame) {
	u.in.Read(out)
	for i := range out {
		out[i] = u.op(out[i])
	}
}

func round(in Value) Value { return floorValue(in + 0.5) }
func floor(in Value) Value { return floorValue(in) }
func ceil(in Value) Value  { return ceilValue(in) }
