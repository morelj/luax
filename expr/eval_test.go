package expr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

func TestEvaluate(t *testing.T) {
	cases := []struct {
		expected lua.LValue
		init     func(l *lua.LState)
		expr     string
		err      bool
	}{
		{
			expr:     "Hello, World!",
			expected: lua.LString("Hello, World!"),
		},
		{
			expr:     "${42}",
			expected: lua.LNumber(42),
		},
		{
			expr:     "${true}",
			expected: lua.LBool(true),
		},
		{
			expr:     `Hello, ${"World"}!`,
			expected: lua.LString("Hello, World!"),
		},
		{
			init: func(l *lua.LState) {
				l.SetGlobal("first_name", lua.LString("Chuck"))
				l.SetGlobal("last_name", lua.LString("Norris"))
			},
			expr:     `Hello, ${first_name} ${last_name}!`,
			expected: lua.LString("Hello, Chuck Norris!"),
		},
		{
			expr:     `$[print("I do not return anything")]`,
			expected: lua.LNil,
		},
		{
			expr:     `${1, 2, 3}`,
			expected: lua.LNumber(1),
		},
		{
			expr:     `${"a","b"}, ${"c", "d"}`,
			expected: lua.LString("a, c"),
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("%d_%s", i, c.expr), func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			l := lua.NewState()
			if c.init != nil {
				c.init(l)
			}

			res, err := Evaluate(l, c.expr)
			if c.err {
				assert.Error(err)
			} else {
				require.NoError(err)
				assert.Equal(c.expected, res)
			}
		})
	}
}
