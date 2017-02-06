package module

import (
	"fmt"

	"github.com/Knetic/govaluate"
	"github.com/mitchellh/mapstructure"
)

func init() {
	Register("MathExp", func(c Config) (Patcher, error) {
		var config struct{ Expression string }
		if err := mapstructure.Decode(c, &config); err != nil {
			return nil, err
		}
		if config.Expression == "" {
			return nil, fmt.Errorf(`"expression" is required when initializing MathExp`)
		}
		return newExpression(config.Expression)
	})
}

type expression struct {
	IO
	in, level *In
	vars      []*In
	exp       *govaluate.EvaluableExpression
	params    map[string]interface{}
}

func newExpression(exp string) (*expression, error) {
	parsed, err := govaluate.NewEvaluableExpression(exp)
	if err != nil {
		return nil, err
	}

	m := &expression{
		in:     &In{Name: "input", Source: zero},
		vars:   []*In{},
		exp:    parsed,
		params: map[string]interface{}{},
	}

	inputs := []*In{m.in}
	for _, v := range parsed.Vars() {
		in := &In{Name: v, Source: NewBuffer(zero)}
		inputs = append(inputs, in)
		m.vars = append(m.vars, in)
		m.params[in.Name] = 0
	}

	return m, m.Expose(
		"Expression",
		inputs,
		[]*Out{{Name: "output", Provider: Provide(m)}},
	)
}

func (e *expression) Read(out Frame) {
	e.in.Read(out)
	for _, v := range e.vars {
		v.ReadFrame()
	}
	for i := range out {
		for _, v := range e.vars {
			e.params[v.Name] = float64(v.LastFrame()[i])
		}
		r, err := e.exp.Evaluate(e.params)
		if err != nil {
			fmt.Println(err)
		}
		switch v := r.(type) {
		case float64:
			out[i] = Value(v)
		case int:
			out[i] = Value(v)
		case int64:
			out[i] = Value(v)
		case bool:
			if v {
				out[i] = 1
			} else {
				out[i] = 0
			}
		}
	}
}
