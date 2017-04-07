// Package osc provides Open Sound Control (OSC) input handling
package osc

import (
	"fmt"
	"net"

	"buddin.us/eolian/dsp"
	"buddin.us/eolian/module"
	"github.com/hypebeast/go-osc/osc"
	"github.com/mitchellh/mapstructure"
)

func init() {
	module.Register("OSCServer", func(c module.Config) (module.Patcher, error) {
		var config oscConfig
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		return newServer(config)
	})
}

type oscConfig struct {
	Port       int
	Addresses  []address
	ClientHost string
	ClientPort int
}

type address struct {
	Path   string
	Interp string
	Max    float64
	Min    float64
}

type server struct {
	module.IO
	*osc.Server
	client *osc.Client

	listener net.PacketConn
	values   map[string]chan dsp.Float64
}

func newServer(c oscConfig) (*server, error) {
	io := &server{
		Server: &osc.Server{
			Dispatcher: osc.NewOscDispatcher(),
		},
		values: map[string]chan dsp.Float64{},
	}

	if c.ClientHost != "" && c.ClientPort > 0 {
		io.client = osc.NewClient(c.ClientHost, c.ClientPort)
	}

	outs := []*module.Out{}
	for _, addr := range c.Addresses {
		io.values[addr.Path] = make(chan dsp.Float64, 100)
		func(addr address) {
			outs = append(outs, &module.Out{
				Name:     addr.Path,
				Provider: dsp.Provide(io.newOut(addr)),
			})
		}(addr)

		if io.client != nil {
			msg := osc.NewMessage(addr.Path)
			msg.Append(int32(0))
			io.client.Send(msg)
		}
	}

	listener, err := net.ListenPacket("udp", fmt.Sprintf(":%d", c.Port))
	if err != nil {
		return io, err
	}
	io.listener = listener

	go io.Serve(listener)
	return io, io.Expose("OSCServer", nil, outs)

}

func (s *server) Close() error {
	if s.listener != nil {
		err := s.listener.Close()
		s.listener = nil
		return err
	}
	return nil
}

func (s *server) newOut(addr address) *serverOut {
	var (
		isScaled  bool
		scaleDiff float64

		values  = s.values[addr.Path]
		interp  = determineInterp(addr.Interp)
		initial = dsp.Float64(addr.Min)
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
				values <- dsp.Float64(v)
			case interpMS:
				values <- dsp.Duration(v).Value()
			case interpHz:
				values <- dsp.Frequency(v).Value()
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
		server: s,
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
	*server
	values chan dsp.Float64
	interp interpolation
	last   dsp.Float64
}

func (o *serverOut) Process(out dsp.Frame) {
	for i := range out {
		select {
		case v := <-o.values:
			if o.interp == interpGate && o.last == v {
				v = -1
			}
			out[i] = v
			o.last = out[i]
		default:
			out[i] = o.last
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
