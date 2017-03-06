package midi

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/brettbuddin/eolian/module"
	"github.com/mitchellh/mapstructure"
	"github.com/rakyll/portmidi"
)

var ccInputPattern = regexp.MustCompile("cc/([0-9]+)/([0-9]+)")

func init() {
	module.Register("MIDIOut", func(c module.Config) (module.Patcher, error) {
		var config controllerConfig
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		return newOut(config)
	})
}

type out struct {
	module.IO
	in      *module.In
	ccs     []*midiCC
	stream  *portmidi.Stream
	signals chan ccSignal
}

type midiCC struct {
	*module.In
	channel, number int
}

type ccSignal struct {
	channel, number int
	value           module.Value
}

func newOut(config controllerConfig) (*out, error) {
	initMIDI()
	id, err := findDevice(config.Device, dirOut)
	if err != nil {
		return nil, err
	}
	stream, err := portmidi.NewOutputStream(id, int64(module.FrameSize), 0)
	if err != nil {
		return nil, err
	}
	fmt.Printf("MIDI: %s (out)\n", portmidi.Info(id).Name)

	signals := make(chan ccSignal)
	go func() {
		for cc := range signals {
			stream.WriteShort(int64(176+cc.channel-1), int64(cc.number), int64(cc.value))
		}
	}()

	m := &out{
		in:      &module.In{Name: "input", Source: module.Value(0)},
		stream:  stream,
		signals: signals,
	}
	return m, m.Expose(
		"MIDIOut",
		[]*module.In{m.in},
		[]*module.Out{
			&module.Out{Name: "output", Provider: module.Provide(m)},
		})
}

func (o *out) Close() error {
	if o.stream != nil {
		if err := o.stream.Close(); err != nil {
			return err
		}
		o.stream = nil
		close(o.signals)
	}
	return nil
}

func (o *out) Patch(name string, t interface{}) error {
	if err := o.IO.Patch(name, t); err != nil {
		matches := ccInputPattern.FindAllStringSubmatch(name, -1)
		if len(matches) == 0 {
			return fmt.Errorf("invalid midi CC input name: %s", name)
		}

		channel, err := strconv.Atoi(matches[0][1])
		if err != nil {
			return err
		}
		number, err := strconv.Atoi(matches[0][2])
		if err != nil {
			return err
		}

		in := &module.In{
			Name:   name,
			Source: module.NewBuffer(module.Value(0)),
		}
		if err := o.IO.AddInput(in); err != nil {
			return err
		}
		o.ccs = append(o.ccs, &midiCC{
			In:      in,
			channel: channel,
			number:  number,
		})
		return o.IO.Patch(name, t)
	}
	return nil
}

func (o *out) Read(out module.Frame) {
	o.in.Read(out)
	for _, cc := range o.ccs {
		frame := cc.ReadFrame()
		var i int
		for j := range out {
			if i == 0 {
				o.signals <- ccSignal{cc.channel, cc.number, frame[j]}
			}
			i = (i + 1) % 100
		}
	}
}
