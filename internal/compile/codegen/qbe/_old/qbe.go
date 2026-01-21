package qbe

import (
	"fmt"
	"io"
	"strconv"

	"codeberg.org/rileyq/usagi/internal/compile/ast"
	"codeberg.org/rileyq/usagi/internal/compile/semantics"
	"codeberg.org/rileyq/usagi/internal/compile/token"
)

type Pass struct {
	b    *Builder
	info *semantics.Info
}

func (p *Pass) Apply(w io.Writer, module *ast.Module) {
	p.module(module)
	p.b.WriteTo(w)
}

func (p *Pass) module(module *ast.Module) {
	for _, binding := range module.Bindings {
		p.binding(binding)
	}
}

func (p *Pass) binding(binding *ast.Binding) {
	sym := p.info.Def[binding.Name]
	_, isFunc := sym.Type().(*semantics.Func)
	if isFunc {
		typ := sym.Type().(*semantics.Func)
		val := binding.Value.(*ast.FuncExpr)
		fn := p.b.Function()
		fn.Name(sym.LinkName())
		if binding.Mode.Export() {
			fn.Export()
		}
		for _, param := range val.Params {
			fn.Param(p.info.Def[param.Name])
		}
		fn.Returns(convertType(typ.ReturnType))
		p.block(val.Body, fn.Body())
	}
}

func (p *Pass) block(expr *ast.BlockExpr, b *BlockBuilder) {
	for _, stmt := range expr.List {
		p.stmt(stmt, b)
	}
	if _, isRet := b.insts[len(b.insts)-1].(*Ret); !isRet {
		b.Ret(nil)
	}
}

func (p *Pass) stmt(stmt ast.Stmt, b *BlockBuilder) {
	switch stmt := stmt.(type) {
	case *ast.ExprStmt:
		p.expr(stmt.X, b)
	}
}

func (p *Pass) expr(expr ast.Expr, b *BlockBuilder) Value {
	switch expr := expr.(type) {
	case *ast.Literal:
		switch expr.Tok {
		case token.Integer:
			n, err := strconv.ParseUint(expr.Value, 10, 64)
			if err != nil {
				panic(err)
			}
			return NewIntegerConstant(int64(n))
		case token.String:
			val, err := strconv.Unquote(expr.Value)
			if err != nil {
				panic(err)
			}
			return p.b.StringLiteral(val)
		}
	case *ast.Identifier:
		sym := b.Symbol(p.info.Use[expr])
		if sym != nil {
			return sym
		}
		return NewGlobal(p.info.Use[expr].LinkName())
	case *ast.ReturnExpr:
		var value Value
		if expr.Value != nil {
			value = p.expr(expr.Value, b)
		}
		b.Ret(value)
		return nil
	case *ast.BinaryExpr:
		l := p.expr(expr.Left, b)
		r := p.expr(expr.Right, b)
		switch expr.Op {
		case token.Plus:
			return b.Add(convertType(p.info.Types[expr]), l, r)
		case token.Minus:
			return b.Sub(convertType(p.info.Types[expr]), l, r)
		case token.Less:
			return b.Slt(convertType(p.info.Types[expr]), l, r)
		}
	case *ast.CallExpr:
		target := p.expr(expr.Base, b)
		args := make([]*TypedValue, 0, len(expr.Args))
		for _, arg := range expr.Args {
			typ := p.info.Types[arg]
			val := p.expr(arg, b)
			args = append(args, NewTypedValue(convertType(typ), val))
		}
		returnType := convertType(p.info.Types[expr.Base].(*semantics.Func).ReturnType)
		if returnType != "" {
			return b.CallAssign(returnType, target, args)
		} else {
			b.Call(target, args)
		}
		return nil
	case *ast.MemberExpr:
		sym := p.info.Selections[expr]
		return NewGlobal(sym.LinkName())
	case *ast.IfExpr:
		cond := p.expr(expr.Cond, b)
		then := b.Block()
		els := b.Block()
		b.Jnz(cond, then, els)
		b.Emit(then)
		p.block(expr.Block, b)
		b.Emit(els)
		return nil
	}
	panic("TODO")
}

type Builder struct {
	funcs []*FunctionBuilder
	data  []*DataBuilder
}

func (b *Builder) Function() *FunctionBuilder {
	fb := &FunctionBuilder{}
	b.funcs = append(b.funcs, fb)
	return fb
}

func (b *Builder) Data() *DataBuilder {
	db := &DataBuilder{}
	b.data = append(b.data, db)
	return db
}

func (b *Builder) StringLiteral(value string) Value {
	name := "lstr" + strconv.Itoa(len(b.data))
	db := b.Data()
	db.Name(name)
	db.StringLiteral(value)
	return NewGlobal(name)
}

func (b *Builder) WriteTo(w io.Writer) (int64, error) {
	var total int64
	for _, fb := range b.funcs {
		n, err := fb.WriteTo(w)
		total += n
		if err != nil {
			return total, err
		}
	}
	for _, db := range b.data {
		n, err := db.WriteTo(w)
		total += n
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

type DataBuilder struct {
	exported bool
	name     string
	items    []*DataItem
}

func (b *DataBuilder) Export() { b.exported = true }

func (b *DataBuilder) Name(name string) { b.name = "$" + name }

func (b *DataBuilder) StringLiteral(value string) {
	b.items = append(b.items, &DataItem{"b", strconv.Quote(value)}, &DataItem{"b", "0"})
}

func (b *DataBuilder) WriteTo(w io.Writer) (int64, error) {
	if b.exported {
		io.WriteString(w, "exported ")
	}
	io.WriteString(w, "data ")
	io.WriteString(w, b.name)
	io.WriteString(w, " = ")
	io.WriteString(w, "{")
	for i, item := range b.items {
		item.WriteTo(w)
		if i < len(b.items)-1 {
			io.WriteString(w, ", ")
		}
	}
	io.WriteString(w, "}\n")
	return 0, nil
}

type DataItem struct {
	typ   string
	value string
}

func (d *DataItem) WriteTo(w io.Writer) (int64, error) {
	io.WriteString(w, d.typ)
	io.WriteString(w, " ")
	io.WriteString(w, d.value)
	return 0, nil
}

type FunctionBuilder struct {
	exported bool
	name     string
	returns  string
	body     *BlockBuilder
	params   []*TypedValue
}

func (b *FunctionBuilder) Export() { b.exported = true }

func (b *FunctionBuilder) Name(name string) { b.name = "$" + name }

func (b *FunctionBuilder) Param(symbol *semantics.Symbol) {
	body := b.Body()
	t := body.Temporary()
	body.names[symbol] = t
	b.params = append(b.params, NewTypedValue(convertType(symbol.Type()), t))
}

func (b *FunctionBuilder) Returns(typ string) { b.returns = typ }

func (b *FunctionBuilder) Body() *BlockBuilder {
	if b.body == nil {
		b.body = &BlockBuilder{temporaries: 1, names: map[*semantics.Symbol]Value{}}
	}
	return b.body
}

func (b *FunctionBuilder) WriteTo(w io.Writer) (int64, error) {
	if b.exported {
		io.WriteString(w, "export ")
	}
	io.WriteString(w, "function ")
	if len(b.returns) > 0 {
		io.WriteString(w, b.returns)
		io.WriteString(w, " ")
	}
	io.WriteString(w, b.name)
	io.WriteString(w, "(")
	for i, param := range b.params {
		param.WriteTo(w)
		if i < len(b.params)-1 {
			io.WriteString(w, ", ")
		}
	}
	io.WriteString(w, ") ")
	b.body.WriteTo(w)
	return 0, nil
}

type BlockBuilder struct {
	temporaries int
	blocks      int
	names       map[*semantics.Symbol]Value
	insts       []Instruction
}

func (b *BlockBuilder) Temporary() *Temporary {
	t := &Temporary{b.temporaries}
	b.temporaries++
	return t
}

func (b *BlockBuilder) Block() *Label {
	l := NewLabel("b" + strconv.Itoa(b.blocks))
	b.blocks++
	return l
}

func (b *BlockBuilder) Symbol(sym *semantics.Symbol) Value {
	t, found := b.names[sym]
	if found {
		return t
	}

	return nil
}

func (b *BlockBuilder) WriteTo(w io.Writer) (int64, error) {
	io.WriteString(w, "{\n")
	for _, inst := range b.insts {
		inst.WriteTo(w)
		io.WriteString(w, "\n")
	}
	io.WriteString(w, "}\n")
	return 0, nil
}

func (b *BlockBuilder) Emit(inst Instruction) {
	if len(b.insts) == 0 {
		if _, isLabel := inst.(*Label); !isLabel {
			b.Emit(NewLabel("start"))
		}
	}

	b.insts = append(b.insts, inst)
}

func (b *BlockBuilder) Add(typ string, l, r Value) Value {
	out := b.Temporary()
	b.Emit(NewAdd(typ, out, l, r))
	return out
}

func (b *BlockBuilder) Sub(typ string, l, r Value) Value {
	out := b.Temporary()
	b.Emit(NewSub(typ, out, l, r))
	return out
}

func (b *BlockBuilder) Ult(typ string, l, r Value) Value {
	out := b.Temporary()
	b.Emit(NewUlt(typ, out, l, r))
	return out
}

func (b *BlockBuilder) Slt(typ string, l, r Value) Value {
	out := b.Temporary()
	b.Emit(NewSlt(typ, out, l, r))
	return out
}

func (b *BlockBuilder) Ret(value Value) {
	b.Emit(NewRet(value))
}

func (b *BlockBuilder) CallAssign(typ string, target Value, args []*TypedValue) *Temporary {
	out := b.Temporary()
	b.Emit(NewCallAssign(typ, out, target, args))
	return out
}

func (b *BlockBuilder) Call(target Value, args []*TypedValue) {
	b.Emit(NewCall(target, args))
}

func (b *BlockBuilder) Jnz(value Value, then *Label, els *Label) {
	b.Emit(NewJnz(value, then, els))
}

func convertType(typ semantics.Type) string {
	switch typ := typ.(type) {
	case *semantics.IntegerType:
		switch typ.Bits() {
		case 0:
			return ""
		case 32:
			return "w"
		case 64:
			return "l"
		}
	case *semantics.Slice:
		return "l"
	}
	panic(fmt.Errorf("%T not handled", typ))
}

type Value interface {
	isValue()
}

type IntegerConstant struct{ value int64 }

func NewIntegerConstant(value int64) *IntegerConstant { return &IntegerConstant{value} }

func (i *IntegerConstant) String() string { return strconv.FormatUint(uint64(i.value), 10) }

func (*IntegerConstant) isValue() {}

type Global struct{ name string }

func NewGlobal(name string) *Global { return &Global{name} }

func (g *Global) String() string { return "$" + g.name }

func (*Global) isValue() {}

type Temporary struct{ n int }

func (*Temporary) isValue() {}

func (t *Temporary) String() string {
	return fmt.Sprintf("%%r%d", t.n)
}

type Instruction interface {
	io.WriterTo

	isInstruction()
}

type threeAddress struct {
	name string
	typ  string
	out  *Temporary
	a, b Value
}

func (t *threeAddress) WriteTo(w io.Writer) (int64, error) {
	fmt.Fprintf(w, "\t%s =%s %s %s, %s", t.out, t.typ, t.name, t.a, t.b)
	return 0, nil
}

func (*threeAddress) isInstruction() {}

type Add struct{ threeAddress }

func NewAdd(typ string, out *Temporary, a, b Value) *Add {
	return &Add{threeAddress{"add", typ, out, a, b}}
}

type Sub struct{ threeAddress }

func NewSub(typ string, out *Temporary, a, b Value) *Sub {
	return &Sub{threeAddress{"sub", typ, out, a, b}}
}

type Ult struct{ threeAddress }

func NewUlt(typ string, out *Temporary, a, b Value) *Ult {
	return &Ult{threeAddress{"cultw", typ, out, a, b}}
}

type Slt struct{ threeAddress }

func NewSlt(typ string, out *Temporary, a, b Value) *Slt {
	return &Slt{threeAddress{"csltw", typ, out, a, b}}
}

type Ret struct{ value Value }

func NewRet(value Value) *Ret { return &Ret{value} }

func (r *Ret) WriteTo(w io.Writer) (int64, error) {
	if r.value != nil {
		fmt.Fprintf(w, "\tret %s", r.value)
	} else {
		io.WriteString(w, "\tret")
	}
	return 0, nil
}

func (*Ret) isInstruction() {}

type Jnz struct {
	value Value
	then  *Label
	els   *Label
}

func NewJnz(value Value, then *Label, els *Label) *Jnz { return &Jnz{value, then, els} }

func (j *Jnz) WriteTo(w io.Writer) (int64, error) {
	fmt.Fprintf(w, "\tjnz %s, @%s, @%s", j.value, j.then.name, j.els.name)
	return 0, nil
}

func (*Jnz) isInstruction() {}

type Label struct{ name string }

func NewLabel(name string) *Label { return &Label{name} }

func (l *Label) WriteTo(w io.Writer) (int64, error) {
	io.WriteString(w, "@")
	io.WriteString(w, l.name)
	return 0, nil
}

func (l *Label) isInstruction() {}

type Call struct {
	target Value
	args   []*TypedValue
}

func NewCall(target Value, args []*TypedValue) *Call {
	return &Call{target, args}
}

func (c *Call) WriteTo(w io.Writer) (int64, error) {
	io.WriteString(w, "\t")
	return c.writeTo(w)
}

func (c *Call) writeTo(w io.Writer) (int64, error) {
	io.WriteString(w, "call ")
	io.WriteString(w, fmt.Sprint(c.target))
	io.WriteString(w, "(")
	for i, arg := range c.args {
		arg.WriteTo(w)
		if i < len(c.args)-1 {
			io.WriteString(w, ", ")
		}
	}
	io.WriteString(w, ")")
	return 0, nil
}

func (*Call) isInstruction() {}

type CallAssign struct {
	Call
	out *Temporary
	typ string
}

func NewCallAssign(typ string, out *Temporary, target Value, args []*TypedValue) *CallAssign {
	return &CallAssign{Call{target, args}, out, typ}
}

func (c *CallAssign) WriteTo(w io.Writer) (int64, error) {
	fmt.Fprintf(w, "\t%s =%s ", c.out, c.typ)
	return c.Call.writeTo(w)
}

type TypedValue struct {
	Type  string
	Value Value
}

func NewTypedValue(typ string, value Value) *TypedValue {
	return &TypedValue{typ, value}
}

func (tv *TypedValue) WriteTo(w io.Writer) (int64, error) {
	fmt.Fprintf(w, "%s %s", tv.Type, tv.Value)
	return 0, nil
}
