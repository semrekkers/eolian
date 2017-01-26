package module

import (
	"bufio"
	"os"
	"strconv"

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
	values []Value
	idx    int
}

func newFileSource(path string) (*fileSource, error) {
	m := &fileSource{
		values: []Value{},
	}

	if err := m.loadData(path); err != nil {
		return nil, err
	}

	err := m.Expose("FileSource", nil, []*Out{{Name: "output", Provider: Provide(m)}})
	return m, err
}

func (f *fileSource) Read(out Frame) {
	for i := range out {
		out[i] = f.values[f.idx]
		f.idx = (f.idx + 1) % len(f.values)
	}
}

func (s *fileSource) loadData(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		if f, err := strconv.ParseFloat(scanner.Text(), 64); err == nil {
			s.values = append(s.values, Value(f))
		}
	}
	return nil
}
