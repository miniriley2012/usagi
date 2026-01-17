package parser

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"codeberg.org/rileyq/usagi/internal/compile/ast"
)

const src = `
const std = @import("std");

const TwoInts = struct (
	a: i32,
	b: i32,
);

const Drop = trait {
	func drop(self: Self) void;
};

impl TwoInts(Drop) {
	func drop(self: TwoInts) void {}
}

// Adds two i32 values
func add(arg: TwoInts) i32 {
	return arg.a + arg.b;
}

func main() void {
	std.print(add(TwoInts(a: 1, b: 2)));
}
`

func TestParser(t *testing.T) {
	p := NewFromReader(bytes.NewReader([]byte(src)))
	module, err := p.Parse("main")
	if err != nil {
		t.Log(err)
		t.Fail()
	}
	var b strings.Builder
	Node(&b, module)
	t.Log(b.String())
}

func Node(w io.Writer, node ast.Node) {
	switch node := node.(type) {
	case *ast.Module:
		module(w, node)
	}
}

func module(w io.Writer, module *ast.Module) {
	for _, b := range module.Decls {
		decl(w, b)
		io.WriteString(w, "\n")
	}
}

func decl(w io.Writer, decl ast.Decl) {
	switch decl := decl.(type) {
	case *ast.Binding:
		binding(w, decl)
	case *ast.ImplDecl:
		impl(w, decl)
	}
}

func impl(w io.Writer, impl *ast.ImplDecl) {
	io.WriteString(w, "impl ")
	expr(w, impl.Type)
	io.WriteString(w, "(")
	expr(w, impl.Trait)
	io.WriteString(w, ") ")
	io.WriteString(w, "{\n")
	for _, def := range impl.Definitions {
		binding(w, def)
	}
	io.WriteString(w, "}\n")
}

func binding(w io.Writer, binding *ast.Binding) {
	var declarators []string
	if binding.Mode.Export() {
		declarators = append(declarators, "export")
	}
	if binding.Mode.Const() {
		declarators = append(declarators, "const")
	}
	if binding.Mode.Func() {
		declarators = append(declarators, "func")
	} else {
		declarators = append(declarators, "let")
	}
	declarators = append(declarators, binding.Name.Name)

	io.WriteString(w, strings.Join(declarators, " "))

	if !binding.Mode.Func() {
		if binding.Type != nil {
			io.WriteString(w, ": ")
			expr(w, binding.Type)
		}

		if binding.Value != nil {
			io.WriteString(w, " = ")
			expr(w, binding.Value)
		}

		io.WriteString(w, ";\n")
	} else {
		funcBody(w, binding.Value.(*ast.FuncExpr))
	}
}

func expr(w io.Writer, x ast.Expr) {
	switch x := x.(type) {
	case *ast.Literal:
		io.WriteString(w, x.Value)
	case *ast.Identifier:
		io.WriteString(w, x.Name)
	case *ast.CallExpr:
		call(w, x)
	case *ast.BinaryExpr:
		expr(w, x.Left)
		io.WriteString(w, " ")
		io.WriteString(w, x.Op.String())
		io.WriteString(w, " ")
		expr(w, x.Right)
	case *ast.ReturnExpr:
		io.WriteString(w, "return ")
		expr(w, x.Value)
	case *ast.MemberExpr:
		expr(w, x.Base)
		io.WriteString(w, ".")
		io.WriteString(w, x.Member.Name)
	case *ast.StructExpr:
		io.WriteString(w, "struct(")
		for i, member := range x.Members {
			expr(w, member.Name)
			io.WriteString(w, ": ")
			expr(w, member.Type)
			if i < len(x.Members)-1 {
				io.WriteString(w, ", ")
			}
		}
		io.WriteString(w, ")")
	case *ast.NamedArg:
		expr(w, x.Name)
		io.WriteString(w, ": ")
		expr(w, x.Value)
	case *ast.TraitExpr:
		io.WriteString(w, "trait {\n")
		for _, member := range x.Members {
			binding(w, member)
		}
		io.WriteString(w, "}")
	}
}

func call(w io.Writer, cexpr *ast.CallExpr) {
	expr(w, cexpr.Base)
	io.WriteString(w, "(")
	for i, arg := range cexpr.Args {
		expr(w, arg)
		if i < len(cexpr.Args)-1 {
			io.WriteString(w, ", ")
		}
	}
	io.WriteString(w, ")")
}

func funcBody(w io.Writer, fexpr *ast.FuncExpr) {
	io.WriteString(w, "(")
	for i, param := range fexpr.Params {
		funcParam(w, param)
		if i < len(fexpr.Params)-1 {
			io.WriteString(w, ", ")
		}
	}
	io.WriteString(w, ") ")
	expr(w, fexpr.ReturnType)
	if fexpr.Body != nil {
		io.WriteString(w, " ")
		blockExpr(w, fexpr.Body)
	} else {
		io.WriteString(w, ";")
	}
	io.WriteString(w, "\n")
}

func funcParam(w io.Writer, param *ast.Param) {
	io.WriteString(w, param.Name.Name)
	io.WriteString(w, ": ")
	expr(w, param.Type)
}

func blockExpr(w io.Writer, expr *ast.BlockExpr) {
	io.WriteString(w, "{\n")
	for _, st := range expr.List {
		stmt(w, st)
	}
	io.WriteString(w, "}")
}

func stmt(w io.Writer, stmt ast.Stmt) {
	switch stmt := stmt.(type) {
	case *ast.ExprStmt:
		expr(w, stmt.X)
		io.WriteString(w, ";\n")
	}
}
