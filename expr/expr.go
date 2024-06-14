// Package expr implements an expression parser based on Lua syntax.
//
// expr extracts Lua expressions and blocks of codes from strings, following a simple syntax.
// It is not able to process the returned expressions: A third party Lua interpreter must be used
// for this purpose.
//
// expr extracts Lua expressions enclosed between ${ and }, and Lua chunks enclosed between $[ and ].
// The difference between a Lua expression and a chunk is that an expression cannot be evaluated by
// itself by a Lua interpreter.
// There are several ways to do this: The simplest is to enclose the expression into a return statement:
// lua_expr becomes return (lua_expr).
//
// The Parse() function returns an Iterator. Calling Next() on it will return the next raw text, expression
// or chunk found.
package expr

import (
	"fmt"
	"strings"
)

// Form of a value
type Form uint8

const (
	None     Form = iota // No value
	Raw                  // The value is a raw text
	LuaExpr              // The value is a Lua expression
	LuaChunk             // The value is a Lua chunk
)

type stateFuncs interface {
	emit(rune)
	pop()
	push(state)
	done(Form)
	skip()
}

type state func(fns stateFuncs, cur, nxt rune)

func stateRaw(fns stateFuncs, cur, nxt rune) {
	switch {
	case cur == '$' && nxt == '{':
		fns.done(Raw)
		fns.skip()
		fns.push(stateLuaExpr)
	case cur == '$' && nxt == '[':
		fns.done(Raw)
		fns.skip()
		fns.push(stateLuaChunk)
	default:
		fns.emit(cur)
		if nxt == 0 {
			// Last character of the input
			fns.done(Raw)
		}
	}
}

func newStateLua(form Form, delim rune) state {
	return func(fns stateFuncs, cur, nxt rune) {
		switch {
		case cur == '\'':
			fns.emit(cur)
			fns.push(stateLuaSingleString)
		case cur == '"':
			fns.emit(cur)
			fns.push(stateLuaDoubleString)
		case cur == '{':
			fns.emit(cur)
			fns.push(stateLuaTable)
		case cur == '-' && nxt == '-':
			fns.emit(cur)
			fns.push(stateLuaComment)
		case cur == delim:
			fns.pop()
			fns.done(form)
		default:
			fns.emit(cur)
		}
	}
}

func stateLuaTable(fns stateFuncs, cur, nxt rune) {
	fns.emit(cur)
	switch cur {
	case '\'':
		fns.push(stateLuaSingleString)
	case '"':
		fns.push(stateLuaDoubleString)
	case '{':
		fns.push(stateLuaTable)
	case '}':
		fns.pop()
	}
}

func stateLuaComment(fns stateFuncs, cur, nxt rune) {
	fns.emit(cur)
	if cur == '\n' {
		fns.pop()
	}
}

func newStateLuaString(delim rune) state {
	return func(fns stateFuncs, cur, nxt rune) {
		switch {
		case cur == delim:
			fns.emit(cur)
			fns.pop()
		case cur == '\\' && nxt == delim:
			fns.emit(cur)
			fns.emit(nxt)
			fns.skip()
		default:
			fns.emit(cur)
		}
	}
}

var (
	stateLuaExpr         = newStateLua(LuaExpr, '}')
	stateLuaChunk        = newStateLua(LuaChunk, ']')
	stateLuaSingleString = newStateLuaString('\'')
	stateLuaDoubleString = newStateLuaString('"')
)

// An Iterator iterates over the values of an input string.
type Iterator struct {
	buf    strings.Builder // Stores the current value
	states []state         // Current stack of states
	input  []rune          // Input text, as a rune slice
	index  int             // Current index in input
	form   Form            // Form of the current value
	err    error           // Whether an error has been encountered
}

// Call the Next() to advance to the next value.
// Returns true if there's a value, false otherwise
func (it *Iterator) Next() bool {
	if it.err != nil {
		// Don't do anything in case of error
		return false
	}

	// Reset buffer and form
	it.buf.Reset()
	it.form = None

	if it.index >= len(it.input) {
		// End of input
		return false
	}

	for ; it.index < len(it.input); it.index++ {
		cur := it.input[it.index]
		var nxt rune
		if it.index < len(it.input)-1 {
			nxt = it.input[it.index+1]
		}

		it.states[len(it.states)-1](it, cur, nxt)
		if it.form != None {
			it.index++
			break
		}
	}

	if it.form == None {
		// Error
		it.err = fmt.Errorf("invalid input at position %d", it.index)
		return false
	}
	if it.buf.Len() == 0 {
		// Empty value, skip to the next
		return it.Next()
	}

	return true
}

func (it *Iterator) emit(rune rune) {
	it.buf.WriteRune(rune)
}

func (it *Iterator) done(form Form) {
	it.form = form
}

func (it *Iterator) skip() {
	it.index += 1
}

func (it *Iterator) pop() {
	it.states = it.states[0 : len(it.states)-1]
}

func (it *Iterator) push(state state) {
	it.states = append(it.states, state)
}

// Text returns the text of the current value.
// Call this method after Next()
func (it *Iterator) Text() string {
	return it.buf.String()
}

// Form returns the form of the current value.
// Call this method after Next()
func (it *Iterator) Form() Form {
	return it.form
}

func (it *Iterator) Error() error {
	return it.err
}

// Parse returns a new Iterator which will iterate over the values of input.
// Next() must be called on the returned Iterator to get the first value.
func Parse(input string) *Iterator {
	states := make([]state, 1, 10)
	states[0] = stateRaw
	return &Iterator{
		input:  []rune(input),
		states: states,
		form:   None,
	}
}
