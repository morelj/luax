package expr

import (
	"strings"

	lua "github.com/yuin/gopher-lua"
)

func Evaluate(l *lua.LState, expr string) (lua.LValue, error) {
	iter := Parse(expr)
	results := make([]lua.LValue, 0, 5)
	for iter.Next() {
		switch iter.Form() {
		case Raw:
			results = append(results, lua.LString(iter.Text()))
		case LuaExpr:
			res, err := evalLua(l, "return "+iter.Text())
			if err != nil {
				return nil, err
			}
			results = append(results, res)
		case LuaChunk:
			res, err := evalLua(l, iter.Text())
			if err != nil {
				return nil, err
			}
			results = append(results, res)
		}
	}
	if iter.Error() != nil {
		return nil, iter.Error()
	}

	if len(results) == 1 {
		// Only one result, return it as is
		return results[0], nil
	}

	// Concatenate all results as a single string
	var buf strings.Builder
	for _, res := range results {
		buf.WriteString(res.String())
	}
	return lua.LString(buf.String()), nil
}

func evalLua(l *lua.LState, code string) (lua.LValue, error) {
	if err := l.DoString(code); err != nil {
		return nil, err
	}
	res := l.Get(1)
	l.Pop(l.GetTop())
	return res, nil
}
