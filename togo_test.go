package luax

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	lua "github.com/yuin/gopher-lua"
)

type person struct {
	FirstName string `lua:"first_name"`
	LastName  string `lua:"last_name"`
}

type test struct {
	String    string              `lua:"string"`
	Int       int                 `lua:"int"`
	Float     float64             `lua:"float"`
	Bool      bool                `lua:"bool"`
	Person    person              `lua:"person"`
	PersonPtr *person             `lua:"person_ptr"`
	IntPtr    *int                `lua:"int_ptr"`
	Array     []string            `lua:"array"`
	Map       map[string]string   `lua:"map"`
	Map2      map[int]string      `lua:"map2"`
	Map3      map[string][]string `lua:"map3"`
}

type userData struct {
	Value string
}

type fromLuaValuerTest struct {
	Value string
}

func (flvt *fromLuaValuerTest) FromLuaValue(l *lua.LState, v lua.LValue) error {
	flvt.Value = "lua:" + v.String()
	return nil
}

func TestToGo(t *testing.T) {
	var anything any
	cases := []struct {
		skip     bool
		lua      func(*lua.LState) (lua.LValue, error)
		target   any
		expected any
		err      bool
	}{
		{
			lua: func(l *lua.LState) (lua.LValue, error) {
				err := l.DoString(`return "test"`)
				if err != nil {
					return nil, err
				}
				return l.Get(1), nil
			},
			target: &fromLuaValuerTest{},
			expected: &fromLuaValuerTest{
				Value: "lua:test",
			},
		},
		{
			skip: true,
			lua: func(l *lua.LState) (lua.LValue, error) {
				err := l.DoString(`return {
					string = "test",
					int = 42,
					float = 13.37,
					bool = true,
					person = { first_name = "Jean", last_name = "Valjean" },
					person_ptr = { first_name = "Chuck", last_name = "Norris" },
					int_ptr = 99999,
					array = { "un", "deux", "trois", [5] = "cinq" },
					map = {
						key = "value",
						k = "v",
						kk = "vv",
					},
					map2 = { "one", "two", "three" },
					map3 = {
						chuck = { "norris", "testa" },
						david = { "hasselhoff" },
						bob = { "marley" },
					},
					}`)
				if err != nil {
					return nil, err
				}
				return l.Get(1), nil
			},
			target: &test{},
			expected: &test{
				String: "test",
				Int:    42,
				Float:  13.37,
				Bool:   true,
				Person: person{
					FirstName: "Jean",
					LastName:  "Valjean",
				},
				PersonPtr: &person{
					FirstName: "Chuck",
					LastName:  "Norris",
				},
				IntPtr: intPtr(99999),
				Array: []string{
					"un",
					"deux",
					"trois",
					"",
					"cinq",
				},
				Map: map[string]string{
					"key": "value",
					"k":   "v",
					"kk":  "vv",
				},
				Map2: map[int]string{
					1: "one",
					2: "two",
					3: "three",
				},
				Map3: map[string][]string{
					"chuck": {"norris", "testa"},
					"david": {"hasselhoff"},
					"bob":   {"marley"},
				},
			},
		},
		{
			skip: true,
			lua: func(l *lua.LState) (lua.LValue, error) {
				ud := l.NewUserData()
				ud.Value = &userData{Value: "test"}
				return ud, nil
			},
			target: &userData{},
			expected: &userData{
				Value: "test",
			},
		},
		{
			lua: func(l *lua.LState) (lua.LValue, error) {
				t := l.NewTable()
				t.RawSet(lua.LString("key"), lua.LString("value"))
				return t, nil
			},
			target: &anything,
			expected: &map[string]any{
				"key": "value",
			},
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			if c.skip {
				t.SkipNow()
			}

			assert := assert.New(t)
			require := require.New(t)

			l := lua.NewState()
			v, err := c.lua(l)
			require.NoError(err, "Lua execution error")

			// Get Lua execution result
			err = ToGo(l, v, c.target)
			if c.err {
				assert.Error(err)
			} else {
				require.NoError(err)
				assert.Equal(c.expected, c.target)
			}
		})
	}
}

func intPtr(v int) *int {
	return &v
}

func float64Ptr(v float64) *float64 {
	return &v
}
