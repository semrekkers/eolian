package lua

var luaUtil = `
local string = require('eolian.string')

function with(o, fn)
	return fn(o)
end

function set(m, arg1, arg2)
	if type(arg1) == "table" then
		if m == nil then
			local keys={}
			for k,_ in pairs(arg1) do
				table.insert(keys, k)
			end
			error('attempt to set inputs "'.. string.join(keys, ', ') ..'" on nil value')
		end
		m:set(arg1)
		return m
	end

	if m == nil then
		error('attempt to set input "'.. arg1 ..'" on nil value')
	end
	m:set({ [tostring(arg1)] = arg2 })
	return m
end

function out(m, name)
	if m == nil then
		error('attempt to get output "' .. name .. '" from nil value')
	end
	return m:output(name)
end

function inspect(o, prefix)
	if type(o) == 'table' and prefix == nil then
		if o['__type'] == 'module' then
			for k,v in pairs(o.inputs()) do
				print(k .. " <- " .. v)
			end
			for k,v in pairs(o.outputs()) do
				print(k .. " -> " .. v)
			end
			return
		end
	end
	for k, v in pairs(o) do
		if k ~= '__namespace' then
			if prefix == nil then
				prefix = ''
			end
			if type(v) == 'table' then
				if v['__type'] == 'module' then
					print(prefix .. k .. " (" .. v:id() .. ")")
				else
					print(prefix .. k)
				end
				inspect(v, ' - ')
			end
		end
	end
end

`
