package luax

import (
	"fmt"
	"reflect"

	lua "github.com/yuin/gopher-lua"
)

type LuaValuer interface {
	LuaValue(*lua.LState) (lua.LValue, error)
}

func ToLua(l *lua.LState, v any) lua.LValue {
	rv := reflect.ValueOf(v)
	res, err := toLua(l, v, rv, rv.Type())
	if err != nil {
		l.RaiseError(err.Error())
	}
	return res
}

func toLua(l *lua.LState, v any, rv reflect.Value, t reflect.Type) (lua.LValue, error) {
	if v, ok := v.(LuaValuer); ok {
		return v.LuaValue(l)
	}

	// Find if there's a Lua type for this Go type
	if goType := getGoType(l, t); goType != nil && goType.metatable != lua.LNil {
		ud := l.NewUserData()
		ud.Metatable = goType.metatable
		ud.Value = v
		return ud, nil
	}

	switch rv.Kind() {
	case reflect.String:
		return lua.LString(rv.String()), nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return lua.LNumber(rv.Int()), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return lua.LNumber(rv.Uint()), nil
	case reflect.Float32, reflect.Float64:
		return lua.LNumber(rv.Float()), nil
	case reflect.Bool:
		return lua.LBool(rv.Bool()), nil
	case reflect.Struct:
		table := l.NewTable()
		if err := structToLua(l, table, rv, t); err != nil {
			return lua.LNil, err
		}
		return table, nil
	case reflect.Map:
		table := l.NewTable()
		iter := rv.MapRange()
		for iter.Next() {
			key := iter.Key()
			value := iter.Value()
			luaKey, err := toLua(l, key.Interface(), key, key.Type())
			if err != nil {
				return lua.LNil, err
			}
			luaValue, err := toLua(l, value.Interface(), value, value.Type())
			if err != nil {
				return lua.LNil, err
			}
			table.RawSet(luaKey, luaValue)
		}
		return table, nil
	case reflect.Slice, reflect.Array:
		table := l.NewTable()
		if err := numericKeysToLua(l, table, rv); err != nil {
			return lua.LNil, err
		}
		return table, nil
	case reflect.Pointer, reflect.Interface:
		elem := rv.Elem()
		return toLua(l, elem.Interface(), elem, elem.Type())
	default:
		return lua.LNil, fmt.Errorf("unsupported kind: %v", rv.Kind())
	}
}

func structToLua(l *lua.LState, target *lua.LTable, rv reflect.Value, t reflect.Type) error {
	for i := 0; i < rv.NumField(); i++ {
		tag := luaStructTagOf(t.Field(i))
		switch {
		case tag.NumericKeys:
			if err := numericKeysToLua(l, target, rv.Field(i)); err != nil {
				return err
			}
		case tag.Inline:
			field := rv.Field(i)
			if field.Kind() != reflect.Struct {
				return fmt.Errorf("field %s: inline is only allowed on structs", t.Field(i).Name)
			}
			if err := structToLua(l, target, field, t.Field(i).Type); err != nil {
				return err
			}
		case !tag.Ignore:
			field := rv.Field(i)

			fieldValue, err := toLua(l, field.Interface(), field, field.Type())
			if err != nil {
				return fmt.Errorf("field %s: %w", t.Field(i).Name, err)
			}
			target.RawSet(lua.LString(tag.FieldName), fieldValue)
		}
	}
	return nil
}

func numericKeysToLua(l *lua.LState, table *lua.LTable, rv reflect.Value) error {
	for i := 0; i < rv.Len(); i++ {
		item := rv.Index(i)
		itemValue, err := toLua(l, item.Interface(), item, item.Type())
		if err != nil {
			return fmt.Errorf("item %d: %w", i, err)
		}
		table.RawSetInt(i+1, itemValue)
	}
	return nil
}
