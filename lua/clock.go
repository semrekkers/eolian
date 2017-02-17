package lua

import lua "github.com/yuin/gopher-lua"

func preloadSynthClock(state *lua.LState) int {
	if err := state.DoString(luaClock); err != nil {
		state.RaiseError(err.Error())
	}
	return 1
}

var luaClock = `
return function(size)
    local synth = require('eolian.synth')
    local proxy = require('eolian.synth.proxy')
    local osc   = synth.Oscillator { algorithm = 'simple' }
    local mult  = synth.Multiple { size = size }

    set(mult, 'input', out(osc, 'pulse'))

    return {
        id = function()
            return string.format("Clock[%s, %s]", osc.id(), mult.id())
        end,
		members = function()
			return { osc.id(), mult.id() }
		end,
        inputs  = osc.inputs,
        outputs = mult.outputs,
        set     = proxy.inputs(osc),
        out     = proxy.outputs(mult),
    }
end
`
