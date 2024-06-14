package luax

import (
	"errors"
	"fmt"
	"reflect"

	lua "github.com/yuin/gopher-lua"
)

func Args(l *lua.LState, startIndex int, target any) error {
	firstArg := l.Get(startIndex)
	switch firstArg.Type() {
	case lua.LTNil:
		return nil
	case lua.LTTable:
		return ToGo(l, firstArg, target)
	default:
		// Process args sequentially
		return sequentialArgs(l, startIndex, target)
	}
}

func sequentialArgs(l *lua.LState, startIndex int, target any) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Ptr {
		return errors.New("target must be a pointer to a struct")
	}
	v = v.Elem()
	if v.Kind() != reflect.Struct {
		return errors.New("target must be a pointer to a struct")
	}

	for i := 0; i < v.NumField(); i++ {
		if err := toGo(l, l.Get(startIndex+i), v.Field(i)); err != nil {
			return fmt.Errorf("error processing arg #%d: %w", startIndex+i, err)
		}
	}

	return nil
}
