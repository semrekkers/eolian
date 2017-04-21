return function(source, output)
    local lastModule, lastOut = source, output
    local o = {}

    o.to = function(...)
        local module
        local input = 'input'
        local output = 'output'

        if #arg == 1 then
            module = arg[1]
        elseif #arg == 2 then
            if type(arg[1]) == 'string' then
                input = arg[1]
                module = arg[2]
            else
                module = arg[1]
                output = arg[2]
            end
        elseif #arg == 3 then
            input  = arg[1]
            module = arg[2]
            output = arg[3]
        elseif #arg > 3 then
            error('too many arguments')
        end

        if module == nil then
            error('nil value provided as module')
        end

        if lastModule ~= nil then
            module:set { [input] = lastModule:out(lastOut) }
        end
        lastModule = module
        lastOut    = output
        return o
    end

    o.sink = function(output)
        return lastModule:out(lastOut)
    end

    return o
end
