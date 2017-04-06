package module

import (
	"sync/atomic"

	"buddin.us/eolian/dsp"
)

func init() {
	Register("Concurrent", func(Config) (Patcher, error) { return newConcurrent() })
}

type concurrent struct {
	IO
	in      *In
	ch      chan dsp.Frame
	stop    chan struct{}
	running atomic.Value
}

func newConcurrent() (*concurrent, error) {
	m := &concurrent{
		in:   NewInBuffer("input", dsp.Float64(0)),
		ch:   make(chan dsp.Frame),
		stop: make(chan struct{}),
	}
	m.running.Store(true)
	go m.readInput()
	return m, m.Expose(
		"Concurrent",
		[]*In{m.in},
		[]*Out{
			&Out{Name: "output", Provider: dsp.Provide(m)},
		})
}

func (c *concurrent) readInput() {
	for {
		select {
		case <-c.stop:
			return
		case c.ch <- c.in.ProcessFrame():
		}
	}
}

func (c *concurrent) Process(out dsp.Frame) {
	frame := <-c.ch
	for i := range out {
		out[i] = frame[i]
	}
}

func (c *concurrent) Patch(name string, t interface{}) error {
	if !c.running.Load().(bool) {
		c.stop = make(chan struct{})
		go c.readInput()
		c.running.Store(true)
	}
	return c.IO.Patch(name, t)
}

func (c *concurrent) Close() error {
	close(c.stop)
	c.running.Store(false)
	return nil
}
