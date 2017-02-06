package module

import (
	"fmt"
	"math"

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
	parsed, err := govaluate.NewEvaluableExpressionWithFunctions(exp, expFns)
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

var expFns = map[string]govaluate.ExpressionFunction{
	"abs": func(args ...interface{}) (interface{}, error) {
		v := args[0].(float64)
		return math.Abs(v), nil
	},
	"min": func(args ...interface{}) (interface{}, error) {
		v1 := args[0].(float64)
		v2 := args[1].(float64)
		return math.Min(v1, v2), nil
	},
	"max": func(args ...interface{}) (interface{}, error) {
		v1 := args[0].(float64)
		v2 := args[1].(float64)
		return math.Max(v1, v2), nil
	},
	"sin": func(args ...interface{}) (interface{}, error) {
		v := args[0].(float64)
		return math.Sin(v), nil
	},
	"cos": func(args ...interface{}) (interface{}, error) {
		v := args[0].(float64)
		return math.Cos(v), nil
	},
	"tan": func(args ...interface{}) (interface{}, error) {
		v := args[0].(float64)
		return math.Tan(v), nil
	},
	"atanh": func(args ...interface{}) (interface{}, error) {
		v := args[0].(float64)
		return math.Atanh(v), nil
	},
	"tanh": func(args ...interface{}) (interface{}, error) {
		v := args[0].(float64)
		return math.Tanh(v), nil
	},
	"asin": func(args ...interface{}) (interface{}, error) {
		v := args[0].(float64)
		return math.Asin(v), nil
	},
	"acos": func(args ...interface{}) (interface{}, error) {
		v := args[0].(float64)
		return math.Acos(v), nil
	},
	"asinh": func(args ...interface{}) (interface{}, error) {
		v := args[0].(float64)
		return math.Asinh(v), nil
	},
	"acosh": func(args ...interface{}) (interface{}, error) {
		v := args[0].(float64)
		return math.Acosh(v), nil
	},
	"pow": func(args ...interface{}) (interface{}, error) {
		x := args[0].(float64)
		y := args[1].(float64)
		return math.Pow(x, y), nil
	},
	"exp": func(args ...interface{}) (interface{}, error) {
		x := args[0].(float64)
		return math.Exp(x), nil
	},
	"exp2": func(args ...interface{}) (interface{}, error) {
		x := args[0].(float64)
		return math.Exp2(x), nil
	},
	"log": func(args ...interface{}) (interface{}, error) {
		x := args[0].(float64)
		return math.Log(x), nil
	},
	"log10": func(args ...interface{}) (interface{}, error) {
		x := args[0].(float64)
		return math.Log10(x), nil
	},
	"log2": func(args ...interface{}) (interface{}, error) {
		x := args[0].(float64)
		return math.Log2(x), nil
	},
	"sqrt": func(args ...interface{}) (interface{}, error) {
		x := args[0].(float64)
		return math.Sqrt(x), nil
	},
	"floor": func(args ...interface{}) (interface{}, error) {
		x := args[0].(float64)
		return math.Floor(x), nil
	},
	"ceil": func(args ...interface{}) (interface{}, error) {
		x := args[0].(float64)
		return math.Ceil(x), nil
	},
	"clamp": func(args ...interface{}) (interface{}, error) {
		x := args[0].(float64)
		min := args[1].(float64)
		max := args[2].(float64)

		if x > max {
			x = max
		} else if x < min {
			x = min
		}
		return x, nil
	},
}
