package luax

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

func CheckTable(l *lua.LState, v lua.LValue) *lua.LTable {
	return check[*lua.LTable](l, lua.LTTable, v)
}

func CheckString(l *lua.LState, v lua.LValue) string {
	if lv, ok := v.(lua.LString); ok {
		return string(lv)
	} else if lua.LVCanConvToString(v) {
		return lua.LVAsString(lv)
	}
	l.RaiseError("invalid type: expected %v, got %v", lua.LTString, v.Type())
	panic("unreachable")
}

func CheckNumber(l *lua.LState, v lua.LValue) lua.LNumber {
	return check[lua.LNumber](l, lua.LTNumber, v)
}

func CheckInt(l *lua.LState, v lua.LValue) int {
	return int(check[lua.LNumber](l, lua.LTNumber, v))
}

func CheckBool(l *lua.LState, v lua.LValue) bool {
	return bool(check[lua.LBool](l, lua.LTBool, v))
}

func check[T lua.LValue](l *lua.LState, t lua.LValueType, v lua.LValue) T {
	if v, ok := v.(T); ok {
		return v
	}

	l.RaiseError("invalid type: expected %v, got %v", t, v.Type())
	panic("unreachable")
}

func As[T lua.LValue](v lua.LValue) (T, error) {
	if v, ok := v.(T); ok {
		return v, nil
	}

	var zero T
	return zero, fmt.Errorf("invalid type: expected %v, got %v", zero.Type(), v.Type())
}
