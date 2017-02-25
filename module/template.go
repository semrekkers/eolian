package module

// import (
// 	"github.com/mitchellh/mapstructure"
// )

// func init() {
// 	Register("Name", func(c Config) (Patcher, error) {
// 		var config struct{}
// 		if err := mapstructure.Decode(c, &config); err != nil {
// 			return nil, err
// 		}
// 		// Defaults
// 		return newModule()
// 	})
// }

// type module struct {
// 	IO
// 	in *In
// }

// func newModule() (*module, error) {
// 	m := &module{
// 		in: &In{Name: "input", Source: NewBuffer(zero)},
// 	}
// 	return m, m.Expose("Name", []*In{m.in}, []*Out{{Name: "output", Provider: Provide(m)}})
// }

// func (m *module) Read(out Frame) {
// 	in := m.in.ReadFrame()
// 	for i := range out {
// 		out[i] = in[i]
// 	}
// }
