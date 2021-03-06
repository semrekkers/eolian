local filepath = require('eolian.filepath')

local environment = {
    assert       = assert,
    channel      = channel,
    coroutine    = coroutine,
    debug        = debug,
    dofile       = dofile,
    error        = error,
    getmetatable = getmetatable,
    io           = io,
    ipairs       = ipairs,
    load         = load,
    loadfile     = loadfile,
    loadstring   = loadstring,
    math         = math,
    next         = next,
    os           = os,
    pairs        = pairs,
    pcall        = pcall,
    print        = print,
    rawequal     = rawequal,
    rawget       = rawget,
    rawset       = rawset,
    require      = require,
    select       = select,
    setmetatable = setmetatable,
    string       = string,
    table        = table,
    tonumber     = tonumber,
    tostring     = tostring,
    type         = type,
    unpack       = unpack,
    xpcall       = xpcall,
}

local close = function(v)
    if type(v) ~= 'table' then
        return
    end
    if type(v.close) == 'function' then
        v:close()
        return
    end
    for _, s in pairs(v) do close(s) end
end

local startPatch = function(v)
    if type(v) ~= 'table' then
        return
    end
    if type(v.startPatch) == 'function' then
        v:startPatch()
        return
    end
    for _, s in pairs(v) do startPatch(s) end
end

local finishPatch = function(v)
    if type(v) ~= 'table' then
        return
    end
    if type(v.finishPatch) == 'function' then
        v:finishPatch()
        return
    end
    for _, s in pairs(v) do finishPatch(s) end
end

local mount = function(sinks)
    if #sinks == 1 then
        Engine:set { left = sinks[1], right = sinks[1] }
    elseif #sinks == 2 then
        Engine:set { left = sinks[1], right = sinks[2] }
    elseif #sinks > 2 then
        error('too many return values from patch')
    end
end

Rack = {
    env = {
        filepath = '',
        path     = '',

        -- Will be removed soon...
        require = function(self, path)
            return dofile(self.path .. '/' .. path)
        end,

        dofile = function(self, path)
            return dofile(self.path .. '/' .. path)
        end
    },
    modules = nil
}

function Rack.clear()
    Engine:set { left = 0, right = 0 }
end

function Rack.build()
    assert(Rack.modules ~= nil, 'no rackfile loaded.')

    Rack.clear()
    close(Rack.modules)

    local build, patch, modules
    local status, err, result = xpcall(function()
        local file = dofile(Rack.env.filepath)
        setfenv(file, environment)
        build, patch = file(Rack.env)
        modules = build()
    end, debug.traceback)
    if not(result) and err ~= nil then
        print(err)
        return
    end

    Rack.modules = modules
    local result, err = pcall(function()
        startPatch(Rack.modules)
        local sinks = {patch(Rack.modules)}
        finishPatch(Rack.modules)
        mount(sinks)
    end)
    if not result then
        print(err)
    end
end

function Rack.patch()
    assert(Rack.modules ~= nil, 'no rackfile loaded.')

    local patch
    local status, err, result = xpcall(function()
        local file = dofile(Rack.env.filepath)
        setfenv(file, environment)
        _, patch = file(Rack.env)
    end, debug.traceback)
    if not(result) and err ~= nil then
        print(err)
        return
    end

    local status, err, result = xpcall(function()
        startPatch(Rack.modules)
        local sinks = {patch(Rack.modules)}
        finishPatch(Rack.modules)
        mount(sinks)
    end, debug.traceback)
    if not(result) and err ~= nil then
        print(err)
    end
end

local originalPath = package.path

function Rack.load(path)
    package.path = filepath.dir(path) .. '/?.lua;' .. originalPath

    Rack.env.filepath  = path
    Rack.env.path      = filepath.dir(path)

    local file = dofile(path)
    setfenv(file, environment)
    local build, patch = file(Rack.env)

    Rack.modules       = build(Rack.env)
    local sinks        = {patch(Rack.modules)}

    mount(sinks)
end
