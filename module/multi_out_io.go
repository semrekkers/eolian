package module

import "buddin.us/eolian/dsp"

type multiOutIO struct {
	IO
	reads int
}

func (io *multiOutIO) incrRead(read func()) {
	if io.reads > 0 {
		io.incr()
		return
	}
	read()
	io.incr()
}

func (io *multiOutIO) incr() {
	if outs := io.IO.OutputsActive(true); outs > 0 {
		io.reads = (io.reads + 1) % outs
	}
}

func provideCopyOut(r dsp.Processor, cache *dsp.Frame) dsp.ProcessorProvider {
	return dsp.ProcessorProviderFunc(func() dsp.Processor {
		return &copyOut{reader: r, cache: cache}
	})
}

type copyOut struct {
	reader dsp.Processor
	cache  *dsp.Frame
}

func (o *copyOut) Process(out dsp.Frame) {
	o.reader.Process(out)
	copy(out, *o.cache)
}
