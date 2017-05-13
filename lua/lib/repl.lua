local join      = require('eolian.string').join
local split     = require('eolian.string').split
local sort      = require('eolian.sort')
local tabwriter = require('eolian.tabwriter')
local time      = require('eolian.time')

local function isPatcher(m)
    return type(m) == 'table' and
        type(m.inputs) == 'function' and
        type(m.outputs) == 'function' and
        type(m.out) == 'function' and
        type(m.set) == 'function' and
        type(m.id) == 'function' and
        type(m.type) == 'function'
end

local function find(group, name, prefix)
    prefix = prefix or ''

    if name == 'Engine:0' then
        return 'engine'
    end

    for k, v in pairs(group) do
        if isPatcher(v) then
            if v:id() == name then
                return join({prefix, k}, ".")
            elseif type(v.members) == 'function' then
                for _,m in ipairs(v:members()) do
                    if m == name then
                        return join({prefix, k}, ".")
                    end
                end
            end
        elseif type(v) == 'table' then
            local result = find(v, name, join({prefix, k}, "."))
            if result ~= nil then
                return result
            end
        end
    end
end

local function writeInputs(w, names, inputs)
    for _,k in ipairs(names) do
        local name  = inputs[k]
        local parts = split(name, '/')
        local path  = find((Rack.modules or {}), parts[1])

        if path ~= nil then
            local rest = {}
            for i=2,#parts do
                table.insert(rest, parts[i])
            end
            name = string.format("%s/%s", path, join(rest, '/'))
        end

        w.write(string.format("%s\t<-\t%s\t\n", k, name))
    end
end

local function writeOutputs(w, names, outputs)
    for _,k in ipairs(names) do
        local outs  = outputs[k]
        local names = {}

        for _,name in ipairs(outs) do
            local parts = split(name, '/')
            local path  = find((Rack.modules or {}), parts[1])

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
end

local function writeState(w, state)
    local stateNames = {}
    for k,v in pairs(state) do
        table.insert(stateNames, k)
    end
    stateNames = sort.strings(stateNames)

    for _,k in ipairs(stateNames) do
        w.write(string.format("%s\t==\t%s\t\n", k, state[k]))
    end
end

local function printTableStructure(t)
    local w = tabwriter.new(8, 8, 1, "\t", "alignRight")
    for k, v in pairs(t) do
        if k ~= '__namespace' then
            if isPatcher(v) then
                v = v:type()
            end
            w.write(string.format("%s\t%s\n", k, v))
        end
    end
    local s, count = string.gsub(w.flush(), "\n$", "")
    if count > 0 then print(s) end
end

local function collectNames(t)
    local n = {}
    for k,v in pairs(t) do
        table.insert(n, k)
    end
    return n
end

local function inspect(o, prefix)
    if isPatcher(o) then
        local w = tabwriter.new(5, 0, 1, " ")
        if o.state ~= nil then
            writeState(w, o:state())
        end
        local inputs  = o:inputs()
        local outputs = o:outputs()

        writeInputs(w, sort.strings(collectNames(inputs)), inputs)
        writeOutputs(w, sort.strings(collectNames(outputs)), outputs)

        local s, count = string.gsub(w.flush(), "\n$", "")
        if count > 0 then print(s) end
        return
    elseif type(o) == 'table' then
        printTableStructure(o)
    else
        print(o)
    end
end

local function autoReturn(line)
    local f, err = loadstring('local v = (' .. line ..'); if v ~= nil then return v end')
    if not f then
        f, err = loadstring(line)
    end
    return f, err
end

local function collectResults(success, ...)
    local n = select('#', ...)
    return success, { n = n, ... }
end

local function exec(line)
    local f, err = autoReturn(line)
    if not f then
        error(err)
        return
    end

    local _, results = collectResults(pcall(f))
    if results == nil then
        return
    end

    for _,v in ipairs(results) do
        inspect(v)
    end
end

return {
    isPatcher = isPatcher,
    exec      = exec,
    inspect   = inspect,
    find      = find,
}
