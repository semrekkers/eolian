package module

func init() {
	Register("Concurrent", func(Config) (Patcher, error) { return NewConcurrent() })
}

type Concurrent struct {
	IO
	in   *In
	ch   chan Frame
	stop chan struct{}
}

func NewConcurrent() (*Concurrent, error) {
	m := &Concurrent{
		in:   &In{Name: "input", Source: NewBuffer(zero)},
		ch:   make(chan Frame),
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
		select {
		case <-c.stop:
			return
		case c.ch <- c.in.ReadFrame():
		}
	}
}

func (reader *Concurrent) Read(out Frame) {
	frame := <-reader.ch
	for i := range out {
		out[i] = frame[i]
	}
}

func (reader *Concurrent) Close() error {
	close(reader.stop)
	return nil
}
