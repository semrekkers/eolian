package module

import (
	"bufio"
	"os"
	"strconv"

	"buddin.us/eolian/dsp"

	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("FileSource", func(c Config) (Patcher, error) {
		var config struct {
			Path string
		}
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		return newFileSource(config.Path)
	})
}

type fileSource struct {
	IO
	values []dsp.Float64
	idx    int
}

func newFileSource(path string) (*fileSource, error) {
	m := &fileSource{
		values: []dsp.Float64{},
	}

	if err := m.loadData(path); err != nil {
		return nil, err
	}

	err := m.Expose("FileSource", nil, []*Out{{Name: "output", Provider: dsp.Provide(m)}})
	return m, err
}

func (f *fileSource) Process(out dsp.Frame) {
	for i := range out {
		out[i] = f.values[f.idx]
		f.idx = (f.idx + 1) % len(f.values)
	}
}

func (f *fileSource) loadData(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		if v, err := strconv.ParseFloat(scanner.Text(), 64); err == nil {
			f.values = append(f.values, dsp.Float64(v))
		}
	}
	return nil
}
