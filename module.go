package luax

import (
	"fmt"
	"reflect"

	lua "github.com/yuin/gopher-lua"
)

type PreloadOption func(l *lua.LState, module *lua.LTable) error

func Constructor(name string, v any) PreloadOption {
	t := reflect.TypeOf(v)
	isPointer := false
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
		isPointer = true
	}

	return func(l *lua.LState, module *lua.LTable) error {
		f := func(l *lua.LState) int {
			v := reflect.New(t)
			if !isPointer {
				v = v.Elem()
			}
			if err := toGo(l, l.Get(1), v); err != nil {
				l.RaiseError(err.Error())
			}
			l.Push(NewUserData(l, v.Interface()))
			return 1
		}
		module.RawSetString(name, l.NewClosure(f))
		return nil
	}
}

func LuaFunction(name string, f lua.LGFunction) PreloadOption {
	return func(l *lua.LState, module *lua.LTable) error {
		module.RawSetString(name, l.NewClosure(f))
		return nil
	}
}

func Table(name string, opts ...PreloadOption) PreloadOption {
	return func(l *lua.LState, module *lua.LTable) error {
		t := l.NewTable()
		for _, opt := range opts {
			if err := opt(l, t); err != nil {
				return err
			}
		}
		module.RawSetString(name, t)
		return nil
	}
}

func Value(name string, value lua.LValue) PreloadOption {
	return func(l *lua.LState, module *lua.LTable) error {
		module.RawSetString(name, value)
		return nil
	}
}

func LuaString(source, name string) PreloadOption {
	return func(l *lua.LState, module *lua.LTable) error {
		chunk, err := CompileString(source, name)
		if err != nil {
			return err
		}
		if err := chunk.Do(l); err != nil {
			return err
		}

		// Check the top of the stack
		top := l.Get(-1)
		switch top := top.(type) {
		case *lua.LFunction:
			// Call the function with the module
			l.Push(module)
			l.Call(1, 0)
			return nil

		case *lua.LTable:
			// Populate the module with the contents of the table
			top.ForEach(func(key, value lua.LValue) {
				module.RawSet(key, value)
			})
			return nil

		default:
			return fmt.Errorf("%s: unexpected return value type: expected function or table, got %v", name, top.Type())
		}
	}
}

func PreloadModule(l *lua.LState, name string, opts ...PreloadOption) {
	l.PreloadModule(name, func(l *lua.LState) int {
		// Initialize module
		mod := l.NewTable()

		for _, opt := range opts {
			if err := opt(l, mod); err != nil {
				l.RaiseError(err.Error())
			}
		}

		l.SetField(mod, "name", lua.LString(name))
		l.Push(mod)
		return 1
	})
}
