package module

func init() {
	Register("BinaryMultiply", func(Config) (Patcher, error) { return NewBinary(multiply) })
	Register("BinaryDivide", func(Config) (Patcher, error) { return NewBinary(divide) })
	Register("BinarySum", func(Config) (Patcher, error) { return NewBinary(sum) })
	Register("BinaryDifference", func(Config) (Patcher, error) { return NewBinary(diff) })
	Register("BinaryOR", func(Config) (Patcher, error) { return NewBinary(or) })
	Register("BinaryXOR", func(Config) (Patcher, error) { return NewBinary(xor) })
	Register("BinaryAND", func(Config) (Patcher, error) { return NewBinary(and) })
}

type Binary struct {
	IO
	a, b *In
	op   BinaryOp
}

func NewBinary(op BinaryOp) (*Binary, error) {
	m := &Binary{
		a:  &In{Name: "a", Source: zero},
		b:  &In{Name: "b", Source: NewBuffer(zero)},
		op: op,
	}
	err := m.Expose(
		[]*In{m.a, m.b},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

type BinaryOp func(Value, Value) Value

func (reader *Binary) Read(out Frame) {
	reader.a.Read(out)
	b := reader.b.ReadFrame()
	for i := range out {
		out[i] = reader.op(out[i], b[i])
	}
}

func multiply(a, b Value) Value { return a * b }
func divide(a, b Value) Value   { return a / b }
func sum(a, b Value) Value      { return a + b }
func diff(a, b Value) Value     { return a - b }
func and(a, b Value) Value {
	if a > 0 && b > 0 {
		return 1
	}
	return -1
}
func or(a, b Value) Value {
	if a > 0 || b > 0 {
		return 1
	}
	return -1
}
func xor(a, b Value) Value {
	if (a > 0 && b <= 0) || (a <= 0 && b > 0) {
		return 1
	}
	return -1
}
