local filepath = require('eolian.filepath')

local reset = function(v)
    if type(v) ~= 'table' then
        return
    end
    if type(v.reset) == 'function' then
        v.reset()
        return
    end
    for _, s in pairs(v) do reset(s) end
end

local close = function(v)
    if type(v) ~= 'table' then
        return
    end
    if type(v.close) == 'function' then
        v.close()
        return
    end
    for _, s in pairs(v) do close(s) end
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
    Engine:set { input = 0 }
end

function Rack.build()
    assert(Rack.modules ~= nil, 'no rackfile loaded.')

    Rack.clear()
    close(Rack.modules)

    local build, patch = dofile(Rack.env.filepath)(Rack.env)
    local modules = build()

    Rack.modules = modules
    local result, err = pcall(function()
        Engine:set { input = patch(Rack.modules) }
    end)
    if not result then
        print(err)
    end
end

function Rack.patch()
    assert(Rack.modules ~= nil, 'no rackfile loaded.')

    local patch
    local status, err, result = xpcall(function()
        _, patch = dofile(Rack.env.filepath)(Rack.env)
    end, debug.traceback)
    if not result then
        print(err)
        return
    end

    print(xpcall(function()
        reset(Rack.modules)
        Engine.reset()
        Engine:set { input = patch(Rack.modules) }
    end, debug.traceback))
end

local originalPath = package.path

function Rack.load(path)
    package.path = filepath.dir(path) .. '/?.lua;' .. originalPath

    Rack.env.filepath  = path
    Rack.env.path      = filepath.dir(path)
    local build, patch = dofile(path)(Rack.env)
    Rack.modules       = build(Rack.env)
    Engine:set { input = patch(Rack.modules) }
end
