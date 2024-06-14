package expr

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIterator(t *testing.T) {
	type expecedItem struct {
		text string
		next bool
		err  bool
		form Form
	}
	cases := []struct {
		input    string
		expected []expecedItem
	}{
		{
			input: "Hello, World!",
			expected: []expecedItem{
				{
					next: true,
					text: "Hello, World!",
					form: Raw,
				},
				{next: false},
			},
		},
		{
			input: "${expr}",
			expected: []expecedItem{
				{
					next: true,
					text: "expr",
					form: LuaExpr,
				},
				{next: false},
			},
		},
		{
			input: "${{expr}}",
			expected: []expecedItem{
				{
					next: true,
					text: "{expr}",
					form: LuaExpr,
				},
				{next: false},
			},
		},
		{
			input: `${"string"}`,
			expected: []expecedItem{
				{
					next: true,
					text: `"string"`,
					form: LuaExpr,
				},
				{next: false},
			},
		},
		{
			input: `${'string'}`,
			expected: []expecedItem{
				{
					next: true,
					text: `'string'`,
					form: LuaExpr,
				},
				{next: false},
			},
		},
		{
			input: `${{"string1", 'string2', key = 42, key2 = {"nes}ted"}}}`,
			expected: []expecedItem{
				{
					next: true,
					text: `{"string1", 'string2', key = 42, key2 = {"nes}ted"}}`,
					form: LuaExpr,
				},
				{next: false},
			},
		},
		{
			input: `$[print("this is a Lua block")
-- this is a comment with a ]
return 42]`,
			expected: []expecedItem{
				{
					next: true,
					text: `print("this is a Lua block")
-- this is a comment with a ]
return 42`,
					form: LuaChunk,
				},
				{next: false},
			},
		},
		{
			input: `Raw text, followed by ${expr}, followed by raw text.`,
			expected: []expecedItem{
				{
					next: true,
					text: `Raw text, followed by `,
					form: Raw,
				},
				{
					next: true,
					text: `expr`,
					form: LuaExpr,
				},
				{
					next: true,
					text: `, followed by raw text.`,
					form: Raw,
				},
				{next: false},
			},
		},
		{
			input: `${"error"`,
			expected: []expecedItem{
				{err: true},
			},
		},
	}

	for i, c := range cases {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			assert := assert.New(t)
			require := require.New(t)

			it := Parse(c.input)
			for j, expected := range c.expected {
				j, expected := j, expected
				assert.Equal(expected.next, it.Next(), "Iteration %d", j)
				if expected.err {
					assert.Error(it.Error())
				} else {
					require.NoError(it.Error())
					assert.Equal(expected.text, it.Text(), "Iteration %d", j)
					assert.Equal(expected.form, it.Form(), "Iteration %d", j)
				}
			}
		})
	}
}
