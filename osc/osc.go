package osc

import (
	"fmt"
	"net"

	"github.com/brettbuddin/eolian/module"
	"github.com/hypebeast/go-osc/osc"
	"github.com/mitchellh/mapstructure"
)

func init() {
	module.Register("OSCServer", func(c module.Config) (module.Patcher, error) {
		var config struct {
			Port      int
			Addresses []address
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		return NewServer(config.Port, config.Addresses)
	})
}

type address struct {
	Path   string
	Interp string
	Max    float64
	Min    float64
}

type Server struct {
	module.IO
	*osc.Server

	listener net.PacketConn
	values   map[string]chan module.Value
}

func NewServer(port int, addresses []address) (*Server, error) {
	io := &Server{
		Server: &osc.Server{
			Dispatcher: osc.NewOscDispatcher(),
		},
		values: map[string]chan module.Value{},
	}

	outs := []*module.Out{}
	for _, addr := range addresses {
		io.values[addr.Path] = make(chan module.Value, 100)
		func(addr address) {
			outs = append(outs, &module.Out{
				Name: addr.Path,
				Provider: module.ReaderProviderFunc(func() module.Reader {
					return io.newOut(addr)
				}),
			})
		}(addr)
	}

	listener, err := net.ListenPacket("udp", fmt.Sprintf(":%d", port))
	if err != nil {
		return io, err
	}
	io.listener = listener

	err = io.Expose(nil, outs)
	go io.Serve(listener)
	return io, err
}

func (s *Server) Close() error {
	if s.listener != nil {
		err := s.listener.Close()
		s.listener = nil
		return err
	}
	return nil
}

func (s *Server) newOut(addr address) *serverOut {
	var (
		isScaled  bool
		scaleDiff float64

		values  = s.values[addr.Path]
		interp  = determineInterp(addr.Interp)
		initial = module.Value(addr.Min)
	)

	if interp == interpGate {
		initial = -1
	}

	if addr.Max != 0 {
		isScaled = true
		scaleDiff = addr.Max - addr.Min
	}

	s.Dispatcher.AddMsgHandler(addr.Path, func(msg *osc.Message) {
		if len(msg.Arguments) != 1 {
			return
		}
		if msg.Address != addr.Path {
			return
		}
		switch raw := msg.Arguments[0].(type) {
		case float32:
			v := float64(raw)
			if isScaled {
				v = (scaleDiff * v) + addr.Min
			}

			switch interp {
			case interpRaw:
				values <- module.Value(v)
			case interpMS:
				values <- module.Duration(v).Value()
			case interpHz:
				values <- module.Frequency(v).Value()
			case interpGate:
				if v == 1 {
					values <- 1
				} else {
					values <- -1
				}
			}
		}
	})

	return &serverOut{
		Server: s,
		values: values,
		interp: interp,
		last:   initial,
	}
}

const (
	interpRaw interpolation = iota
	interpHz
	interpMS
	interpGate
)

type interpolation int

type serverOut struct {
	*Server
	values chan module.Value
	interp interpolation
	last   module.Value
}

func (reader *serverOut) Read(out module.Frame) {
	for i := range out {
		select {
		case v := <-reader.values:
			if reader.interp == interpGate && reader.last == v {
				v = -1
			}
			out[i] = v
			reader.last = out[i]
		default:
			out[i] = reader.last
		}
	}
}

func determineInterp(v string) interpolation {
	switch v {
	case "":
		return interpRaw
	case "ms":
		return interpMS
	case "hz":
		return interpHz
	case "gate":
		return interpGate
	default:
		return interpRaw
	}
}
