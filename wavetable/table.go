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
	return t.StepCatmullRom()
}

func (t *Table) StepCatmullRom() float64 {
	x := int(t.phase * float64(t.size))
	t.incr()

	i := int(x)
	x -= i

	j := i - 1
	if j < 0 {
		j += t.size
	}
	y0 := t.tables[t.offset+j]
	y1 := t.tables[t.offset+i]

	i = (i + 1) % t.size
	y2 := t.tables[t.offset+i]

	i = (i + 1) % t.size
	y3 := t.tables[t.offset+i]

	c0 := y1
	c1 := 0.5 * (y2 - y0)
	c2 := y0 - 2.5*y1 + 2*y2 - 0.5*y3
	c3 := 0.5*(y3-y0) + 1.5*(y1-y2)

	fx := float64(x)
	return ((c3*fx+c2)*fx+c1)*fx + c0
}
