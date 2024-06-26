package luax

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	lua "github.com/yuin/gopher-lua"
)

func TestToLua(t *testing.T) {
	cases := []struct {
		value        any
		expected     lua.LValue
		expectedFunc func(l *lua.LState) lua.LValue
	}{
		{
			value:    "Hello, World!",
			expected: lua.LString("Hello, World!"),
		},
		{
			value:    42,
			expected: lua.LNumber(42),
		},
		{
			value:    true,
			expected: lua.LTrue,
		},
		{
			value: map[string]string{
				"key": "value",
			},
			expectedFunc: func(l *lua.LState) lua.LValue {
				t := l.NewTable()
				t.RawSet(lua.LString("key"), lua.LString("value"))
				return t
			},
		},
		{
			value: []string{"one", "two", "three"},
			expectedFunc: func(l *lua.LState) lua.LValue {
				t := l.NewTable()
				t.RawSet(lua.LNumber(1), lua.LString("one"))
				t.RawSet(lua.LNumber(2), lua.LString("two"))
				t.RawSet(lua.LNumber(3), lua.LString("three"))
				return t
			},
		},
		{
			value:    map[string]string(nil),
			expected: lua.LNil,
		},
		{
			value:    []string(nil),
			expected: lua.LNil,
		},
		{
			value: [3]string{"one", "two", "three"},
			expectedFunc: func(l *lua.LState) lua.LValue {
				t := l.NewTable()
				t.RawSet(lua.LNumber(1), lua.LString("one"))
				t.RawSet(lua.LNumber(2), lua.LString("two"))
				t.RawSet(lua.LNumber(3), lua.LString("three"))
				return t
			},
		},
		{
			value:    nil,
			expected: lua.LNil,
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			assert := assert.New(t)

			l := lua.NewState()
			res := ToLua(l, c.value)

			expected := c.expected
			if c.expectedFunc != nil {
				expected = c.expectedFunc(l)
			}
			assert.Equal(expected, res)
		})
	}
}
