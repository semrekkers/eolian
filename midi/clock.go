package midi

import (
	"github.com/brettbuddin/eolian/module"
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
	stream                  *portmidi.Stream
	deviceID                portmidi.DeviceID
	frameRate, reads, count int
	events                  []portmidi.Event
}

func newClock(device string, frameRate int) (*clock, error) {
	initMIDI()
	id, err := findInputDevice(device)
	if err != nil {
		return nil, err
	}
	stream, err := portmidi.NewInputStream(id, int64(module.FrameSize))
	if err != nil {
		return nil, err
	}
	m := &clock{
		stream:    stream,
		deviceID:  id,
		frameRate: frameRate,
		events:    make([]portmidi.Event, module.FrameSize),
	}
	outs := []*module.Out{
		{Name: "pulse", Provider: module.Provide(&clockPulse{m})},
		{Name: "reset", Provider: module.Provide(&clockReset{m})},
	}
	return m, m.Expose("MIDIClock", nil, outs)
}

func (c *clock) read(out module.Frame) {
	if c.reads == 0 && c.stream != nil {
		for i := range out {
			events, err := c.stream.Read(1)
			if err != nil {
				panic(err)
			}
			if len(events) == 1 && (events[0].Status == 248 || events[0].Status == 250) {
				c.events[i] = events[0]
				c.count++
			}
		}
	}
	if outs := c.OutputsActive(); outs > 0 {
		c.reads = (c.reads + 1) % outs
	}
}

func (c *clock) Output(name string) (*module.Out, error) {
	if c.stream == nil {
		var err error
		c.stream, err = portmidi.NewInputStream(c.deviceID, int64(module.FrameSize))
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
	}
	return nil
}

type clockPulse struct {
	*clock
}

func (reader *clockPulse) Read(out module.Frame) {
	reader.read(out)
	for i := range out {
		if reader.count%int(reader.frameRate) == 0 {
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

func (reader *clockReset) Read(out module.Frame) {
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
