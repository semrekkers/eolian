package lua

var luaUtil = `
local string = require('eolian.string')
local sort   = require('eolian.sort')

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

local function actsLikeModule(m)
	return type(m['inputs']) == 'function' and
			type(m['outputs']) == 'function' and
			type(m['output']) == 'function' and
			type(m['set']) == 'function' and
			type(m['id']) == 'function'
end

function inspect(o, prefix)
	if type(o) == 'table' and prefix == nil then
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

			for _,k in ipairs(inputNames) do
				print(k .. " <- " .. inputs[k])
			end
			for _,k in ipairs(outputNames) do
				print(k .. " -> " .. outputs[k])
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
				if actsLikeModule(v) then
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
