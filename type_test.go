package luax

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	lua "github.com/yuin/gopher-lua"
)

type Person struct {
	FirstName string `lua:"first_name"`
	LastName  string `lua:"last_name"`
}

func (p *Person) luaFullName(l *lua.LState) int {
	l.Push(lua.LString("Full name is: " + p.FirstName + " " + p.LastName))
	return 1
}

type Person2 struct {
	FirstName string
	LastName  string
}

func (p *Person2) FromLuaValue(l *lua.LState, v lua.LValue) error {
	p.FirstName = l.GetField(v, "f").String()
	p.LastName = l.GetField(v, "l").String()
	return nil
}

func (p *Person2) LuaIndex(l *lua.LState, index int) int {
	switch l.CheckString(index) {
	case "first_name":
		l.Push(lua.LString(p.FirstName))
		return 1
	case "last_name":
		l.Push(lua.LString(p.LastName))
		return 1
	}
	return 0
}

func (p *Person2) LuaNewIndex(l *lua.LState, index, value int) {
	switch l.CheckString(index) {
	case "first_name":
		p.FirstName = l.CheckString(value)
	case "last_name":
		p.LastName = l.CheckString(value)
	}
}

func TestRegisterType(t *testing.T) {
	assert := assert.New(t)

	l := lua.NewState()

	RegisterType(l, "person", (*Person)(nil),
		LuaMethod("full_name", (*Person).luaFullName),
	)
	p := &Person{
		FirstName: "Chuck",
		LastName:  "Norris",
	}
	chuck := ToLua(l, p)
	l.SetGlobal("chuck", chuck)
	if err := l.DoString(`
		chuck.last_name = "Berry"
		return chuck:full_name()`); err != nil {
		t.Fatal(err)
	}

	assert.Equal(lua.LString("Full name is: Chuck Berry"), l.Get(1))
	assert.Equal("Chuck", p.FirstName)
	assert.Equal("Berry", p.LastName)
}

func xTestRegisterType2(t *testing.T) {
	l := lua.NewState()

	RegisterType(l, "person2", (*Person2)(nil))
	p := &Person2{
		FirstName: "Chuck",
		LastName:  "Norris",
	}
	chuck := ToLua(l, p)
	l.SetGlobal("chuck", chuck)
	if err := l.DoString(`
		chuck.last_name = "Berry"
		return "chuck: " .. chuck.first_name .. " " .. chuck.last_name`); err != nil {
		t.Fatal(err)
	}
	t.Fatalf("%v // %s %s", l.Get(1), p.FirstName, p.LastName)
}

func xTestPreloadModule(t *testing.T) {
	l := lua.NewState()

	RegisterType(l, "person", (*Person)(nil),
		LuaMethod("full_name", (*Person).luaFullName),
	)
	PreloadModule(l, "mymodule",
		Constructor("person", (*Person)(nil)),
	)

	if err := l.DoString(`
		local mymodule = require "mymodule"
		local chuck = mymodule.person {
			first_name = "Chuck",
			last_name = "Norris",
		}
		return chuck:full_name()
		`); err != nil {
		t.Fatal(err)
	}
	t.Fatalf("%v", l.Get(1))
}

func TestConstructor(t *testing.T) {
	assert := assert.New(t)

	l := lua.NewState()

	RegisterType(l, "person2", (*Person2)(nil))
	PreloadModule(l, "mymodule",
		Constructor("person", (*Person2)(nil)),
	)

	if err := l.DoString(`
		local mymodule = require "mymodule"
		local chuck = mymodule.person {
			f = "Chuck",
			l = "Norris",
		}
		return chuck.last_name .. ", " .. chuck.first_name
		`); err != nil {
		t.Fatal(err)
	}
	assert.Equal(lua.LString("Norris, Chuck"), l.Get(1))
}

type I interface {
	Do() string
}

type FirstI struct {
	Value string `lua:"value"`
}

func (f *FirstI) Do() string {
	return f.Value
}

type OtherI struct {
	Value int `lua:"value"`
}

func (o *OtherI) Do() string {
	return fmt.Sprintf("=> %d", o.Value)
}

func TestInterface(t *testing.T) {
	assert := assert.New(t)
	type S struct {
		Name string `lua:"name"`
		I    I      `lua:"i"`
	}

	l := lua.NewState()
	RegisterType(l, "s", (*S)(nil))
	RegisterType(l, "first_i", (*FirstI)(nil))
	RegisterType(l, "other_i", (*OtherI)(nil))
	PreloadModule(l, "mymodule",
		Constructor("s", (*S)(nil)),
		Constructor("first_i", (*FirstI)(nil)),
		Constructor("other_i", (*OtherI)(nil)),
	)
	if err := l.DoString(`
		local mymodule = require "mymodule"
		local s1 = mymodule.s {
			name = "test",
			i = mymodule.first_i {
				value = "test value"
			}
		}
		local s2 = mymodule.s {
			name = "test 2",
			i = mymodule.other_i {
				value = 42
			}
		}
		return s1.i, s2.i
	`); err != nil {
		t.Fatal(err)
	}

	i1 := l.Get(1).(*lua.LUserData).Value.(I)
	assert.Equal("test value", i1.Do())
	i2 := l.Get(2).(*lua.LUserData).Value.(I)
	assert.Equal("=> 42", i2.Do())
}
