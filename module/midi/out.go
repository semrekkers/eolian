package midi

import (
	"fmt"
	"regexp"
	"strconv"

	"buddin.us/eolian/dsp"
	"buddin.us/eolian/module"
	"github.com/mitchellh/mapstructure"
	"github.com/rakyll/portmidi"
)

var ccInputPattern = regexp.MustCompile("([0-9]+)/cc/([0-9]+)")

func init() {
	module.Register("MIDIOut", func(c module.Config) (module.Patcher, error) {
		var config struct{ Device string }
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		return newOut(config.Device)
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
	last            dsp.Float64
}

type ccSignal struct {
	channel, number int
	value           dsp.Float64
}

func newOut(device string) (*out, error) {
	initMIDI()
	id, err := findDevice(device, dirOut)
	if err != nil {
		return nil, err
	}
	stream, err := portmidi.NewOutputStream(id, int64(dsp.FrameSize), 0)
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
		in:      module.NewIn("input", dsp.Float64(0)),
		stream:  stream,
		signals: signals,
	}
	return m, m.Expose(
		"MIDIOut",
		[]*module.In{m.in},
		[]*module.Out{
			&module.Out{Name: "output", Provider: dsp.Provide(m)},
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
			Source: dsp.NewBuffer(dsp.Float64(0)),
		}
		if err := o.IO.AddInput(in); err != nil {
			return err
		}
		o.ccs = append(o.ccs, &midiCC{
			In:      in,
			channel: channel,
			number:  number,
			last:    -1,
		})
		return o.IO.Patch(name, t)
	}
	return nil
}

func (o *out) Process(out dsp.Frame) {
	o.in.Process(out)
	for i, cc := range o.ccs {
		frame := cc.ProcessFrame()
		for j := range out {
			if o.ccs[i].last != frame[j] {
				o.signals <- ccSignal{cc.channel, cc.number, frame[j]}
			}
			o.ccs[i].last = frame[j]
		}
	}
}
