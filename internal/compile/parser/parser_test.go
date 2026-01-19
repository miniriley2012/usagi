package parser

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"codeberg.org/rileyq/usagi/internal/compile/ast"
	"codeberg.org/rileyq/usagi/internal/compile/token"
)

const src = `
const std = @import("std");

struct TwoInts (
	a: i32,
	b: i32,
);

trait Drop {
	func drop(self: Self) void;
}

// Equivalent to
//   trait Linear {}
//   impl Linear(!Drop);
trait Linear(!Drop) {}

impl TwoInts(Drop) {
	func drop(self: TwoInts) void {}
}

func genericAdd(x: forSome Integer, y: @TypeOf(x)) @TypeOf(x) {
	return x + y;
}

// Adds two i32 values
const add: func(arg: TwoInts) i32 = func(arg: TwoInts) i32 {
	return arg.a + arg.b;
};

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
	if impl.Traits != nil {
		io.WriteString(w, "(")
		for i, t := range impl.Traits {
			expr(w, t)
			if i < len(impl.Traits)-1 {
				io.WriteString(w, ", ")
			}
		}
		io.WriteString(w, ") ")
	}
	io.WriteString(w, "{\n")
	for _, def := range impl.Definitions {
		binding(w, def)
	}
	io.WriteString(w, "}\n")
}

func binding(w io.Writer, b *ast.Binding) {
	var declarators []string
	if b.Mode.Export() {
		declarators = append(declarators, "export")
	}
	if b.Mode.Const() {
		declarators = append(declarators, "const")
	}
	if b.Token != token.Const {
		declarators = append(declarators, b.Token.String())
	}
	declarators = append(declarators, b.Name.Name)

	io.WriteString(w, strings.Join(declarators, " "))

	switch b.Token {
	case token.Func:
		funcBody(w, b.Value.(*ast.FuncExpr))
	case token.Struct:
		fields(w, b.Value.(*ast.StructExpr).Members)
		io.WriteString(w, ";\n")
	case token.Trait:
		traitBody(w, b.Value.(*ast.TraitExpr))
		io.WriteString(w, "\n")
	default:
		if b.Type != nil {
			io.WriteString(w, ": ")
			expr(w, b.Type)
		}

		if b.Value != nil {
			io.WriteString(w, " = ")
			expr(w, b.Value)
		}

		io.WriteString(w, ";\n")
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
		io.WriteString(w, "struct")
		fields(w, x.Members)
	case *ast.NamedArg:
		expr(w, x.Name)
		io.WriteString(w, ": ")
		expr(w, x.Value)
	case *ast.TraitExpr:
		io.WriteString(w, "trait")
		traitBody(w, x)
	case *ast.FuncExpr:
		funcExpr(w, x)
	case *ast.UnaryExpr:
		io.WriteString(w, x.Op.String())
		expr(w, x.Base)
	case *ast.ExistentialExpr:
		io.WriteString(w, "forSome ")
		expr(w, x.Base)
	}
}

func traitBody(w io.Writer, trait *ast.TraitExpr) {
	if trait.Traits != nil {
		io.WriteString(w, "(")
		for i, t := range trait.Traits {
			expr(w, t)
			if i < len(trait.Traits)-1 {
				io.WriteString(w, ", ")
			}
		}
		io.WriteString(w, ") ")
	}
	io.WriteString(w, "{\n")
	for _, m := range trait.Members {
		binding(w, m)
	}
	io.WriteString(w, "}")
}

func funcExpr(w io.Writer, fexpr *ast.FuncExpr) {
	io.WriteString(w, "func(")
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
	}
}

func fields(w io.Writer, fields []*ast.Field) {
	io.WriteString(w, "(")
	for i, field := range fields {
		expr(w, field.Name)
		io.WriteString(w, ": ")
		expr(w, field.Type)
		if i < len(fields)-1 {
			io.WriteString(w, ", ")
		}
	}
	io.WriteString(w, ")")
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
