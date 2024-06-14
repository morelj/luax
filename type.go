package luax

import (
	"fmt"
	"reflect"

	lua "github.com/yuin/gopher-lua"
)

type LuaIndexer interface {
	LuaIndex(l *lua.LState, index int) int
}

type LuaNewIndexer interface {
	LuaNewIndex(l *lua.LState, index, value int)
}

// newIndexFunc returns the __index function
// It first calls delegate, and if no value is returned, looks into the metatable
func newIndexFunc(delegate lua.LGFunction) lua.LGFunction {
	return func(l *lua.LState) int {
		res := delegate(l)
		if res > 0 {
			return res
		}

		mt, mtOk := l.GetMetatable(l.Get(1)).(*lua.LTable)
		field, fieldOk := l.Get(2).(lua.LString)
		if mtOk && fieldOk {
			f := mt.RawGet(field)
			if f != lua.LNil {
				l.Push(f)
				return 1
			}
		}

		// No value found, return nil
		l.Push(lua.LNil)
		return 1
	}
}

func indexLuaIndexer(l *lua.LState) int {
	userData := l.CheckUserData(1)
	return userData.Value.(LuaIndexer).LuaIndex(l, 2)
}

func indexStruct(l *lua.LState) int {
	// TODO: Handle numeric keys
	userData := l.CheckUserData(1)
	index := l.CheckString(2)

	v := reflect.ValueOf(userData.Value)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	vType := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		tag := luaStructTagOf(vType.Field(i))
		if tag.FieldName != "" && index == tag.FieldName {
			l.Push(ToLua(l, field.Interface()))
			return 1
		}
	}

	return 0
}

func newIndexLuaNewIndexer(l *lua.LState) int {
	userData := l.CheckUserData(1)
	userData.Value.(LuaNewIndexer).LuaNewIndex(l, 2, 3)
	return 0
}

func newIndexStruct(l *lua.LState) int {
	// TODO: Handle numeric keys
	userData := l.CheckUserData(1)
	index := l.CheckString(2)
	value := l.Get(3)

	v := reflect.ValueOf(userData.Value)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	vType := v.Type()
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		tag := luaStructTagOf(vType.Field(i))
		if tag.FieldName != "" && index == tag.FieldName {
			if err := toGo(l, value, field); err != nil {
				l.RaiseError(err.Error())
			}
		}
	}
	return 0
}

func CheckUserData[T any](l *lua.LState, n int) T {
	ud := l.CheckUserData(n)
	if v, ok := ud.Value.(T); ok {
		return v
	}

	var t T
	l.ArgError(n, fmt.Sprintf("%T expected - got %T", t, ud.Value))
	return t
}

func AsUserData[T any](l *lua.LState, v lua.LValue) T {
	if v, ok := v.(*lua.LUserData); ok {
		if t, ok := v.Value.(T); ok {
			return t
		}

		var t T
		l.RaiseError("%T expected - got %T", t, v.Value)
	}

	l.RaiseError("invalid type: expected userdata, got %v", v.Type())
	panic("unreachable")
}

// NewUserData returns a new UserData containing the given value.
// If a Go type is registered for the type of value, the corresponding metatable is associated to the userdata.
func NewUserData(l *lua.LState, value any) *lua.LUserData {
	userData := l.NewUserData()
	userData.Value = value

	t := reflect.TypeOf(value)
	if goType := getGoType(l, t); goType != nil {
		userData.Metatable = goType.metatable
	}
	return userData
}

type Method func() (string, lua.LGFunction, error)

func LuaMethod[T any](name string, f func(T, *lua.LState) int) Method {
	return func() (string, lua.LGFunction, error) {
		return name, func(l *lua.LState) int {
			t := CheckUserData[T](l, 1)
			return f(t, l)
		}, nil
	}
}

func GoMethod(name string, f any) Method {
	return func() (string, lua.LGFunction, error) {
		rf := reflect.ValueOf(f)
		t := rf.Type()
		if t.Kind() != reflect.Func {
			return name, nil, fmt.Errorf("invalid type: expected function")
		}
		if t.NumIn() == 0 {
			return name, nil, fmt.Errorf("invalid function type: missing receiver arg")
		}

		return name, func(l *lua.LState) int {
			args := make([]reflect.Value, t.NumIn())

			// First arg must be receiver
			args[0] = reflect.ValueOf(l.CheckUserData(1).Value)

			// Get following args
			for i := 1; i < t.NumIn(); i++ {
				// TODO
			}

			// TODO
			return 0
		}, nil
	}
}

type goTypeDescriptor struct {
	metatable *lua.LTable
}

func getGoType(l *lua.LState, t reflect.Type) *goTypeDescriptor {
	if m := getGoTypesMap(l); m != nil {
		return m[t]
	}
	return nil
}

func setGoType(l *lua.LState, t reflect.Type, gt *goTypeDescriptor) {
	if m := getGoTypesMap(l); m != nil {
		m[t] = gt
	} else {
		reg := l.Get(lua.RegistryIndex)
		m := map[reflect.Type]*goTypeDescriptor{t: gt}
		ud := l.NewUserData()
		ud.Value = m
		l.SetField(reg, "__go_types", ud)
	}
}

func getGoTypesMap(l *lua.LState) map[reflect.Type]*goTypeDescriptor {
	reg := l.Get(lua.RegistryIndex)
	t := l.GetField(reg, "__go_types")
	if t == lua.LNil {
		return nil
	}

	return t.(*lua.LUserData).Value.(map[reflect.Type]*goTypeDescriptor)
}

// RegisterType register a new Go type in l.
// The type is registered under name.
// v must be a value of the source type (usually the zero value), and methods the methods to expose in Lua.
func RegisterType(l *lua.LState, name string, v any, methods ...Method) {
	goType := reflect.TypeOf(v)

	mt := l.NewTypeMetatable(name)

	// Register the global Go type
	setGoType(l, goType, &goTypeDescriptor{
		metatable: mt,
	})

	funcs := make(map[string]lua.LGFunction)

	// Add methods
	for _, method := range methods {
		name, f, err := method()
		if err != nil {
			panic(fmt.Errorf("failed to register method %s: %w", name, err))
		}
		funcs[name] = f
	}

	// We need to get the underlying type if this is a pointer
	if goType.Kind() == reflect.Pointer {
		goType = goType.Elem()
	}

	// Find operations
	if _, ok := v.(LuaIndexer); ok {
		funcs["__index"] = newIndexFunc(indexLuaIndexer)
	} else if goType.Kind() == reflect.Struct {
		funcs["__index"] = newIndexFunc(indexStruct)
	}

	if _, ok := v.(LuaNewIndexer); ok {
		funcs["__newindex"] = newIndexLuaNewIndexer
	} else if goType.Kind() == reflect.Struct {
		funcs["__newindex"] = newIndexStruct
	}

	l.SetFuncs(mt, funcs)
}
