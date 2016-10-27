package midi

import (
	"github.com/brettbuddin/eolian/module"
	"github.com/mitchellh/mapstructure"
	"github.com/rakyll/portmidi"
)

func init() {
	module.Register("MIDIClock", func(c module.Config) (module.Patcher, error) {
		var config ClockConfig
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		return NewClock(config.Device)
	})
}

type ClockConfig struct {
	Device int
}

type Clock struct {
	*portmidi.Stream
	module.IO

	frameRate int
	reads     int
	count     int
	events    []portmidi.Event
}

func NewClock(deviceID int) (*Clock, error) {
	initMIDI()
	stream, err := portmidi.NewInputStream(portmidi.DeviceID(deviceID), int64(module.FrameSize))
	if err != nil {
		return nil, err
	}
	m := &Clock{
		Stream:    stream,
		frameRate: 24,
		events:    make([]portmidi.Event, module.FrameSize),
	}

	outs := []*module.Out{
		{
			Name:     "pulse",
			Provider: module.ReaderProviderFunc(func() module.Reader { return &clockPulse{m} }),
		},
		{
			Name:     "reset",
			Provider: module.ReaderProviderFunc(func() module.Reader { return &clockReset{m} }),
		},
	}

	err = m.Expose(nil, outs)
	return m, err
}

func (c *Clock) read(out module.Frame) {
	if c.reads == 0 {
		for i := range out {
			events, err := c.Stream.Read(1)
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

func (s *Clock) Close() error {
	if s.Stream != nil {
		if err := s.Stream.Close(); err != nil {
			return err
		}
		s.Stream = nil
	}
	return nil
}

type clockPulse struct {
	*Clock
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
	*Clock
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
