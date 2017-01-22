package lua

import lua "github.com/yuin/gopher-lua"

func preloadSynthInterpolate(state *lua.LState) int {
	if err := state.DoString(luaInterpolate); err != nil {
		state.RaiseError(err.Error())
	}
	return 1
}

var luaInterpolate = `
return function(module, ranges)
    local synth = require('eolian.synth')
    local proxy = require('eolian.synth.proxy')

    local proxied = {}

    if type(module) == 'function' then
        module = module()
    end

    for k, range in pairs(ranges) do
        proxied[k] = synth.Interpolate(range)
        module:set { [k] = proxied[k]:output() }
    end

    return {
        set = function(_, inputs)
            for k, v in pairs(inputs) do
                if type(v) == 'table' then
                    local prefix = k
                    for k,v in pairs(v) do
                        local full = prefix .. '/' .. k
                        if proxied[full] ~= nil then
                            proxied[full]:set { input = v }
                        else
                            module:set { [full] = v }
                        end
                    end
                else
                    if proxied[k] ~= nil then
                        proxied[k]:set { input = v }
                    else
                        module:set { [k] = v }
                    end
                end
            end
        end,
        output = proxy.outputs(module),
    }
end
`
