package qbe

import (
	"errors"
	"testing"

	"codeberg.org/rileyq/usagi/internal/compile/ast"
	"codeberg.org/rileyq/usagi/internal/compile/parser"
	"codeberg.org/rileyq/usagi/internal/compile/semantics"
)

const std = `
const printf: func(fmt: [*]u8) i32 = @extern("printf");
`

const main = `
const std = @import("std");

const thing = std.printf;

struct TwoInts(a: i32, b: i32);

// func add(arg: TwoInts) i32 {
// 	return arg.a + arg.b;
// }

func add(a: i32, b: i32) i32 {
	return a + b;
}

export func main() i32 {
	std.printf("call through module member\n");
	thing("call through local declaration\n");
	std.printf("2 + 2 = %d", add(2, 2));
	return 0;
}
`

func loadModule(name string, src string, info *semantics.Info, importer semantics.Importer) (*ast.Module, *semantics.Module, error) {
	moduleAst, err := parser.ParseBytes(name, []byte(src))
	if err != nil {
		return nil, nil, err
	}

	moduleInterface, err := semantics.Check(&semantics.CheckConfig{
		Module:          moduleAst,
		Info:            info,
		Importer:        importer,
		CheckFuncBodies: true,
	})
	if err != nil {
		return moduleAst, nil, err
	}

	return moduleAst, moduleInterface, nil
}

func TestQBE(t *testing.T) {
	importer := &testImporter{}

	_, stdModule, err := loadModule("std", std, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	importer.Add("std", stdModule)

	info := semantics.Info{
		Types:  map[ast.Expr]*semantics.TypeAndValue{},
		Defs:   map[*ast.Identifier]semantics.Symbol{},
		Uses:   map[*ast.Identifier]semantics.Symbol{},
		Scopes: map[ast.Node]*semantics.Scope{},
	}
	mainAst, _, err := loadModule("main", main, &info, importer)
	if err != nil {
		t.Fatal(err)
	}

	// decls := DeclarationOrder(mainAst, &info)
	// t.Log(decls)

	module, err := Translate(mainAst, &info)
	if err != nil {
		t.Fatal(err)
	}
	_ = module

	module.WriteTo(t.Output())

	// t.Log(info.Types[stdAst.Decls[0].(*ast.Binding).Type])
	// t.Log(Universe)
}

type testImporter struct {
	imports map[string]*semantics.Module
}

func (importer *testImporter) Add(name string, module *semantics.Module) {
	if importer.imports == nil {
		importer.imports = map[string]*semantics.Module{}
	}

	importer.imports[name] = module
}

func (importer *testImporter) Import(name string) (*semantics.Module, error) {
	module, found := importer.imports[name]
	if !found {
		return nil, errors.New("module not found")
	}
	return module, nil
}
