package midi

import (
	"buddin.us/eolian/dsp"
	"buddin.us/eolian/module"
	"github.com/mitchellh/mapstructure"
	"github.com/rakyll/portmidi"
)

func init() {
	module.Register("MIDIClock", func(c module.Config) (module.Patcher, error) {
		var config struct {
			Device    string
			FrameRate int
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.FrameRate == 0 {
			config.FrameRate = 24
		}
		return newClock(config.Device, config.FrameRate)
	})
}

type clock struct {
	module.IO
	stream           *portmidi.Stream
	streamEvents     <-chan portmidi.Event
	stopStreamEvents chan struct{}

	deviceID                portmidi.DeviceID
	frameRate, reads, count int
	events                  []portmidi.Event
}

func newClock(device string, frameRate int) (*clock, error) {
	initMIDI()
	id, err := findDevice(device, dirIn)
	if err != nil {
		return nil, err
	}
	stream, err := portmidi.NewInputStream(id, int64(dsp.FrameSize))
	if err != nil {
		return nil, err
	}

	stop := make(chan struct{})

	m := &clock{
		stream:           stream,
		streamEvents:     streamEvents(stream, stop),
		stopStreamEvents: stop,
		deviceID:         id,
		frameRate:        frameRate,
		events:           make([]portmidi.Event, dsp.FrameSize),
	}
	outs := []*module.Out{
		{Name: "pulse", Provider: dsp.Provide(&clockPulse{m})},
		{Name: "reset", Provider: dsp.Provide(&clockReset{m})},
	}
	return m, m.Expose("MIDIClock", nil, outs)
}

func (c *clock) read(out dsp.Frame) {
	if c.reads == 0 && c.stream != nil {
		for i := range out {
			select {
			case c.events[i] = <-c.streamEvents:
				if e := c.events[i]; e.Status == 248 || e.Status == 250 {
					c.count++
				}
			default:
				c.events[i] = portmidi.Event{}
			}
		}
	}
	if outs := c.OutputsActive(true); outs > 0 {
		c.reads = (c.reads + 1) % outs
	}
}

func (c *clock) Output(name string) (*module.Out, error) {
	if c.stream == nil {
		var err error
		c.stream, err = portmidi.NewInputStream(c.deviceID, int64(dsp.FrameSize))
		if err != nil {
			return nil, err
		}
	}
	return c.IO.Output(name)
}

func (c *clock) Close() error {
	if c.stream != nil {
		if err := c.stream.Close(); err != nil {
			return err
		}
		c.stream = nil
		go func() { c.stopStreamEvents <- struct{}{} }()
	}
	return nil
}

type clockPulse struct {
	*clock
}

func (reader *clockPulse) Process(out dsp.Frame) {
	reader.read(out)
	for i := range out {
		if reader.count%reader.frameRate == 0 {
			out[i] = 1
			reader.count = 0
		} else {
			out[i] = -1
		}
	}
}

type clockReset struct {
	*clock
}

func (reader *clockReset) Process(out dsp.Frame) {
	reader.read(out)
	for i := range out {
		if reader.events[i].Status == 250 {
			out[i] = 1
			reader.count = 0
		} else {
			out[i] = -1
		}
	}
}
