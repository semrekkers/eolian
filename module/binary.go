package module

func init() {
	Register("BinaryMultiply", func(Config) (Patcher, error) { return NewBinary(Multiply) })
	Register("BinaryDivide", func(Config) (Patcher, error) { return NewBinary(Divide) })
	Register("BinarySum", func(Config) (Patcher, error) { return NewBinary(Sum) })
	Register("BinaryDifference", func(Config) (Patcher, error) { return NewBinary(Diff) })
	Register("BinaryOR", func(Config) (Patcher, error) { return NewBinary(OR) })
	Register("BinaryXOR", func(Config) (Patcher, error) { return NewBinary(XOR) })
	Register("BinaryAND", func(Config) (Patcher, error) { return NewBinary(AND) })
}

type Binary struct {
	IO
	a, b *In
	op   BinaryOp
}

func NewBinary(op BinaryOp) (*Binary, error) {
	m := &Binary{
		a:  &In{Name: "a", Source: NewBuffer(zero)},
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
	a := reader.a.ReadFrame()
	b := reader.b.ReadFrame()
	for i := range out {
		out[i] = reader.op(a[i], b[i])
	}
}

func Multiply(a, b Value) Value { return a * b }
func Divide(a, b Value) Value   { return a / b }
func Sum(a, b Value) Value      { return a + b }
func Diff(a, b Value) Value     { return a - b }
func AND(a, b Value) Value {
	if a > 0 && b > 0 {
		return 1
	}
	return -1
}
func OR(a, b Value) Value {
	if a > 0 || b > 0 {
		return 1
	}
	return -1
}
func XOR(a, b Value) Value {
	if (a > 0 && b <= 0) || (a <= 0 && b > 0) {
		return 1
	}
	return -1
}
