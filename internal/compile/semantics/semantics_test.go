package semantics

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

func add(arg: TwoInts) i32 {
	return arg.a + arg.b;
}

export func main() i32 {
	std.printf("call through module member\n");
	thing("call through local declaration\n");
	std.printf("2 + 2 = %d", add(TwoInts(a: 2, b: 2)));
	return 0;
}
`

func loadModule(name string, src string, info *Info, importer Importer) (*ast.Module, *Module, error) {
	moduleAst, err := parser.ParseBytes(name, []byte(src))
	if err != nil {
		return nil, nil, err
	}

	moduleInterface, err := Check(&CheckConfig{
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

func TestSemantics(t *testing.T) {
	importer := &testImporter{}

	_, stdModule, err := loadModule("std", std, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	importer.Add("std", stdModule)

	info := Info{
		Types: map[ast.Expr]*semantics.TypeAndValue{},
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
