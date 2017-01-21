package module

import "math"

func init() {
	Register("Multiply", func(Config) (Patcher, error) { return NewBinary(multiply, 0, 0) })
	Register("Divide", func(Config) (Patcher, error) { return NewBinary(divide, 0, 1) })
	Register("Sum", func(Config) (Patcher, error) { return NewBinary(sum, 0, 0) })
	Register("Difference", func(Config) (Patcher, error) { return NewBinary(diff, 0, 0) })
	Register("Mod", func(Config) (Patcher, error) { return NewBinary(mod, 0, 1) })
	Register("OR", func(Config) (Patcher, error) { return NewBinary(or, 0, 0) })
	Register("XOR", func(Config) (Patcher, error) { return NewBinary(xor, 0, 0) })
	Register("AND", func(Config) (Patcher, error) { return NewBinary(and, 0, 0) })
}

type Binary struct {
	IO
	a, b *In
	op   BinaryOp
}

func NewBinary(op BinaryOp, a, b Value) (*Binary, error) {
	m := &Binary{
		a:  &In{Name: "a", Source: a},
		b:  &In{Name: "b", Source: NewBuffer(b)},
		op: op,
	}
	err := m.Expose(
		[]*In{m.a, m.b},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

type BinaryOp func(Value, Value) Value

func (bin *Binary) Read(out Frame) {
	bin.a.Read(out)
	b := bin.b.ReadFrame()
	for i := range out {
		out[i] = bin.op(out[i], b[i])
	}
}

func diff(a, b Value) Value     { return a - b }
func divide(a, b Value) Value   { return a / b }
func mod(a, b Value) Value      { return Value(math.Mod(float64(a), float64(b))) }
func multiply(a, b Value) Value { return a * b }
func sum(a, b Value) Value      { return a + b }
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
