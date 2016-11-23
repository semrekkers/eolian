package lua

var luaRack = `
function with(o, fn)
	return fn(o)
end

function inspect(o, prefix)
	if type(o) == 'table' and prefix == nil then
		if o['_type'] == 'module' then
			print(o:inspect())
			return
		end
	end
	for k, v in pairs(o) do
		if k ~= '_namespace' then
			if prefix == nil then
				prefix = ''
			end
			if type(v) == 'table' then
				print(prefix .. k)
				inspect(v, ' - ')
			end
		end
	end
end

local loadfile = function(path)
	local r = dofile(path)
	assert(type(r.build) == 'function', 'rack does not implement "build" method.')
	assert(type(r.patch) == 'function', 'rack does not implement "patch" method.')
	return r
end

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
	filepath = '',
	path	 = '',
	modules  = nil
}

function Rack.clear()
	synth.Engine:set { input = 0 }
end

function Rack.rebuild()
	assert(Rack.modules ~= nil, 'no rackfile loaded.')
	Rack.clear()
	close(Rack.modules)

	local r      = loadfile(Rack.filepath)
	Rack.modules = r:build(Rack.path)

	synth.Engine:set { input = r:patch(Rack.modules) }
end

function Rack.repatch()
	assert(Rack.modules ~= nil, 'no rackfile loaded.')
	reset(Rack.modules)
	local r = loadfile(Rack.filepath)
	r:patch(Rack.modules)
end

function Rack.load(path)
	local r = loadfile(path)

	Rack.filepath = path
	Rack.path	  = filepath.dir(path)
	Rack.modules  = with(r:build(Rack.path), function(modules)
		synth.Engine:set { input = r:patch(modules) }
		return modules
	end)
end
`
