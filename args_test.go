package luax

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	lua "github.com/yuin/gopher-lua"
)

type testArgs struct {
	First  string `lua:"first"`
	Second string `lua:"second"`
}

func newTestFunc[T any](startIndex int, expected T) func(*testing.T) lua.LGFunction {
	return func(t *testing.T) lua.LGFunction {
		return func(l *lua.LState) int {
			var args T
			Args(l, startIndex, &args)
			assert.Equal(t, &expected, &args)
			return 0
		}
	}
}

func newTable(kv ...lua.LValue) *lua.LTable {
	t := &lua.LTable{}
	for i := 0; i < len(kv); i += 2 {
		t.RawSet(kv[i], kv[i+1])
	}
	return t
}

func TestArgs(t *testing.T) {
	cases := []struct {
		f    func(t *testing.T) lua.LGFunction
		args []lua.LValue
	}{
		{
			f: newTestFunc(1, testArgs{
				First:  "first",
				Second: "second",
			}),
			args: []lua.LValue{
				lua.LString("first"),
				lua.LString("second"),
			},
		},
		{
			f: newTestFunc(1, testArgs{
				First:  "arg1",
				Second: "arg2",
			}),
			args: []lua.LValue{
				newTable(lua.LString("first"), lua.LString("arg1"), lua.LString("second"), lua.LString("arg2")),
			},
		},
		{
			f: newTestFunc(1, testArgs{
				First: "arg1",
			}),
			args: []lua.LValue{
				newTable(lua.LString("first"), lua.LString("arg1")),
			},
		},
		{
			f: newTestFunc(2, testArgs{
				First:  "arg1",
				Second: "arg2",
			}),
			args: []lua.LValue{
				lua.LString("ignore-me"),
				newTable(lua.LString("first"), lua.LString("arg1"), lua.LString("second"), lua.LString("arg2")),
			},
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			l := lua.NewState()
			l.NewTable()
			l.Push(l.NewFunction(c.f(t)))
			for _, arg := range c.args {
				l.Push(arg)
			}
			l.Call(len(c.args), 0)
		})
	}
}
