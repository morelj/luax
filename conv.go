package luax

import lua "github.com/yuin/gopher-lua"

func AsTable(l *lua.LState, v lua.LValue) *lua.LTable {
	return as[*lua.LTable](l, lua.LTTable, v)
}

func AsString(l *lua.LState, v lua.LValue) lua.LString {
	return as[lua.LString](l, lua.LTString, v)
}

func AsNumber(l *lua.LState, v lua.LValue) lua.LNumber {
	return as[lua.LNumber](l, lua.LTNumber, v)
}

func as[T lua.LValue](l *lua.LState, t lua.LValueType, v lua.LValue) T {
	if v, ok := v.(T); ok {
		return v
	}

	l.RaiseError("invalid type: expected %v, got %v", t, v.Type())
	panic("unreachable")
}
