package module

func init() {
	Register("Concurrent", func(Config) (Patcher, error) { return newConcurrent() })
}

type concurrent struct {
	IO
	in   *In
	ch   chan Frame
	stop chan struct{}
}

func newConcurrent() (*concurrent, error) {
	m := &concurrent{
		in:   &In{Name: "input", Source: NewBuffer(zero)},
		ch:   make(chan Frame),
		stop: make(chan struct{}),
	}
	go m.readInput()
	return m, m.Expose(
		"Concurrent",
		[]*In{m.in},
		[]*Out{
			&Out{Name: "output", Provider: Provide(m)},
		})
}

func (c *concurrent) readInput() {
	for {
		select {
		case <-c.stop:
			return
		case c.ch <- c.in.ReadFrame():
		}
	}
}

func (c *concurrent) Read(out Frame) {
	frame := <-c.ch
	for i := range out {
		out[i] = frame[i]
	}
}

func (c *concurrent) Close() error {
	close(c.stop)
	return nil
}
