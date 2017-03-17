return function(size)
    local synth = require('eolian.synth')
    local proxy = require('eolian.synth.proxy')
    local clock = synth.Clock()
    local mult  = synth.Multiple { size = size }

    mult:set('input', clock:out())

    return {
        id = function()
            return string.format("Clock[%s, %s]", clock.id(), mult.id())
        end,
		members = function()
			return { clock.id(), mult.id() }
		end,
        inputs  = clock.inputs,
        outputs = mult.outputs,
        set     = proxy.inputs(clock),
        out     = proxy.outputs(mult),
    }
end
