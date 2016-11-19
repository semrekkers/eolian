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
		return NewFileSource(config.Path)
	})
}

type FileSource struct {
	IO
	path   string
	values []Value
	idx    int
}

func NewFileSource(path string) (*FileSource, error) {
	m := &FileSource{
		path:   path,
		values: []Value{},
	}

	if err := m.loadData(); err != nil {
		return nil, err
	}

	err := m.Expose(nil, []*Out{{Name: "output", Provider: Provide(m)}})
	return m, err
}

func (reader *FileSource) Read(out Frame) {
	for i := range out {
		out[i] = reader.values[reader.idx]
		reader.idx = (reader.idx + 1) % len(reader.values)
	}
}

func (s *FileSource) loadData() error {
	file, err := os.Open(s.path)
	if err != nil {
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		if f, err := strconv.ParseFloat(scanner.Text(), 64); err == nil {
			s.values = append(s.values, Value(f))
		}
	}
	return nil
}
