package luax

import (
	"fmt"
	"reflect"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

type FromLuaValuer interface {
	FromLuaValue(*lua.LState, lua.LValue) error
}

func ToGo(l *lua.LState, v lua.LValue, target any) error {
	targetValue := reflect.ValueOf(target)
	if targetValue.Kind() != reflect.Pointer {
		return fmt.Errorf("unsupported kind %v, must be %v", targetValue.Kind(), reflect.Pointer)
	}

	return toGo(l, v, targetValue.Elem())
}

func asFromLuaValuer(v reflect.Value) FromLuaValuer {
	vPtr := v
	if v.Kind() != reflect.Pointer {
		if !v.CanAddr() {
			return nil
		}
		vPtr = v.Addr()
	}

	if !vPtr.CanInterface() {
		return nil
	}

	if res, ok := vPtr.Interface().(FromLuaValuer); ok {
		return res
	}
	return nil
}

func toGo(l *lua.LState, v lua.LValue, target reflect.Value) error {
	// Check if target is userdata and target is same type
	if v, ok := v.(*lua.LUserData); ok {
		vValue := reflect.ValueOf(v.Value)
		if vValue.Type() == target.Type() {
			target.Set(vValue)
			return nil
		}
		if vValue.Kind() == reflect.Pointer && vValue.Type().Elem() == target.Type() {
			target.Set(vValue.Elem())
			return nil
		}
	}

	if fromLuaValuer := asFromLuaValuer(target); fromLuaValuer != nil {
		return fromLuaValuer.FromLuaValue(l, v)
	}

	switch target.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v, ok := v.(lua.LNumber); ok {
			target.SetInt(int64(v))
			return nil
		}
		return fmt.Errorf("type error: expected %v, got %v", lua.LTNumber, v.Type())

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if v, ok := v.(lua.LNumber); ok {
			target.SetUint(uint64(v))
			return nil
		}
		return fmt.Errorf("type error: expected %v, got %v", lua.LTNumber, v.Type())

	case reflect.Float32, reflect.Float64:
		if v, ok := v.(lua.LNumber); ok {
			target.SetFloat(float64(v))
			return nil
		}
		return fmt.Errorf("type error: expected %v, got %v", lua.LTNumber, v.Type())

	case reflect.String:
		if v, ok := v.(lua.LString); ok {
			target.SetString(string(v))
			return nil
		}
		return fmt.Errorf("type error: expected %v, got %v", lua.LTString, v.Type())

	case reflect.Bool:
		if v, ok := v.(lua.LBool); ok {
			target.SetBool(bool(v))
			return nil
		}
		return fmt.Errorf("type error: expected %v, got %v", lua.LTBool, v.Type())

	case reflect.Struct:
		return toGoStruct(l, v, target)

	case reflect.Map:
		return toGoMap(l, v, target)

	case reflect.Slice:
		return toGoSlice(l, v, target, false)

	case reflect.Pointer:
		var elem reflect.Value
		if target.IsNil() {
			elem = reflect.New(target.Type().Elem())
			target.Set(elem)
		} else {
			elem = target.Elem()
		}
		return toGo(l, v, elem)

	case reflect.Interface:
		res := toGoAny(l, v)
		if res == nil {
			target.SetZero()
		} else {
			target.Set(reflect.ValueOf(res))
		}
		return nil

	default:
		return fmt.Errorf("unsupported target type %v", target.Kind())
	}
}

func toGoMap(l *lua.LState, v lua.LValue, target reflect.Value) error {
	if v.Type() != lua.LTTable {
		return fmt.Errorf("type error: expected %v, got %v", lua.LTTable, v.Type())
	}

	if target.IsNil() {
		target.Set(reflect.MakeMap(target.Type()))
	} else {
		target.Clear()
	}

	targetType := target.Type()
	keyType := targetType.Key()
	valueType := targetType.Elem()

	var err error
	l.ForEach(v.(*lua.LTable), func(key, value lua.LValue) {
		if err != nil {
			return
		}

		mapKey := reflect.New(keyType)
		if err2 := toGo(l, key, mapKey.Elem()); err2 != nil {
			err = fmt.Errorf("invalid key %v: %w", key.String(), err2)
			return
		}

		mapValue := reflect.New(valueType)
		if err2 := toGo(l, value, mapValue.Elem()); err2 != nil {
			err = fmt.Errorf("invalid value for key %v: %w", key.String(), err2)
		}

		target.SetMapIndex(mapKey.Elem(), mapValue.Elem())
	})
	return err
}

func toGoSlice(l *lua.LState, v lua.LValue, target reflect.Value, ignoreBadKeys bool) error {
	if v.Type() != lua.LTTable {
		return fmt.Errorf("type error: expected %v, got %v", lua.LTTable, v.Type())
	}
	count := l.ObjLen(v)

	if target.IsNil() || target.Cap() < count {
		// Nil slice or insufficent capacity: allocate a new slice
		target.Set(reflect.MakeSlice(target.Type(), count, count))
	} else if target.Len() != count {
		// Sufficent capacity but incorrent len: re-slice
		target.Set(target.Slice(0, count))
		target.Clear()
	}
	var err error
	l.ForEach(v.(*lua.LTable), func(key, value lua.LValue) {
		if err != nil {
			// Short circuit if there's already an error
			return
		}

		if key.Type() == lua.LTNumber {
			i := int(key.(lua.LNumber))
			if i >= 1 {
				if err2 := toGo(l, value, target.Index(i-1)); err2 != nil {
					err = fmt.Errorf("index %d: %w", i, err2)
				}
			}
		} else if !ignoreBadKeys {
			err = fmt.Errorf("invalid key in array: %s", key.String())
		}
	})
	return err
}

func toGoStruct(l *lua.LState, v lua.LValue, target reflect.Value) error {
	if v.Type() != lua.LTTable {
		return fmt.Errorf("type error: expected %v, got %v", lua.LTTable, v.Type())
	}

	targetType := target.Type()
	for i := 0; i < targetType.NumField(); i++ {
		f := targetType.Field(i)

		tag := luaStructTagOf(f)
		switch {
		case tag.NumericKeys:
			toGoSlice(l, v, target.Field(i), true)
		case !tag.Ignore:
			fieldValue := l.GetTable(v, lua.LString(tag.FieldName))
			if fieldValue != lua.LNil {
				if err := toGo(l, fieldValue, target.Field(i)); err != nil {
					return fmt.Errorf("field '%s': %w", tag.FieldName, err)
				}
			}
		}
	}
	return nil
}

type luaStructTag struct {
	Ignore      bool
	FieldName   string
	NumericKeys bool
	Inline      bool
}

func luaStructTagOf(field reflect.StructField) luaStructTag {
	tag := field.Tag.Get("lua")
	parts := strings.Split(tag, ",")

	if len(parts) == 0 {
		// Default tag
		return luaStructTag{}
	}

	if parts[0] == "-" {
		// Ignore
		return luaStructTag{
			Ignore: true,
		}
	}

	t := luaStructTag{}
	if parts[0] == "" {
		t.FieldName = field.Name
	} else {
		t.FieldName = parts[0]
	}
	for i := 1; i < len(parts); i++ {
		switch parts[i] {
		case "numkeys":
			t.NumericKeys = true
		case "inline":
			t.Inline = true
		}
	}
	return t
}

func toGoAny(l *lua.LState, v lua.LValue) any {
	switch v := v.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(v)
	case lua.LNumber:
		return float64(v)
	case lua.LString:
		return string(v)
	case *lua.LUserData:
		return v.Value
	case *lua.LTable:
		var (
			array    []any
			strTable map[string]any
			table    map[any]any
		)

		l.ForEach(v, func(key, value lua.LValue) {
			if numKey, ok := key.(lua.LNumber); ok {
				// Numeric key

				index := int(numKey)
				switch {
				case array == nil:
					array = make([]any, index, v.Len())
				case cap(array) < index:
					// Not enough capacity in the slice: append enough elements
					array = append(array, make([]any, index-cap(array))...)
				case len(array) < index:
					// Enough capacity but not enough length: re-slice
					array = array[:index]
				}
				array[index-1] = toGoAny(l, value)
			} else if strKey, ok := key.(lua.LString); ok {
				if strTable == nil {
					strTable = make(map[string]any)
				}
				strTable[string(strKey)] = toGoAny(l, value)
			} else {
				// Other key
				if table == nil {
					table = make(map[any]any)
				}
				table[toGoAny(l, key)] = toGoAny(l, value)
			}
		})

		switch {
		case len(array) > 0 && table == nil && strTable == nil:
			return array
		case len(array) == 0 && table != nil && strTable == nil:
			return table
		case len(array) == 0 && table == nil && strTable != nil:
			return strTable
		default:
			merge(array, strTable, table)
			return table
		}
	default:
		panic(fmt.Errorf("failed to convert %s", v.Type()))
	}
}

func merge(array []any, strTable map[string]any, table map[any]any) {
	for i := range array {
		table[i] = array[i]
	}
	for k, v := range strTable {
		table[k] = v
	}
}
