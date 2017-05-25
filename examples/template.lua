-- More information about the structure of this file: https://github.com/brettbuddin/eolian/wiki/Rack-Files
return function()
    local synth              = require('eolian.synth')
    local theory             = require('eolian.theory')
    local value              = require('eolian.value')
    local hz, ms, pitch, bpm = value.hz, 
                               value.ms, 
                               value.pitch, 
                               value.bpm

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
