package lua

var luaRack = `
function with(o, fn)
	return fn(o)
end

function inputProxy(m)
	return function(_, inputs)
		m:set(inputs)
	end
end

function outputProxy(m)
	return function(_, output)
		if output == nil then
			return m:output()
		else
			return m:output(output)
		end
	end
end

function inspect(o, prefix)
	if type(o) == 'table' and prefix == nil then
		if o['__type'] == 'module' then
			print(o:info())
			return
		end
	end
	for k, v in pairs(o) do
		if k ~= '__namespace' then
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
	-- assert(type(r.build) == 'function', 'rack does not implement "build" method.')
	-- assert(type(r.patch) == 'function', 'rack does not implement "patch" method.')
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
	env = {
		filepath = '',
		path	 = '',
		require  = function(self, path)
			return dofile(self.path .. '/' .. path)
		end
	},
	modules = nil
}

function Rack.clear()
	synth.Engine:set { input = 0 }
end

function Rack.rebuild()
	assert(Rack.modules ~= nil, 'no rackfile loaded.')
	Rack.clear()
	close(Rack.modules)

	local build, patch = dofile(Rack.env.filepath)(Rack.env)
	Rack.modules = build()
	synth.Engine:set { input = patch(Rack.modules) }
end

function Rack.repatch()
	assert(Rack.modules ~= nil, 'no rackfile loaded.')
	reset(Rack.modules)
	local _, patch = dofile(Rack.env.filepath)(Rack.env)
	synth.Engine:set { input = patch(Rack.modules) }
end

function Rack.load(path)
	local loadFunc = dofile(path)

	Rack.env.filepath  = path
	Rack.env.path	   = filepath.dir(path)
	local build, patch = loadFunc(Rack.env)
	Rack.modules       = build(Rack.env)

	synth.Engine:set { input = patch(Rack.modules) }
end
`
