package midi

import (
	"fmt"

	"buddin.us/eolian/dsp"
	"buddin.us/eolian/module"
	"buddin.us/musictheory"
	"github.com/mitchellh/mapstructure"
	"github.com/rakyll/portmidi"
)

var (
	pitches = map[int]dsp.Float64{}
)

func init() {
	p := musictheory.NewPitch(musictheory.C, musictheory.Natural, 0)
	for i := 12; i < 127; i++ {
		pitches[i] = dsp.Frequency(p.Freq()).Value()
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

type controller struct {
	module.IO
	stream           *portmidi.Stream
	streamEvents     <-chan portmidi.Event
	stopStreamEvents chan struct{}

	deviceID  portmidi.DeviceID
	frameRate int
	events    []portmidi.Event
	reads     int
}

func newController(config controllerConfig) (*controller, error) {
	initMIDI()
	id, err := findDevice(config.Device, dirIn)
	if err != nil {
		return nil, err
	}
	stream, err := portmidi.NewInputStream(id, int64(dsp.FrameSize))
	if err != nil {
		return nil, err
	}
	fmt.Printf("MIDI: %s (in)\n", portmidi.Info(id).Name)

	stop := make(chan struct{})

	m := &controller{
		stream:           stream,
		streamEvents:     streamEvents(stream, stop),
		stopStreamEvents: stop,
		deviceID:         id,
		frameRate:        config.FrameRate,
		events:           make([]portmidi.Event, dsp.FrameSize),
	}
	outs := []*module.Out{}

	outs = append(outs, polyphonicOutputs(m, config.Polyphony)...)

	outs = append(outs,
		&module.Out{Name: "sync", Provider: dsp.Provide(&ctrlSync{controller: m})},
		&module.Out{Name: "reset", Provider: dsp.Provide(&ctrlReset{controller: m})},
		&module.Out{Name: "pitchBend", Provider: dsp.Provide(&ctrlPitchBend{controller: m})})

	for _, c := range config.CCChannels {
		for n := 0; n < 128; n++ {
			func(c, n int) {
				outs = append(outs, &module.Out{
					Name: fmt.Sprintf("cc/%d/%d", c, n),
					Provider: dsp.Provide(&ctrlCC{
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

func (c *controller) read(out dsp.Frame) {
	if c.reads == 0 && c.stream != nil {
		for i := range out {
			select {
			case c.events[i] = <-c.streamEvents:
			default:
				c.events[i] = portmidi.Event{}
			}
		}
	}
	if outs := c.OutputsActive(true); outs > 0 {
		c.reads = (c.reads + 1) % outs
	}
}

func (c *controller) Output(name string) (*module.Out, error) {
	if c.stream == nil {
		var err error
		c.stream, err = portmidi.NewInputStream(c.deviceID, int64(dsp.FrameSize))
		if err != nil {
			return nil, err
		}
	}
	return c.IO.Output(name)
}

func (c *controller) Close() error {
	if c.stream != nil {
		if err := c.stream.Close(); err != nil {
			return err
		}
		c.stream = nil
		go func() { c.stopStreamEvents <- struct{}{} }()
	}
	return nil
}

type ctrlGate struct {
	controller    *controller
	channelOffset int
	stateFunc     gateStateFunc
	state         *gateState
}

func (reader *ctrlGate) Process(out dsp.Frame) {
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
	value                dsp.Float64
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
	lastVelocity  dsp.Float64
}

func (reader *ctrlVelocity) Process(out dsp.Frame) {
	reader.controller.read(out)
	for i := range out {
		if int(reader.controller.events[i].Status) == 144+reader.channelOffset {
			data2 := int(reader.controller.events[i].Data2)
			if data2 == 0 {
				out[i] = reader.lastVelocity
			} else {
				out[i] = dsp.Float64(data2) / 127
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

func (reader *ctrlSync) Process(out dsp.Frame) {
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
	pitch         dsp.Float64
}

func (reader *ctrlPitch) Process(out dsp.Frame) {
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
	controller *controller
}

func (reader ctrlReset) Process(out dsp.Frame) {
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
	controller *controller
}

func (reader *ctrlPitchBend) Process(out dsp.Frame) {
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
				out[i] = dsp.Float64((float64(e.Data2) - 64) / 64)
			}
		}
	}
}

type ctrlCC struct {
	controller     *controller
	status, number int
	value          dsp.Float64
}

func (reader *ctrlCC) Process(out dsp.Frame) {
	reader.controller.read(out)
	for i := range out {
		e := reader.controller.events[i]
		if int(e.Status) == reader.status && int(e.Data1) == reader.number {
			reader.value = dsp.Float64(float64(e.Data2) / 127)
		}
		out[i] = reader.value
	}
}

func polyphonicOutputs(m *controller, count int) []*module.Out {
	outs := []*module.Out{}

	if count == 0 {
		outs = append(outs, &module.Out{
			Name: "gate",
			Provider: dsp.Provide(&ctrlGate{
				controller: m,
				stateFunc:  gateUp,
				state:      &gateState{which: -1},
			}),
		},
			&module.Out{Name: "pitch", Provider: dsp.Provide(&ctrlPitch{controller: m})},
			&module.Out{Name: "velocity", Provider: dsp.Provide(&ctrlVelocity{controller: m})})
	} else {
		for i := 0; i < count; i++ {
			outs = append(outs, &module.Out{
				Name: fmt.Sprintf("%d/gate", i),
				Provider: dsp.Provide(&ctrlGate{
					controller:    m,
					stateFunc:     gateUp,
					state:         &gateState{which: -1},
					channelOffset: i,
				}),
			},
				&module.Out{Name: fmt.Sprintf("%d/pitch", i), Provider: dsp.Provide(&ctrlPitch{controller: m, channelOffset: i})},
				&module.Out{Name: fmt.Sprintf("%d/velocity", i), Provider: dsp.Provide(&ctrlVelocity{controller: m, channelOffset: i})})
		}
	}

	return outs
}
