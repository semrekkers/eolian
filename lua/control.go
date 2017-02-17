package lua

import lua "github.com/yuin/gopher-lua"

func preloadSynthControl(state *lua.LState) int {
	if err := state.DoString(luaControl); err != nil {
		state.RaiseError(err.Error())
	}
	return 1
}

var luaControl = `
return function(m, options, defaultInput)
    options = options or {}
    defaultInput = defaultInput or 'control'

    if type(m) == 'function' then
        m = m()
    end

    local synth  = require('eolian.synth')
    local proxy  = require('eolian.synth.proxy')
    local eolianString = require('eolian.string')

    local controls = {}
    for name,_ in pairs(m.inputs()) do
        if options[name] ~= nil then
            controls[name] = synth.Control(options[name])
            set(m, name, out(controls[name]))
        end
    end

    return {
        id = function()
			return string.format("Controlled[%s]", m.id())
		end,
		members = function()
			local m = { m.id() }
			for _,c in ipairs(controls) do
				table.insert(m, c)
			end
			return m
		end,
        inputs  = m.inputs,
        outputs = m.outputs,
        set = function(_, inputs)
            for k,v in pairs(inputs) do
                local segs = eolianString.split(k, '/')
                if #segs == 2 and controls[segs[1]] ~= nil then
                    set(controls[segs[1]], segs[2], v)
                elseif controls[k] ~= nil then
                    set(controls[k], defaultInput, v)
                else
                    set(m, k, v)
                end
            end
        end,
        out = proxy.outputs(m)
    }
end
`
