package engine

import (
	"fmt"

	"github.com/brettbuddin/eolian/module"
	"github.com/gordonklaus/portaudio"
)

type Engine struct {
	module.IO
	in *module.In

	device *portaudio.DeviceInfo
	errors chan error
	stop   chan error
}

func New(deviceIndex int) (*Engine, error) {
	portaudio.Initialize()

	devices, err := portaudio.Devices()
	if err != nil {
		return nil, err
	}

	if deviceIndex >= len(devices) {
		return nil, fmt.Errorf("device index out of range")
	}

	fmt.Println("Output:", devices[deviceIndex].Name)

	direct, err := module.NewDirect()
	if err != nil {
		return nil, err
	}
	m := &Engine{
		in:     &module.In{Name: "input", Source: module.NewBuffer(direct.In)},
		errors: make(chan error),
		stop:   make(chan error),
		device: devices[deviceIndex],
	}
	err = m.Expose([]*module.In{m.in}, nil)
	return m, err
}

func (e *Engine) Errors() chan error {
	return e.errors
}

func (e *Engine) params() portaudio.StreamParameters {
	params := portaudio.LowLatencyParameters(nil, e.device)
	params.Output.Channels = 1
	params.SampleRate = module.SampleRate
	params.FramesPerBuffer = module.FrameSize
	return params
}

func (e *Engine) Run() {
	stream, err := portaudio.OpenStream(e.params(), e.portAudioCallback)

	if err != nil {
		e.errors <- err
		return
	}

	if err = stream.Start(); err != nil {
		e.errors <- err
	}
	<-e.stop

	err = stream.Stop()
	if err == nil {
		err = stream.Close()
	}
	e.stop <- err
}

func (e *Engine) Stop() error {
	defer portaudio.Terminate()
	e.stop <- nil
	err := <-e.stop
	close(e.errors)
	close(e.stop)
	return err
}

func (e *Engine) portAudioCallback(_, out []float32) {
	frame := e.in.ReadFrame()
	for i := range out {
		out[i] = float32(frame[i])
	}
}
