-- More information about the structure of this file: https://github.com/brettbuddin/eolian/wiki/Rack-Files

return function(env)
    local function build()
        return {}
    end

    local function patch(modules)
        return 0
    end

    return build, patch
end
