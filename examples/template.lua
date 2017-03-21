-- More information about the structure of this file: https://github.com/brettbuddin/eolian/wiki/Rack-Files

return function(env)
    local synth = require('eolian.synth')

    local function build()
        return {
            sink = synth.Multiple { size = 2 },
        }
    end

    local function patch(modules)
        return modules.sink:out(0), modules.sink:out(1)
    end

    return build, patch
end
