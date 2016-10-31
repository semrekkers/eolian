package midi

import (
	"fmt"

	"github.com/brettbuddin/eolian/module"
	"github.com/brettbuddin/musictheory"
	"github.com/mitchellh/mapstructure"
	"github.com/rakyll/portmidi"
)

var (
	pitches = map[int]module.Value{}
)

func init() {
	p := musictheory.NewPitch(musictheory.C, musictheory.Natural, 0)
	for i := 12; i < 127; i++ {
		pitches[i] = module.Frequency(p.Freq()).Value()
		p = p.Transpose(musictheory.Minor(2)).(musictheory.Pitch)
	}

	module.Register("MIDIController", func(c module.Config) (module.Patcher, error) {
		var config ControllerConfig
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}

		// Default to 16-31 CC numbers
		if len(config.CCOutputs) == 0 {
			for i := 16; i < 32; i++ {
				config.CCOutputs = append(config.CCOutputs, i)
			}
		}

		return NewController(config.Device, config.CCOutputs)
	})
}

type ControllerConfig struct {
	Device    int
	CCOutputs []int `mapstructure:"ccOutputs"`
}

type Controller struct {
	module.IO
	*portmidi.Stream

	frameRate int
	events    []portmidi.Event
	reads     int

	lastClock module.Value
	clockTick int
}

func NewController(deviceID int, ccOutputs []int) (*Controller, error) {
	initMIDI()
	stream, err := portmidi.NewInputStream(portmidi.DeviceID(deviceID), int64(module.FrameSize))
	if err != nil {
		return nil, err
	}

	m := &Controller{
		Stream:    stream,
		frameRate: 25,
		events:    make([]portmidi.Event, module.FrameSize),
	}
	outs := []*module.Out{
		{
			Name: "gate",
			Provider: module.ReaderProviderFunc(func() module.Reader {
				return &ctrlGate{
					Controller: m,
					stateFunc:  gateUp,
					state:      &gateState{control: -1},
				}
			}),
		},
		{
			Name: "pitch",
			Provider: module.ReaderProviderFunc(func() module.Reader {
				return &ctrlPitch{Controller: m}
			}),
		},
		{
			Name: "sync",
			Provider: module.ReaderProviderFunc(func() module.Reader {
				return &ctrlSync{Controller: m}
			}),
		},
		{
			Name: "reset",
			Provider: module.ReaderProviderFunc(func() module.Reader {
				return &ctrlReset{Controller: m}
			}),
		},
		{
			Name: "pitchBend",
			Provider: module.ReaderProviderFunc(func() module.Reader {
				return &ctrlPitchBend{Controller: m}
			}),
		},
		{
			Name: "modWheel",
			Provider: module.ReaderProviderFunc(func() module.Reader {
				return &ctrlCC{Controller: m, number: 1}
			}),
		},
	}

	for _, n := range ccOutputs {
		outs = append(outs, &module.Out{
			Name: fmt.Sprintf("cc/%d", n),
			Provider: module.ReaderProviderFunc(func() module.Reader {
				return &ctrlCC{Controller: m, number: n}
			}),
		})
	}

	err = m.Expose(nil, outs)
	return m, nil
}

func (c *Controller) read(out module.Frame) {
	if c.reads == 0 {
		for i := range out {
			if c.Stream == nil {
				continue
			}
			events, err := c.Stream.Read(1)
			if err != nil {
				panic(err)
			}
			if len(events) == 1 {
				c.events[i] = events[0]
			} else {
				c.events[i] = portmidi.Event{}
			}
		}
	}
	if outs := c.OutputsActive(); outs > 0 {
		c.reads = (c.reads + 1) % outs
	}
}

func (c *Controller) Close() error {
	if c.Stream != nil {
		if err := c.Stream.Close(); err != nil {
			return err
		}
		c.Stream = nil
	}
	return nil
}

type ctrlGate struct {
	*Controller
	stateFunc gateStateFunc
	state     *gateState
}

func (reader *ctrlGate) Read(out module.Frame) {
	reader.read(out)
	for i := range out {
		reader.state.event = reader.events[i]
		reader.stateFunc = reader.stateFunc(reader.state)
		out[i] = reader.state.value
	}
}

type gateState struct {
	event   portmidi.Event
	control int
	value   module.Value
}

func gateRolling(s *gateState) gateStateFunc {
	s.value = -1
	return gateDown
}

func gateDown(s *gateState) gateStateFunc {
	s.value = 1
	switch s.event.Status {
	case 144:
		if s.event.Data2 > 0 {
			if int(s.event.Data1) != s.control {
				s.control = int(s.event.Data1)
				return gateRolling
			} else {
				s.control = -1
				return gateUp
			}
		} else {
			if int(s.event.Data1) == s.control {
				s.control = -1
				return gateUp
			}
		}
	case 128:
		s.control = -1
		return gateUp
	}
	return gateDown
}

func gateUp(s *gateState) gateStateFunc {
	s.value = -1
	if s.event.Status == 144 && s.event.Data2 > 0 {
		s.control = int(s.event.Data1)
		return gateDown
	}
	return gateUp
}

type gateStateFunc func(*gateState) gateStateFunc

type ctrlSync struct {
	*Controller
	tick int
}

func (reader *ctrlSync) Read(out module.Frame) {
	reader.read(out)
	for i := range out {
		if reader.events[i].Status == 248 || reader.events[i].Status == 250 {
			reader.tick++
		}

		if reader.tick%reader.frameRate == 0 {
			out[i] = 1
			reader.tick = 0
		} else {
			out[i] = -1
		}
	}
}

type ctrlReset struct {
	*Controller
}

func (reader ctrlReset) Read(out module.Frame) {
	reader.read(out)
	for i := range out {
		if reader.events[i].Status == 250 {
			out[i] = 1
		} else {
			out[i] = -1
		}
	}
}

type ctrlPitch struct {
	*Controller
	pitch module.Value
}

func (reader *ctrlPitch) Read(out module.Frame) {
	reader.read(out)
	for i := range out {
		if reader.events[i].Status == 144 {
			data1 := int(reader.events[i].Data1)
			data2 := int(reader.events[i].Data2)
			if data2 == 0 {
				continue
			}

			if v, ok := pitches[data1]; ok {
				if data2 > 0 {
					reader.pitch = v
				}
			}
		}
		out[i] = reader.pitch
	}
}

type ctrlPitchBend struct {
	*Controller
}

func (reader *ctrlPitchBend) Read(out module.Frame) {
	reader.read(out)
	for i := range out {
		e := reader.events[i]
		if e.Status == 224 && e.Data1 == 0 {
			switch e.Data2 {
			case 127:
				out[i] = 1
			case 64:
				out[i] = 0
			case 0:
				out[i] = -1
			default:
				out[i] = module.Value((float64(e.Data2) - 64) / 64)
			}
		}
	}
}

type ctrlCC struct {
	*Controller
	number int
}

func (reader *ctrlCC) Read(out module.Frame) {
	reader.read(out)
	for i := range out {
		e := reader.events[i]
		if e.Status == 176 && int(e.Data1) == reader.number {
			switch e.Data2 {
			case 0:
				out[i] = 0
			default:
				out[i] = module.Value(float64(e.Data2) / 127)
			}
		}
	}
}
