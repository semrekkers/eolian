package module

import (
	"fmt"
	"strconv"
	"strings"
)

type IO struct {
	ins  map[string]*In
	outs map[string]*Out
}

func (io *IO) Expose(ins []*In, outs []*Out) error {
	io.lazyInit()
	for _, in := range ins {
		if _, ok := io.ins[in.Name]; ok {
			return fmt.Errorf(`duplicate input exposed "%s"`, in.Name)
		}
		in.io = io
		if b, ok := in.Source.(*Buffer); ok {
			in.initial = b.Reader
		} else {
			in.initial = in.Source
		}
		io.ins[in.Name] = in
	}
	for _, out := range outs {
		if _, ok := io.outs[out.Name]; ok {
			return fmt.Errorf(`duplicate output exposed "%s"`, out.Name)
		}
		if out.Provider == nil {
			return fmt.Errorf(`provider must be set for output "%s"`, out.Name)
		}
		out.io = io
		io.outs[out.Name] = out
	}
	return nil
}

func (io *IO) Unpatch(name string) error {
	io.lazyInit()
	return io.Patch(name, zero)
}

func (inout *IO) Patch(name string, t interface{}) error {
	inout.lazyInit()
	input, ok := inout.ins[name]
	if !ok {
		return fmt.Errorf(`unknown input "%s"`, name)
	}

	if err := input.Close(); err != nil {
		return err
	}

	var (
		reader Reader
		err    error
	)

	switch v := t.(type) {
	case Port:
		if reader, err = v.Patcher.Output(v.Port); err != nil {
			return err
		}
	case Value:
		reader = v
	case Pitch:
		reader = v
	case Hz:
		reader = v
	case MS:
		reader = v
	case string:
		if floatV, err := strconv.ParseFloat(v, 64); err == nil {
			reader = Value(floatV)
		} else if intV, err := strconv.Atoi(v); err == nil {
			reader = Value(intV)
		} else {
			r, err := ParseValueString(v)
			if err != nil {
				return err
			}
			reader = r.(ReadValuer)
		}
	case int:
		reader = Value(v)
	case float64:
		reader = Value(v)
	case Reader:
		reader = v
	default:
		return fmt.Errorf(`unpatchable source value %v (%T)`, v, v)
	}

	input.SetSource(reader)

	return nil
}

func (io *IO) Inputs() map[string]*In {
	io.lazyInit()
	return io.ins
}

func (io *IO) Outputs() map[string]*Out {
	io.lazyInit()
	return io.outs
}

func (io *IO) Output(name string) (Reader, error) {
	io.lazyInit()
	if o, ok := io.outs[name]; ok {
		if o.IsActive() {
			return nil, fmt.Errorf(`output "%s" is already patched`, name)
		}

		r := &ReaderCloser{io, name, o.Provider.Reader()}
		io.outs[name].reader = r
		return r, nil
	}
	return nil, fmt.Errorf(`output "%s" doesn't exist`, name)
}

func (io *IO) OutputsActive() int {
	io.lazyInit()
	var i int
	for _, out := range io.outs {
		if out.IsActive() {
			i++
		}
	}
	return i
}

func (io *IO) String() string {
	out := "inputs:\n"
	for name, in := range io.ins {
		if v, ok := in.Source.(fmt.Stringer); ok {
			out += fmt.Sprintf("- %s: %s\n", name, v.String())
		} else {
			out += fmt.Sprintf("- %s\n", name)
		}
	}
	out += "outputs:\n"
	for name, e := range io.outs {
		if e.IsActive() {
			out += fmt.Sprintf("- %s (active)\n", name)
		} else {
			out += fmt.Sprintf("- %s\n", name)
		}
	}
	return strings.TrimRight(out, "\n")
}

func (inout *IO) closeOutput(name string) error {
	inout.lazyInit()
	if o, ok := inout.outs[name]; ok {
		o.reader = nil
		return nil
	}
	return fmt.Errorf(`output "%s" doesn't exist`, name)
}

func (io *IO) lazyInit() {
	if io.ins == nil {
		io.ins = map[string]*In{}
	}
	if io.outs == nil {
		io.outs = map[string]*Out{}
	}
}

func (io *IO) Reset() error {
	for _, in := range io.ins {
		if nested, ok := in.Source.(*In); ok {
			if err := nested.Close(); err != nil {
				return err
			}
		} else {
			if err := in.Close(); err != nil {
				return err
			}
			in.SetSource(in.initial)
		}
	}
	return nil
}

type Patcher interface {
	Patch(string, interface{}) error
	Output(string) (Reader, error)
	Reset() error
}

type Lister interface {
	Inputs() map[string]*In
	Outputs() map[string]*Out
}

type In struct {
	Source Reader
	Name   string

	io      *IO
	initial Reader
}

func (reader *In) Read(f Frame) {
	reader.Source.Read(f)
}

func (setter *In) SetSource(r Reader) {
	switch v := setter.Source.(type) {
	case SourceSetter:
		v.SetSource(r)
	case Reader:
		setter.Source = r
	}
}

func (i *In) ReadFrame() Frame {
	return i.Source.(*Buffer).ReadFrame()
}

func (i *In) LastFrame() Frame {
	return i.Source.(*Buffer).Frame
}

func (i *In) Close() error {
	if closer, ok := i.Source.(Closer); ok {
		return closer.Close()
	}
	return nil
}

type Out struct {
	Name     string
	Provider ReaderProvider

	io     *IO
	reader Reader
}

func (o *Out) IsActive() bool {
	return o.reader != nil
}

func (closer *Out) Close() error {
	if v, ok := closer.reader.(Closer); ok {
		return v.Close()
	}
	closer.reader = nil
	return nil
}

type ReaderCloser struct {
	io   *IO
	Name string
	Reader
}

func (closer *ReaderCloser) Close() error {
	return closer.io.closeOutput(closer.Name)
}

type Port struct {
	Patcher
	Port string
}
