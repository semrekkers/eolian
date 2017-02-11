package lua

var luaUtil = `
local join      = require('eolian.string').join
local split      = require('eolian.string').split
local sort      = require('eolian.sort')
local tabwriter = require('eolian.tabwriter')

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
			error('attempt to set inputs "'.. join(keys, ', ') ..'" on nil value')
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
	return type(m) == 'table' and
			type(m['inputs']) == 'function' and
			type(m['outputs']) == 'function' and
			type(m['output']) == 'function' and
			type(m['set']) == 'function' and
			type(m['id']) == 'function'
end

function find(group, name, prefix)
	prefix = prefix or ''

	for k, v in pairs(group) do
		if actsLikeModule(v) then
			if v.id() == name then
				if prefix == "" then
					return k
				end
				return prefix .. "." .. k
			end
		elseif type(v) == 'table' then
			return find(v, name, prefix .. "." .. k)
		end
	end
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

			local w = tabwriter.new(8, 8, 1, "\t", "alignRight")
			w.write("-------------------------------------\n")
			for _,k in ipairs(inputNames) do
				local name  = inputs[k]
				local parts = split(name, '/')
				local path  = find(Rack.modules, parts[1])

				if path ~= nil then
					local rest = {}
					for i=2,#parts do
						table.insert(rest, parts[i])
					end
					name = path .. '/' .. join(rest, '/')
				end

				w.write(k .. "\t<--\t" .. name .. "\n")
			end
			for _,k in ipairs(outputNames) do
				local name  = outputs[k]
				local parts = split(name, '/')
				local path  = find(Rack.modules, parts[1])

				if path ~= nil then
					local rest = {}
					for i=2,#parts do
						table.insert(rest, parts[i])
					end
					name = path .. '/' .. join(rest, '/')
				end

				w.write(k .. "\t-->\t" .. name .. "\n")
			end
			print(w.flush())
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
