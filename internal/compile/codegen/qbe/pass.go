package qbe

import (
	"fmt"

	"codeberg.org/rileyq/usagi/internal/compile/ast"
	"codeberg.org/rileyq/usagi/internal/compile/semantics"
	"codeberg.org/rileyq/usagi/internal/compile/token"
)

func Translate(module *ast.Module, info *semantics.Info) (*Module, error) {
	pass := pass{info: info}
	return pass.module(module), nil
}

type pass struct {
	info      *semantics.Info
	curModule *ModuleBuilder
	curFunc   *FunctionBuilder
	curBlock  *BlockBuilder
}

func (p *pass) module(node *ast.Module) *Module {
	p.curModule = &ModuleBuilder{}
	for _, decl := range DeclarationOrder(node, p.info) {
		def := p.decl(decl)
		if def != nil {
			p.curModule.Add(def)
		}
	}
	return p.curModule.Module()
}

func (p *pass) decl(decl ast.Decl) Definition {
	switch decl := decl.(type) {
	case *ast.Binding:
		return p.binding(decl)
	default:
		panic("TODO")
	}
}

func (p *pass) binding(binding *ast.Binding) Definition {
	val := p.expr(binding.Value)

	if val, isFunc := val.(*Function); isFunc {
		if binding.Mode.Export() {
			val.Linkage |= LinkageExport
		}
		val.Name = binding.Name.Name
	}

	return val
}

func (p *pass) expr(expr ast.Expr) Value {
	tv := p.info.Types[expr]

	switch expr := expr.(type) {
	case *ast.Literal:
		val := p.info.Types[expr].Value()
		if val != nil {
			return p.asValue(val)
		}
		panic("TODO")
	case *ast.Identifier:
		sym := p.info.Uses[expr]
		if val := sym.Value(); val != nil {
			return p.asValue(val)
		}
		if _, isSig := sym.Type().(*semantics.Signature); isSig {
			return NewGlobal(sym.LinkName(), p.asType(sym.Type()))
		}
		panic("TODO")
	case *ast.FuncExpr:
		return p.funcExpr(expr)
	case *ast.CallExpr:
		semTyp := p.info.Types[expr]
		typ := p.asType(semTyp.Type())
		base := p.expr(expr.Base)
		args := make([]Value, 0, len(expr.Args))
		for _, arg := range expr.Args {
			args = append(args, p.expr(arg))
		}
		var out Value
		if typ != nil {
			out = p.curFunc.Temporary(typ)
		}
		p.curBlock.Add(NewCall(out, typ, base, args))
		return out
	case *ast.MemberExpr:
		if val := tv.Value(); val != nil {
			return p.asValue(val)
		}
		panic("TODO")
	case *ast.ReturnExpr:
		value := p.expr(expr.Value)
		p.curBlock.Add(NewRet(value))
		return nil
	case *ast.BinaryExpr:
		if val := tv.Value(); val != nil {
			return p.asValue(val)
		}
		left := p.expr(expr.Left)
		right := p.expr(expr.Right)
		switch expr.Op {
		case token.Plus:
			typ := p.asType(p.info.Types[expr].Type())
			result := p.curFunc.Temporary(typ)
			p.curBlock.Add(NewAdd(result, typ, left, right))
			return result
		}
		panic("TODO")
	default:
		panic(fmt.Errorf("unhandled expr: %T", expr))
	}
}

func (p *pass) funcExpr(expr *ast.FuncExpr) *Function {
	var fb FunctionBuilder
	startBlock := BlockBuilder{name: "start"}
	sig := p.info.Types[expr].Type().(*semantics.Signature)
	for _, param := range expr.Params {
		fb.Param(param.Name.Name, nil)
	}
	fb.Returns(p.asType(sig.ReturnType()))
	oldFunc := p.curFunc
	p.curFunc = &fb
	p.curBlock = &startBlock
	for _, stmt := range expr.Body.List {
		switch stmt := stmt.(type) {
		case *ast.ExprStmt:
			p.expr(stmt.X)
		}
	}
	p.curFunc = oldFunc
	fb.Add(startBlock.Block())
	return fb.Function()
}

func (p *pass) asType(typ semantics.Type) Type {
	switch typ := typ.(type) {
	case *semantics.Signature:
		return Long
	case *semantics.IntegerType:
		if typ.Bits() <= 32 {
			return Word
		} else if typ.Bits() <= 64 {
			return Double
		} else {
			panic("TODO")
		}
	default:
		panic(fmt.Errorf("unhandled type: %s", typ))
	}
}

func (p *pass) asValue(val semantics.Value) Value {
	switch val := val.(type) {
	case *semantics.StringLiteral:
		return p.curModule.StringLiteral(val.Value())
	case *semantics.IntegerLiteral:
		return Constant(val.Value().Int64())
	case *semantics.ExternalSymbol:
		return NewGlobal(val.Name(), p.asType(val.Type()))
	default:
		panic(fmt.Errorf("unhandled value: %T", val))
	}
}

func DeclarationOrder(module *ast.Module, info *semantics.Info) []ast.Decl {
	var p declarationOrderPass
	p.info = info
	p.seen = map[semantics.Symbol]struct{}{}
	p.defs = map[semantics.Symbol]*ast.Binding{}
	return p.module(module)
}

type declarationOrderPass struct {
	info *semantics.Info
	uses []semantics.Symbol
	defs map[semantics.Symbol]*ast.Binding
	seen map[semantics.Symbol]struct{}
}

func (p *declarationOrderPass) module(module *ast.Module) []ast.Decl {
	for _, decl := range module.Decls {
		p.decl(decl)
	}
	decls := make([]ast.Decl, 0, len(p.uses))
	for _, use := range p.uses {
		decls = append(decls, p.defs[use])
	}
	return decls
}

func (p *declarationOrderPass) decl(decl ast.Decl) {
	switch decl := decl.(type) {
	case *ast.Binding:
		p.binding(decl)
	}
}

func (p *declarationOrderPass) binding(binding *ast.Binding) {
	sym := p.info.Defs[binding.Name]
	p.def(sym, binding)
	if binding.Type != nil {
		p.expr(binding.Type)
	}
	if binding.Value != nil {
		p.expr(binding.Value)
	}
	p.see(sym)
}

func (p *declarationOrderPass) expr(expr ast.Expr) {
	switch expr := expr.(type) {
	case *ast.Literal:
	case *ast.Identifier:
		if sym := p.info.Uses[expr]; sym != nil {
			p.see(sym)
		}
	case *ast.FuncExpr:
		for _, param := range expr.Params {
			p.expr(param.Type)
		}
		p.expr(expr.ReturnType)
		p.expr(expr.Body)
	case *ast.BlockExpr:
		for _, stmt := range expr.List {
			switch stmt := stmt.(type) {
			case *ast.DeclStmt:
				p.decl(stmt.X)
			case *ast.ExprStmt:
				p.expr(stmt.X)
			}
		}
	case *ast.CallExpr:
		p.expr(expr.Base)
		for _, arg := range expr.Args {
			p.expr(arg)
		}
	case *ast.MemberExpr:
		p.expr(expr.Base)
	case *ast.ReturnExpr:
		p.expr(expr.Value)
	case *ast.StructExpr:
		for _, member := range expr.Members {
			p.expr(member.Type)
		}
	case *ast.BinaryExpr:
		p.expr(expr.Left)
		p.expr(expr.Right)
	default:
		panic(fmt.Errorf("unhandled ast.Expr: %T", expr))
	}
}

func (p *declarationOrderPass) def(sym semantics.Symbol, decl *ast.Binding) {
	p.defs[sym] = decl
}

func (p *declarationOrderPass) see(sym semantics.Symbol) {
	if _, found := p.seen[sym]; found {
		return
	}
	switch sym.Value().(type) {
	case *semantics.ExternalSymbol, *semantics.ModuleImport:
		return
	}
	if sym.Scope().Module() == nil || sym.Scope() != sym.Scope().Module().Scope() {
		return
	}

	p.seen[sym] = struct{}{}
	p.uses = append(p.uses, sym)
}
