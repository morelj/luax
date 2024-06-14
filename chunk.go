package luax

import (
	"bytes"
	"errors"
	"io/fs"

	lua "github.com/yuin/gopher-lua"
	"github.com/yuin/gopher-lua/parse"
)

type Chunk struct {
	proto  *lua.FunctionProto
	source string
}

func CompileString(source, name string) (*Chunk, error) {
	r := bytes.NewReader([]byte(source))
	chunk, err := parse.Parse(r, name)
	if err != nil {
		return nil, err
	}
	proto, err := lua.Compile(chunk, name)
	if err != nil {
		return nil, err
	}
	return &Chunk{
		source: source,
		proto:  proto,
	}, nil
}

func CompileFile(fsys fs.FS, filename string, required bool) (*Chunk, error) {
	data, err := fs.ReadFile(fsys, filename)
	if err != nil {
		if !required && errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	return CompileString(string(data), filename)
}

func (c *Chunk) Do(l *lua.LState) error {
	l.Push(l.NewFunctionFromProto(c.proto))
	return l.PCall(0, lua.MultRet, nil)
}
