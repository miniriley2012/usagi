package semantics

import (
	"fmt"
	"strconv"

	"codeberg.org/rileyq/usagi/internal/compile/ast"
	"codeberg.org/rileyq/usagi/internal/compile/token"
)

type CheckConfig struct {
	Module          *ast.Module
	Info            *Info
	Importer        Importer
	CheckFuncBodies bool
}

func Check(cfg *CheckConfig) (*Module, error) {
	var p pass
	p.info = cfg.Info
	p.importer = cfg.Importer
	p.checkFuncBodies = cfg.CheckFuncBodies
	return p.Apply(cfg.Module)
}

type pass struct {
	scope           *Scope
	cur             *Scope
	resultLocation  *symbol
	info            *Info
	importer        Importer
	checkFuncBodies bool
	returnType      Type
}

func (p *pass) Apply(moduleAst *ast.Module) (*Module, error) {
	module := p.module(moduleAst)
	return module, nil
}

func (p *pass) module(m *ast.Module) *Module {
	scope := NewScope(Universe, m.Pos(), m.End(), fmt.Sprintf("module %q", m.Name))
	curModule := &Module{name: m.Name, scope: scope}
	scope.module = curModule
	p.cur = scope
	p.scope = scope
	for _, decl := range m.Decls {
		p.decl(decl)
	}
	p.scope = nil
	p.cur = nil
	return curModule
}

func (p *pass) decl(decl ast.Decl) {
	switch decl := decl.(type) {
	case *ast.Binding:
		p.binding(decl)
	default:
		panic(fmt.Errorf("unhandled decl node %T", decl))
	}
}

func (p *pass) binding(b *ast.Binding) {
	var typeResult *TypeAndValue
	var valueResult *TypeAndValue

	sym := NewSymbol(b.Name.Name, NewTypeAndValue(nil, nil))
	p.resultLocation = sym

	if b.Type != nil {
		typeResult = p.expr(b.Type)
		sym.tv.typ = typeResult.Value().(*TypeValue).Type()
	}

	if b.Value != nil {
		valueResult = p.expr(b.Value)

		if typeResult != nil && !valueResult.Type().IsAssignableTo(typeResult.Type()) {
			panic(fmt.Errorf("%s is not assignable to %s", valueResult.Type(), typeResult.Type()))
		} else {
			sym.tv.typ = valueResult.Type()
		}

		sym.tv.val = valueResult.Value()
	}

	if p.info != nil && p.info.Uses != nil {
		p.info.Uses[b.Name] = sym
	}

	p.resultLocation = nil
	p.cur.Insert(sym)
}

func (p *pass) stmt(stmt ast.Stmt) {
	switch stmt := stmt.(type) {
	case *ast.DeclStmt:
		p.decl(stmt.X)
	case *ast.ExprStmt:
		p.expr(stmt.X)
	default:
		panic(fmt.Sprintf("unexpected ast.Stmt: %#v", stmt))
	}
}

func (p *pass) expr(expr ast.Expr) *TypeAndValue {
	tv := p.expr2(expr)
	if tv.Type() != nil && p.info != nil && p.info.Types != nil {
		p.info.Types[expr] = tv
	}
	return tv
}

func (p *pass) expr2(expr ast.Expr) *TypeAndValue {
	switch expr := expr.(type) {
	case *ast.Literal:
		switch expr.Tok {
		case token.String:
			value, err := strconv.Unquote(expr.Value)
			if err != nil {
				panic(err)
			}
			val := NewStringLiteral(value)
			return NewTypeAndValue(val.Type(), val)
		case token.Integer:
			value, err := NewIntegerLiteralFromString(expr.Value)
			if err != nil {
				panic(err)
			}
			return NewTypeAndValue(value.Type(), value)
		default:
			panic(fmt.Errorf("unknown token %q for literal", expr.Tok))
		}
	case *ast.Identifier:
		integer := NewIntegerTypeFromName(expr.Name)
		if integer != nil {
			return NewTypeAndValue(integer, NewTypeValue(integer))
		}
		sym := p.cur.Lookup(expr.Name)
		if sym != nil {
			if p.info != nil && p.info.Uses != nil {
				p.info.Uses[expr] = sym
			}
			return NewTypeAndValue(sym.Type(), sym.Value())
		}
		return nil
	case *ast.FuncExpr:
		var comment string
		if p.resultLocation != nil {
			comment = fmt.Sprintf("func %q", p.resultLocation.Name())
		} else {
			comment = "func"
		}

		funcScope := NewScope(p.cur, token.NoPos, token.NoPos, comment)
		p.cur = funcScope
		defer func() {
			p.cur = funcScope.parent
		}()

		params := make([]*NameAndType, 0, len(expr.Params))
		for _, param := range expr.Params {
			typ := p.expr(param.Type).Value().(*TypeValue).Type()
			tv := NewNameAndType(param.Name.Name, typ)
			params = append(params, tv)
			funcScope.Insert(NewSymbol(tv.Name(), NewTypeAndValue(tv.Type(), nil)))
		}
		returnType := p.expr(expr.ReturnType).Value().(*TypeValue).Type()
		sig := NewSignature(params, returnType)
		if expr.Body == nil {
			return NewTypeAndValue(sig, NewTypeValue(sig))
		}
		if p.checkFuncBodies {
			oldReturnType := p.returnType
			p.returnType = returnType
			defer func() { p.returnType = oldReturnType }()
			for _, stmt := range expr.Body.List {
				p.stmt(stmt)
			}
		}
		return NewTypeAndValue(sig, nil)
	case *ast.ManyPointerExpr:
		typ := NewManyPointer(p.expr(expr.Base).Value().(*TypeValue).Type())
		return NewTypeAndValue(typ, NewTypeValue(typ))
	case *ast.CallExpr:
		base := p.expr(expr.Base)
		args := make([]*TypeAndValue, 0, len(expr.Args))
		for _, argNode := range expr.Args {
			args = append(args, p.expr(argNode))
		}
		return p.call(base, args)
	case *ast.MemberExpr:
		base := p.expr(expr.Base)
		return p.member(base, expr.Member.Name)
	case *ast.StructExpr:
		members := make([]*NameAndType, 0, len(expr.Members))
		for _, member := range expr.Members {
			name := member.Name.Name
			typ := p.expr(member.Type).Value().(*TypeValue).Type()
			members = append(members, NewNameAndType(name, typ))
		}
		typ := NewStructType(members)
		return NewTypeAndValue(typ, NewTypeValue(typ))
	case *ast.ReturnExpr:
		value := p.expr(expr.Value)
		if !value.Type().IsAssignableTo(p.returnType) {
			panic(fmt.Errorf("%s is not assignable to return type %s", value.Type(), p.returnType))
		}
		return NewTypeAndValue(NewIntegerType(false, 0), value.Value())
	case *ast.BinaryExpr:
		left := p.expr(expr.Left)
		right := p.expr(expr.Right)
		switch expr.Op {
		case token.Plus:
			l, isLeftInt := left.Type().(*IntegerType)
			r, isRightInt := right.Type().(*IntegerType)
			if !isLeftInt || !isRightInt || !l.Equal(r) {
				panic(fmt.Errorf("cannot add %s and %s", left.Type(), right.Type()))
			}
			if left.Value() != nil && right.Value() != nil {
				l := left.Value().(*IntegerLiteral)
				r := right.Value().(*IntegerLiteral)
				value := NewIntegerLiteral(l.Value().Add(l.Value(), r.Value()))
				return NewTypeAndValue(value.Type(), value)
			}
			return NewTypeAndValue(l, nil)
		case token.Assign:
			leftType := left.Type()
			rightType := right.Type()
			if !rightType.IsAssignableTo(leftType) {
				panic(fmt.Errorf("%s is not assignable to %s", rightType, leftType))
			}
			return NewTypeAndValue(NewIntegerType(false, 0), nil)
		default:
			panic(fmt.Sprintf("unexpected token.Type: %#v", expr.Op))
		}
	case *ast.NamedArg:
		arg := NewNamedArgument(expr.Name.Name, p.expr(expr.Value))
		return NewTypeAndValue(arg.Type(), arg.Value())
	default:
		panic(fmt.Errorf("unhandled expr node %T", expr))
	}
}

func (p *pass) member(base *TypeAndValue, member string) *TypeAndValue {
	if moduleImport, isImport := base.Value().(*ModuleImport); isImport {
		sym := moduleImport.Module().Scope().Lookup(member)
		if sym == nil {
			panic(fmt.Errorf("member %q not found in module", member))
		}
		return NewTypeAndValue(sym.Type(), sym.Value())
	}

	if structType, isStruct := base.Type().(*StructType); isStruct {
		for _, field := range structType.Members() {
			if field.name == member {
				return NewTypeAndValue(field.Type(), nil)
			}
		}
	}

	panic(fmt.Errorf("unhandled base %q for member expression", base))
}

func (p *pass) call(base *TypeAndValue, args []*TypeAndValue) *TypeAndValue {
	if builtin, isBuiltin := base.Value().(*Builtin); isBuiltin {
		return p.builtin(builtin, args)
	}

	if sig, isSig := base.Type().(*Signature); isSig {
		return NewTypeAndValue(sig.ReturnType(), nil)
	}

	if structType, isStruct := base.Value().(*TypeValue).Type().(*StructType); isStruct {
		if len(args) != len(structType.Members()) {
			panic(fmt.Errorf("wrong arguments for struct constructor"))
		}

		for i := range args {
			if !args[i].typ.IsAssignableTo(structType.Members()[i].Type()) {
				panic(fmt.Errorf("%s is not assignable to %s", args[i].typ, structType.Members()[i].Type()))
			}
		}

		return NewTypeAndValue(structType, nil)
	}

	panic(fmt.Errorf("unhandled base for call: %T", base.Type()))
}

func (p *pass) builtin(builtin *Builtin, args []*TypeAndValue) *TypeAndValue {
	switch builtin.id {
	case BuiltinImport:
		if len(args) != 1 {
			panic(fmt.Errorf("Incorrect number of args for %s", builtin))
		}
		name := args[0].Value().(*StringLiteral).Value()
		if p.importer == nil {
			panic(fmt.Errorf("@import used but no importer is set"))
		}
		module, err := p.importer.Import(name)
		if err != nil {
			panic(err)
		}
		val := NewModuleImport(module)
		return NewTypeAndValue(val.Type(), val)
	case BuiltinExtern:
		if len(args) != 1 {
			panic(fmt.Errorf("Incorrect number of args for %s", builtin))
		}
		typ := p.resultLocation.Type()
		linkName := args[0].Value().(*StringLiteral).Value()
		p.resultLocation.linkName = linkName
		return NewTypeAndValue(typ, NewExternalSymbol(linkName, typ))
	default:
		panic(fmt.Sprintf("unexpected semantics.BuiltinID: %#v", builtin.id))
	}
}

type Module struct {
	name  string
	scope *Scope
}

func (m *Module) Name() string { return m.name }

func (m *Module) Scope() *Scope { return m.scope }

type Info struct {
	Types  map[ast.Expr]*TypeAndValue
	Defs   map[*ast.Identifier]Symbol
	Uses   map[*ast.Identifier]Symbol
	Scopes map[ast.Node]*Scope
}

type Importer interface {
	Import(name string) (*Module, error)
}
