package lua

var rackLua = `
Rack = {}

function with(o, fn)
	return fn(o)
end

local loadrack = function(path)
	local r = dofile(path)
	if type(r.build) ~= 'function' then
		error('rack at "' .. path .. '" does not implement "build" method.')
	end
	if type(r.patch) ~= 'function' then
		error('rack at "' .. path .. '" does not implement "patch" method.')
	end
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

local rack = nil

Rack.mount = function(r)
	synth.Engine:set { input = r.output() }
	rack = r
end

function Rack.inspect(o, prefix)
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
				Rack.inspect(v, ' - ')
			end
		end
	end
end

function Rack.clear()
	synth.Engine:set { input = 0 }
end

function Rack.load(path)
	local r = loadrack(path)
	r.path = filepath.dir(path)
	local built = r:build()

	assert(built.modules, 'modules should be exposed in the build action')
	assert(built.output, 'output should be exposed in the build action')

	r:patch(built.modules)

	local obj = {
		mount = function(self)
			Rack.mount(self)
		end,
		output = built.output,
		rebuild = function(self)
			r = loadrack(path)
			r.path = filepath.dir(path)
			Rack.clear()
			reset(built.modules)
			close(built.modules)
			built = r:build()
			r:patch(built.modules)
			self:mount()
		end,
		repatch = function()
			r = loadrack(path)
			reset(built.modules)
			r:patch(built.modules)
		end
	}
	setmetatable(obj, { 
		__index = function(table, k)
			if k == "modules" then
				return built.modules
			end
			return rawget(built.modules, k)
		end
	})
	return obj
end
`
