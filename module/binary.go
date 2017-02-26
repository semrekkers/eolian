package module

import "math"

func init() {
	Register("Multiply", func(Config) (Patcher, error) { return newBinary("Multiply", multiply, 0, 0) })
	Register("Divide", func(Config) (Patcher, error) { return newBinary("Divide", divide, 0, 1) })
	Register("Sum", func(Config) (Patcher, error) { return newBinary("Sum", sum, 0, 0) })
	Register("Difference", func(Config) (Patcher, error) { return newBinary("Difference", diff, 0, 0) })
	Register("Mod", func(Config) (Patcher, error) { return newBinary("Mod", mod, 0, 1) })
	Register("OR", func(Config) (Patcher, error) { return newBinary("OR", or, 0, 0) })
	Register("XOR", func(Config) (Patcher, error) { return newBinary("XOR", xor, 0, 0) })
	Register("AND", func(Config) (Patcher, error) { return newBinary("AND", and, 0, 0) })
	Register("Max", func(Config) (Patcher, error) { return newBinary("Max", max, 0, 0) })
	Register("Min", func(Config) (Patcher, error) { return newBinary("Min", min, 0, 0) })
}

type binary struct {
	IO
	a, b *In
	op   binaryOp
}

func newBinary(name string, op binaryOp, a, b Value) (*binary, error) {
	m := &binary{
		a:  &In{Name: "a", Source: a},
		b:  &In{Name: "b", Source: NewBuffer(b)},
		op: op,
	}
	err := m.Expose(
		name,
		[]*In{m.a, m.b},
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
	return m, err
}

type binaryOp func(Value, Value) Value

func (bin *binary) Read(out Frame) {
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
func max(a, b Value) Value {
	if a > b {
		return a
	}
	return b
}
func min(a, b Value) Value {
	if a < b {
		return a
	}
	return b
}
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
