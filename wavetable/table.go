package wavetable

//go:generate go run cmd/gen/main.go

type Table struct {
	tables       []float64
	sampleRate   float64
	offset, size int
	phase, delta float64
}

func NewTable(tables []float64, size int, sampleRate float64) *Table {
	return &Table{
		sampleRate: sampleRate,
		tables:     tables,
		size:       size,
	}
}

func (t *Table) SetDelta(f float64) {
	var target int
	for i, b := range Breakpoints {
		if f <= b {
			target = i
			break
		}
	}
	t.offset = ((len(Breakpoints) - 1) - target) * t.size
	t.delta = f
}

func (t *Table) incr() {
	t.phase += t.delta
	if t.phase > 1.0 {
		t.phase -= t.phase
	}
}

func (t *Table) Step() float64 {
	x := int(t.phase * float64(t.size))
	t.incr()

	i := int(x)
	x -= i

	j := i + 1
	if j >= t.size {
		j = 0
	}

	return float64(1-x)*t.tables[t.offset+i] + float64(x)*t.tables[t.offset+j]
}
