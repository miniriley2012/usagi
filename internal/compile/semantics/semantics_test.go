package semantics

import (
	"errors"
	"testing"

	"codeberg.org/rileyq/usagi/internal/compile/ast"
	"codeberg.org/rileyq/usagi/internal/compile/parser"
)

const std = `
const printf: func(fmt: [*]u8) i32 = @extern("printf");
`

const main = `
const std = @import("std");
`

func loadModule(name string, src string, info *Info, importer Importer) (*ast.Module, *Module, error) {
	moduleAst, err := parser.ParseBytes(name, []byte(src))
	if err != nil {
		return nil, nil, err
	}

	moduleInterface, err := Check(&CheckConfig{
		Module:   moduleAst,
		Info:     info,
		Importer: importer,
	})
	if err != nil {
		return moduleAst, nil, err
	}

	return moduleAst, moduleInterface, nil
}

func TestSemantics(t *testing.T) {
	importer := &testImporter{}

	_, stdModule, err := loadModule("std", std, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	importer.Add("std", stdModule)

	info := Info{
		Types: map[ast.Expr]Type{},
	}
	_, _, err = loadModule("main", main, &info, importer)
	if err != nil {
		t.Fatal(err)
	}

	// t.Log(info.Types[stdAst.Decls[0].(*ast.Binding).Type])
	t.Log(Universe)
}

type testImporter struct {
	imports map[string]*Module
}

func (importer *testImporter) Add(name string, module *Module) {
	if importer.imports == nil {
		importer.imports = map[string]*Module{}
	}

	importer.imports[name] = module
}

func (importer *testImporter) Import(name string) (*Module, error) {
	module, found := importer.imports[name]
	if !found {
		return nil, errors.New("module not found")
	}
	return module, nil
}
