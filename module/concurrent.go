package module

func init() {
	Register("Concurrent", func(Config) (Patcher, error) { return NewConcurrent() })
}

type Concurrent struct {
	IO
	in   *In
	ch   chan Value
	stop chan struct{}
}

func NewConcurrent() (*Concurrent, error) {
	m := &Concurrent{
		in:   &In{Name: "input", Source: NewBuffer(zero)},
		ch:   make(chan Value),
		stop: make(chan struct{}),
	}
	go m.readInput()
	return m, m.Expose(
		[]*In{m.in},
		[]*Out{
			&Out{Name: "output", Provider: Provide(m)},
		})
}

func (c *Concurrent) readInput() {
	for {
		frame := c.in.ReadFrame()
		for _, v := range frame {
			select {
			case c.ch <- v:
			case <-c.stop:
				return
			}
		}
	}
}

func (reader *Concurrent) Read(out Frame) {
	for i := range out {
		out[i] = <-reader.ch
	}
}

func (reader *Concurrent) Close() error {
	close(reader.stop)
	return nil
}
