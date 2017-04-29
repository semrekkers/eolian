-- More information about the structure of this file: https://github.com/brettbuddin/eolian/wiki/Rack-Files
local synth = require('eolian.synth')

return function(env)
    local function build()
        return {
            mix = synth.PanMix(),
        }
    end

    local function patch(m)
        m.mix:set {
            -- mixer inputs...
        }
        return m.mix:out('a', 'b')
    end

    return build, patch
end
