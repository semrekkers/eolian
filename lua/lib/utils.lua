local join      = require('eolian.string').join
local split     = require('eolian.string').split
local sort      = require('eolian.sort')
local tabwriter = require('eolian.tabwriter')
local time      = require('eolian.time')

function with(o, fn)
    return fn(o)
end

function set(m, ...)
    assert(m, 'attempt to set inputs on nil value')
    assert(actsLikeModule(m), 'attempt to set inputs on non-module value')
    return m:set(unpack(arg))
end

function out(m, name)
    assert(m, 'attempt to get output "' .. tostring(name) .. '" from nil value')
    assert(actsLikeModule(m), 'attempt to get output "' .. tostring(name) .. '" from non-module value')
    return m:out(name)
end

function actsLikeModule(m)
    return type(m) == 'table' and
        type(m.inputs) == 'function' and
        type(m.outputs) == 'function' and
        type(m.out) == 'function' and
        type(m.set) == 'function' and
        type(m.id) == 'function'
end

function find(group, name, prefix)
    prefix = prefix or ''

    if string.match(name, 'Engine:0') then
        return 'engine'
    end

    for k, v in pairs(group) do
        if actsLikeModule(v) then
            if v.id() == name then
                if prefix == "" then
                    return k
                end
                return string.format("%s.%s", prefix, k)
            elseif type(v['members']) == 'function' then
                for _,m in ipairs(v.members()) do
                    if m == name then
                        if prefix == "" then
                            return k
                        end
                        return string.format("%s.%s", prefix, k)
                    end
                end
            end
        elseif type(v) == 'table' then
            local result = nil
            if prefix == "" then
                result = find(v, name, k)
            else
                result = find(v, name, string.format("%s.%s", prefix, k))
            end
            if result ~= nil then
                return result
            end
        end
    end
end

function inspect(o, prefix)
    if actsLikeModule(o) then
        local inputNames  = {}
        local outputNames = {}
        local inputs      = o.inputs()
        local outputs     = o.outputs()

        for k,v in pairs(inputs) do
            table.insert(inputNames, k)
        end
        for k,v in pairs(outputs) do
            table.insert(outputNames, k)
        end

        inputNames = sort.strings(inputNames)
        outputNames = sort.strings(outputNames)

        local w = tabwriter.new(5, 0, 1, " ")
        if o['state'] ~= nil then
            local stateNames = {}
            local state = o.state()

            for k,v in pairs(state) do
                table.insert(stateNames, k)
            end
            stateNames = sort.strings(stateNames)

            for _,k in ipairs(stateNames) do
                w.write(string.format("%s\t==\t%s\t\n", k, state[k]))
            end
        end

        for _,k in ipairs(inputNames) do
            local name  = inputs[k]
            local parts = split(name, '/')
            local path  = find(Rack.modules, parts[1])

            if path ~= nil then
                local rest = {}
                for i=2,#parts do
                    table.insert(rest, parts[i])
                end
                name = string.format("%s/%s", path, join(rest, '/'))
            end

            w.write(string.format("%s\t<-\t%s\t\n", k, name))
        end

        for _,k in ipairs(outputNames) do
            local outs  = outputs[k]
            local names = {}

            for _,name in ipairs(outs) do
                local parts = split(name, '/')
                local path  = find(Rack.modules, parts[1])

                if path ~= nil then
                    local rest = {}
                    for i=2,#parts do
                        table.insert(rest, parts[i])
                    end
                    table.insert(names, string.format("%s/%s", path, join(rest, '/')))
                end
            end

            w.write(string.format("%s\t->\t%s\t\n", k, join(names, ', ')))
        end

        local s, count = string.gsub(w.flush(), "\n$", "")
        if count > 0 then print(s) end
        return
    end

    if type(o) == 'table' then
        local w = tabwriter.new(8, 8, 1, "\t", "alignRight")
        for k, v in pairs(o) do
            if k ~= '__namespace' then
                if actsLikeModule(v) then
                    if v.type == nil then
                        v = '(module)'
                    else
                        v = v.type()
                    end
                end
                w.write(string.format("%s\t%s\n", k, v))
            end
        end
        local s, count = string.gsub(w.flush(), "\n$", "")
        if count > 0 then print(s) end
    else
        print(o)
    end
end

local function autoReturn(line)
    local f, err = loadstring('return ' .. line)
    if not f then
        f, err = loadstring(line)
    end
    return f, err
end

local function collectResults(success, ...)
    local n = select('#', ...)
    return success, { n = n, ... }
end

function execLine(line)
    local f, err = autoReturn(line)
    if not f then
        error(err)
        return
    end

    local _, results = collectResults(pcall(f))
    for _,v in ipairs(results) do
        inspect(v)
    end
end

function ping(m, input)
    m:set(input, -1)
    time.sleep(50)
    m:set(input, 1)
    time.sleep(50)
    m:set(input, -1)
end
