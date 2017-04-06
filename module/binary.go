package module

import (
	"math"

	"buddin.us/eolian/dsp"
)

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

func newBinary(name string, op binaryOp, a, b dsp.Float64) (*binary, error) {
	m := &binary{
		a:  NewIn("a", a),
		b:  NewInBuffer("b", b),
		op: op,
	}
	err := m.Expose(
		name,
		[]*In{m.a, m.b},
		[]*Out{{Name: "output", Provider: dsp.Provide(m)}},
	)
	return m, err
}

type binaryOp func(dsp.Float64, dsp.Float64) dsp.Float64

func (bin *binary) Process(out dsp.Frame) {
	bin.a.Process(out)
	b := bin.b.ProcessFrame()
	for i := range out {
		out[i] = bin.op(out[i], b[i])
	}
}

func diff(a, b dsp.Float64) dsp.Float64     { return a - b }
func divide(a, b dsp.Float64) dsp.Float64   { return a / b }
func mod(a, b dsp.Float64) dsp.Float64      { return dsp.Float64(math.Mod(float64(a), float64(b))) }
func multiply(a, b dsp.Float64) dsp.Float64 { return a * b }
func sum(a, b dsp.Float64) dsp.Float64      { return a + b }
func max(a, b dsp.Float64) dsp.Float64 {
	if a > b {
		return a
	}
	return b
}
func min(a, b dsp.Float64) dsp.Float64 {
	if a < b {
		return a
	}
	return b
}
func and(a, b dsp.Float64) dsp.Float64 {
	if a > 0 && b > 0 {
		return 1
	}
	return -1
}
func or(a, b dsp.Float64) dsp.Float64 {
	if a > 0 || b > 0 {
		return 1
	}
	return -1
}
func xor(a, b dsp.Float64) dsp.Float64 {
	if (a > 0 && b <= 0) || (a <= 0 && b > 0) {
		return 1
	}
	return -1
}
