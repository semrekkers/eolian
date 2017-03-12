package module

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

func provideCopyOut(r Reader, cache *Frame) ReaderProvider {
	return ReaderProviderFunc(func() Reader {
		return &copyOut{reader: r, cache: cache}
	})
}

type copyOut struct {
	reader Reader
	cache  *Frame
}

func (o *copyOut) Read(out Frame) {
	o.reader.Read(out)
	copy(out, *o.cache)
}
