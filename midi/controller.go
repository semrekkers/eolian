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
		var config controllerConfig
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}

		if len(config.CCChannels) == 0 {
			config.CCChannels = []int{1}
		}

		if config.FrameRate == 0 {
			config.FrameRate = 24
		}

		if config.Polyphony < 1 {
			config.Polyphony = 1
		}

		return newController(config)
	})
}

type controllerConfig struct {
	Device               string
	Polyphony, FrameRate int
	CCChannels           []int `mapstructure:"ccChannels"`
}

type cc struct {
	Channel, Number int
}

type controller struct {
	module.IO
	*portmidi.Stream

	frameRate int
	events    []portmidi.Event
	reads     int

	lastClock module.Value
	clockTick int
}

func newController(config controllerConfig) (*controller, error) {
	initMIDI()

	if config.Device == "" {
		return nil, fmt.Errorf("no device name specified")
	}

	var deviceID portmidi.DeviceID = -1
	for i := 0; i < portmidi.CountDevices(); i++ {
		id := portmidi.DeviceID(i)
		info := portmidi.Info(id)
		if info.Name == config.Device && info.IsInputAvailable {
			deviceID = id
		}
	}

	if deviceID == -1 {
		return nil, fmt.Errorf(`unknown device "%s"`, config.Device)
	}

	stream, err := portmidi.NewInputStream(deviceID, int64(module.FrameSize))
	if err != nil {
		return nil, err
	}
	fmt.Printf("MIDI: %s\n", portmidi.Info(deviceID).Name)

	m := &controller{
		Stream:    stream,
		frameRate: config.FrameRate,
		events:    make([]portmidi.Event, module.FrameSize),
	}
	outs := []*module.Out{}

	outs = append(outs, polyphonicOutputs(m, config.Polyphony)...)

	outs = append(outs,
		&module.Out{Name: "sync", Provider: module.Provide(&ctrlSync{controller: m})},
		&module.Out{Name: "reset", Provider: module.Provide(&ctrlReset{controller: m})},
		&module.Out{Name: "pitchBend", Provider: module.Provide(&ctrlPitchBend{controller: m})})

	for _, c := range config.CCChannels {
		for n := 0; n < 128; n++ {
			func(c, n int) {
				outs = append(outs, &module.Out{
					Name: fmt.Sprintf("cc/%d/%d", c, n),
					Provider: module.Provide(&ctrlCC{
						controller: m,
						status:     176 + (c - 1),
						number:     n,
					}),
				})
			}(c, n)
		}
	}

	return m, m.Expose("MIDIController", nil, outs)
}

func (c *controller) read(out module.Frame) {
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

func (c *controller) Close() error {
	if c.Stream != nil {
		if err := c.Stream.Close(); err != nil {
			return err
		}
		c.Stream = nil
	}
	return nil
}

type ctrlGate struct {
	controller    *controller
	channelOffset int
	stateFunc     gateStateFunc
	state         *gateState
}

func (reader *ctrlGate) Read(out module.Frame) {
	reader.controller.read(out)
	for i := range out {
		reader.state.event = reader.controller.events[i]
		reader.state.channelOffset = reader.channelOffset
		reader.stateFunc = reader.stateFunc(reader.state)
		out[i] = reader.state.value
	}
}

type gateState struct {
	event                portmidi.Event
	which, channelOffset int
	value                module.Value
}

func gateRolling(s *gateState) gateStateFunc {
	s.value = -1
	return gateDown
}

func gateDown(s *gateState) gateStateFunc {
	s.value = 1
	which := int(s.event.Data1)

	switch int(s.event.Status) {
	case 144 + s.channelOffset:
		if s.event.Data2 > 0 {
			if which != s.which {
				s.which = which
				return gateRolling
			}
			s.which = -1
			return gateUp
		}
		if which == s.which {
			s.which = -1
			return gateUp
		}
	case 128:
		if which == s.which {
			s.which = -1
			return gateUp
		}
	}
	return gateDown
}

func gateUp(s *gateState) gateStateFunc {
	s.value = -1
	if int(s.event.Status) == 144+s.channelOffset && s.event.Data2 > 0 {
		s.which = int(s.event.Data1)
		return gateDown
	}
	return gateUp
}

type gateStateFunc func(*gateState) gateStateFunc

type ctrlVelocity struct {
	controller    *controller
	channelOffset int
	lastVelocity  module.Value
}

func (reader *ctrlVelocity) Read(out module.Frame) {
	reader.controller.read(out)
	for i := range out {
		if int(reader.controller.events[i].Status) == 144+reader.channelOffset {
			data2 := int(reader.controller.events[i].Data2)
			if data2 == 0 {
				out[i] = reader.lastVelocity
			} else {
				out[i] = module.Value(data2) / 127
				reader.lastVelocity = out[i]
			}
		} else {
			out[i] = reader.lastVelocity
		}
	}
}

type ctrlSync struct {
	controller *controller
	tick       int
}

func (reader *ctrlSync) Read(out module.Frame) {
	reader.controller.read(out)
	for i := range out {
		if reader.controller.events[i].Status == 248 || reader.controller.events[i].Status == 250 {
			reader.tick++
		}

		if reader.tick%reader.controller.frameRate == 0 {
			out[i] = 1
			reader.tick = 0
		} else {
			out[i] = -1
		}
	}
}

type ctrlPitch struct {
	controller    *controller
	channelOffset int
	pitch         module.Value
}

func (reader *ctrlPitch) Read(out module.Frame) {
	reader.controller.read(out)
	for i := range out {
		if int(reader.controller.events[i].Status) == 144+reader.channelOffset {
			data1 := int(reader.controller.events[i].Data1)
			data2 := int(reader.controller.events[i].Data2)
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

type ctrlReset struct {
	controller    *controller
	channelOffset int
}

func (reader ctrlReset) Read(out module.Frame) {
	reader.controller.read(out)
	for i := range out {
		if reader.controller.events[i].Status == 250 {
			out[i] = 1
		} else {
			out[i] = -1
		}
	}
}

type ctrlPitchBend struct {
	controller    *controller
	channelOffset int
}

func (reader *ctrlPitchBend) Read(out module.Frame) {
	reader.controller.read(out)
	for i := range out {
		e := reader.controller.events[i]
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
	controller     *controller
	status, number int
	value          module.Value
}

func (reader *ctrlCC) Read(out module.Frame) {
	reader.controller.read(out)
	for i := range out {
		e := reader.controller.events[i]
		if int(e.Status) == reader.status && int(e.Data1) == reader.number {
			reader.value = module.Value(float64(e.Data2) / 127)
		}
		out[i] = reader.value
	}
}

func polyphonicOutputs(m *controller, count int) []*module.Out {
	outs := []*module.Out{}

	if count == 0 {
		outs = append(outs, &module.Out{
			Name: "gate",
			Provider: module.Provide(&ctrlGate{
				controller: m,
				stateFunc:  gateUp,
				state:      &gateState{which: -1},
			}),
		},
			&module.Out{Name: "pitch", Provider: module.Provide(&ctrlPitch{controller: m})},
			&module.Out{Name: "velocity", Provider: module.Provide(&ctrlVelocity{controller: m})})
	} else {
		for i := 0; i < count; i++ {
			outs = append(outs, &module.Out{
				Name: fmt.Sprintf("%d/gate", i),
				Provider: module.Provide(&ctrlGate{
					controller:    m,
					stateFunc:     gateUp,
					state:         &gateState{which: -1},
					channelOffset: i,
				}),
			},
				&module.Out{Name: fmt.Sprintf("%d/pitch", i), Provider: module.Provide(&ctrlPitch{controller: m, channelOffset: i})},
				&module.Out{Name: fmt.Sprintf("%d/velocity", i), Provider: module.Provide(&ctrlVelocity{controller: m, channelOffset: i})})
		}
	}

	return outs
}
