-- More information about the structure of this file: https://github.com/brettbuddin/eolian/wiki/Rack-Files

return function(env)
    local synth = require('eolian.synth')

    local function build()
        return {
            mono = synth.Multiple { size = 2 },
        }
    end

    local function patch(modules)
        return modules.mono:out(0), modules.mono:out(1)
    end

    return build, patch
end
