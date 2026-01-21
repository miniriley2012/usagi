package qbe

import (
	"bytes"
	"testing"

	"codeberg.org/rileyq/usagi/internal/compile/ast"
	"codeberg.org/rileyq/usagi/internal/compile/parser"
	"codeberg.org/rileyq/usagi/internal/compile/semantics"
)

const stdSrc = `
export const printf: func(fmt: [*]u8, ...) i32 = @extern("printf");
`

const src = `
const std = @import("std");

const func fib(n: i32) i32 {
	if n < 2 {
		return n;
	}
	return fib(n - 1) + fib(n - 2);
}

const func do(n: i32) void {
	if n < 0 {
		return;
	}
	std.printf("result: %d\n", fib(n));
	return do(n - 1);
}

export const func main() i32 {
	do(10);
	return 0;
}
`

func parse(name string, source []byte) *ast.Module {
	p := parser.NewFromReader(bytes.NewReader(source))
	module, err := p.Parse(name)
	if err != nil {
		panic(err)
	}
	return module
}

func TestQBE(t *testing.T) {
	stdAst := parse("std", []byte(stdSrc))
	mainAst := parse("main", []byte(src))

	stdModule := semantics.NewPass(nil).Apply(stdAst, nil)

	sema := semantics.NewPass(&semantics.Config{
		CheckFuncBodies: true,
		Importer: &importer{
			modules: map[string]*semantics.Module{
				"std": stdModule,
			},
		},
	})
	info := semantics.Info{
		Types:      map[ast.Expr]semantics.Type{},
		Def:        map[*ast.Identifier]*semantics.Symbol{},
		Use:        map[*ast.Identifier]*semantics.Symbol{},
		Selections: map[*ast.MemberExpr]*semantics.Symbol{},
	}

	sema.Apply(mainAst, &info)

	codegen := &Pass{
		b:    &Builder{},
		info: &info,
	}

	codegen.Apply(t.Output(), mainAst)
}

type importer struct {
	modules map[string]*semantics.Module
}

func (i *importer) Import(name string) (*semantics.Module, error) {
	return i.modules[name], nil
}
