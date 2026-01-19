package parser

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"slices"

	"codeberg.org/rileyq/usagi/internal/compile/ast"
	"codeberg.org/rileyq/usagi/internal/compile/scanner"
	"codeberg.org/rileyq/usagi/internal/compile/token"
)

type Parser struct {
	scn  Scanner
	t    *token.Token
	errs []error
}

func (p *Parser) Parse(name string) (*ast.Module, error) {
	p.next()

	var decls []ast.Decl

	for p.t != nil {
		decl := p.decl()
		if decl != nil {
			decls = append(decls, decl)
		}
	}

	return &ast.Module{Name: name, Decls: decls}, p.wrappedError()
}

func (p *Parser) wrappedError() error {
	return errors.Join(p.errs...)
}

var topLevelRecoveryTokens = []token.Type{token.Semicolon}

func (p *Parser) decl() ast.Decl {
	switch p.t.Type {
	case token.Export, token.Const, token.Let, token.Func, token.Struct, token.Trait:
		return p.binding()
	case token.Impl:
		return p.impl()
	}
	p.unexpected("declaration")
	return nil
}

func (p *Parser) impl() *ast.ImplDecl {
	p.expect(token.Impl)

	typ := p.expr2(nil, token.PrecedenceCall)
	p.expect(token.OpenParen)
	trait := p.expr()
	p.expect(token.CloseParen)

	var defs []*ast.Binding
	p.expect(token.OpenBrace)
	for {
		if p.accept(token.CloseBrace) != nil {
			break
		}
		binding := p.binding()
		if binding != nil {
			defs = append(defs, binding)
		}
	}

	return &ast.ImplDecl{
		Type:        typ,
		Trait:       trait,
		Definitions: defs,
	}
}

func (p *Parser) binding() *ast.Binding {
	var mode ast.BindingMode
	var typ ast.Expr
	var val ast.Expr

	if p.accept(token.Export) != nil {
		mode |= ast.ModeExport
	}

	if p.accept(token.Const) != nil {
		mode |= ast.ModeConst
	}

	if mode&ast.ModeConst != 0 && p.t.Type == token.Identifier {
		name := p.identifier()

		if p.accept(token.Colon) != nil {
			typ = p.expr()
		}

		if p.accept(token.Assign) != nil {
			val = p.expr()
		}

		p.expect(token.Semicolon)

		return &ast.Binding{
			Token: token.Const,
			Mode:  mode,
			Name:  name,
			Type:  typ,
			Value: val,
		}
	}

	switch p.t.Type {
	case token.Let:
		return p.letBinding(mode)
	case token.Func:
		return p.funcBinding(mode)
	case token.Struct:
		return p.structBinding(mode)
	case token.Trait:
		return p.traitBinding(mode)
	default:
		p.unexpected("binding")
		return nil
	}
}

func (p *Parser) traitBinding(mode ast.BindingMode) *ast.Binding {
	p.expect(token.Trait)
	name := p.identifier()
	members := p.traitMembers()
	return &ast.Binding{
		Token: token.Trait,
		Mode:  mode,
		Name:  name,
		Type:  nil,
		Value: &ast.TraitExpr{Members: members},
	}
}

func (p *Parser) structBinding(mode ast.BindingMode) *ast.Binding {
	p.expect(token.Struct)
	name := p.identifier()
	members := p.fields()
	p.expect(token.Semicolon)
	return &ast.Binding{
		Token: token.Struct,
		Mode:  mode,
		Name:  name,
		Type:  nil,
		Value: &ast.StructExpr{Members: members},
	}
}

func (p *Parser) letBinding(mode ast.BindingMode) *ast.Binding {
	var typ ast.Expr
	var val ast.Expr

	p.expect(token.Let)

	name := p.identifier()

	if p.accept(token.Colon) != nil {
		typ = p.expr()
	}

	if p.accept(token.Assign) != nil {
		val = p.expr()
	}

	return &ast.Binding{
		Token: token.Let,
		Mode:  mode,
		Name:  name,
		Type:  typ,
		Value: val,
	}
}

func (p *Parser) funcBinding(mode ast.BindingMode) *ast.Binding {
	p.expect(token.Func)

	name := p.identifier()
	if name == nil {
		return nil
	}

	fn := p.funcBody()

	if fn.Body == nil {
		p.expect(token.Semicolon)
	}

	return &ast.Binding{
		Token: token.Func,
		Mode:  mode,
		Name:  name,
		Type:  nil,
		Value: fn,
	}
}

func (p *Parser) funcBody() *ast.FuncExpr {
	var params []*ast.Param
	var body *ast.BlockExpr

	p.expect(token.OpenParen)
	for {
		if p.accept(token.CloseParen) != nil {
			break
		}
		params = append(params, p.param())
		if p.accept(token.Comma) != nil {
			continue
		} else if p.accept(token.CloseParen) != nil {
			break
		} else {
			p.expect(token.CloseParen)
		}
	}

	returnType := p.expr()

	if p.t.Type == token.OpenBrace {
		body = p.blockExpr()
	}

	return &ast.FuncExpr{
		Params:     params,
		ReturnType: returnType,
		Body:       body,
	}
}

func (p *Parser) blockExpr() *ast.BlockExpr {
	var stmts []ast.Stmt

	p.expect(token.OpenBrace)
	for {
		if p.accept(token.CloseBrace) != nil {
			break
		}
		stmts = append(stmts, p.stmt())
	}

	return &ast.BlockExpr{List: stmts}
}

func (p *Parser) stmt() ast.Stmt {
	switch p.t.Type {
	case token.Return, token.Identifier:
		x := p.expr()
		p.expect(token.Semicolon)
		return &ast.ExprStmt{X: x}
	case token.If:
		x := p.expr()
		return &ast.ExprStmt{X: x}
	default:
		panic("expected statement")
	}
}

func (p *Parser) param() *ast.Param {
	if p.t.Type == token.Ellipses {
		p.next()
		return &ast.Param{
			Name: nil,
			Type: &ast.VarArgExpr{},
		}
	}

	name := p.identifier()
	p.expect(token.Colon)
	typ := p.expr()

	return &ast.Param{
		Name: name,
		Type: typ,
	}
}

func (p *Parser) expr() ast.Expr {
	return p.expr2(nil, token.PrecedenceNone)
}

func (p *Parser) expr2(left ast.Expr, prec token.Precedence) ast.Expr {
	if left == nil {
		left = p.unaryOperand()
	}

	for p.t.Type.Precedence() > prec {
		left = p.binaryExpr(left)
	}

	return left
}

func (p *Parser) binaryExpr(left ast.Expr) ast.Expr {
	t := p.t.Type
	switch t {
	case token.OpenParen:
		return p.call(left)
	case token.Dot:
		p.next()
		member := p.identifier()
		return &ast.MemberExpr{Base: left, Member: member}
	case token.Less, token.Minus, token.Plus:
		p.next()
		right := p.expr2(nil, t.Precedence())
		return &ast.BinaryExpr{
			Left:  left,
			Op:    t,
			Right: right,
		}
	default:
		return left
	}
}

func (p *Parser) call(base ast.Expr) ast.Expr {
	var args []ast.Expr

	p.expect(token.OpenParen)
	for {
		if p.accept(token.CloseParen) != nil {
			break
		}

		args = append(args, p.argument())

		if p.accept(token.CloseParen) != nil {
			break
		} else if p.accept(token.Comma) != nil {
			continue
		} else {
			p.expect(token.Comma)
		}
	}

	return &ast.CallExpr{
		Base: base,
		Args: args,
	}
}

func (p *Parser) argument() ast.Expr {
	if p.t.Type == token.Identifier {
		ident := p.identifier()
		if p.accept(token.Colon) != nil {
			value := p.expr()
			return &ast.NamedArg{
				Name:  ident,
				Value: value,
			}
		}
		return p.expr2(ident, token.PrecedenceNone)
	}
	return p.expr()
}

func (p *Parser) unaryOperand() ast.Expr {
	switch p.t.Type {
	case token.Identifier:
		return p.identifier()
	case token.String:
		return p.string()
	case token.Integer:
		return p.integer()
	case token.Return:
		var expr ast.Expr
		p.next()
		if p.t.Type != token.Semicolon {
			expr = p.expr()
		}
		return &ast.ReturnExpr{Value: expr}
	case token.Func:
		p.next()
		return p.funcBody()
	case token.Ellipses:
		p.next()
		return &ast.VarArgExpr{}
	case token.OpenBracket:
		return p.sliceOrManyPointer()
	case token.If:
		return p.ifExpr()
	case token.Struct:
		return p.structExpr()
	case token.Trait:
		return p.traitExpr()
	default:
		p.error(NewParseError(p.t.Pos, p.t.End, fmt.Errorf("expected unary operand but found %q", p.t.Type)))
		return &ast.BadExpr{}
	}
}

func (p *Parser) traitExpr() *ast.TraitExpr {
	p.expect(token.Trait)
	members := p.traitMembers()
	return &ast.TraitExpr{
		Members: members,
	}
}

func (p *Parser) traitMembers() []*ast.Binding {
	var members []*ast.Binding
	p.expect(token.OpenBrace)
	for {
		if p.accept(token.CloseBrace) != nil {
			break
		}
		binding := p.binding()
		if binding != nil {
			members = append(members, binding)
		}
	}
	return members
}

func (p *Parser) structExpr() *ast.StructExpr {
	p.expect(token.Struct)
	members := p.fields()
	return &ast.StructExpr{Members: members}
}

func (p *Parser) fields() []*ast.Field {
	var fields []*ast.Field
	p.expect(token.OpenParen)
	for {
		if p.accept(token.CloseParen) != nil {
			break
		}
		fields = append(fields, p.field())
		if p.accept(token.CloseParen) != nil {
			break
		}
		p.expect(token.Comma)
	}
	return fields
}

func (p *Parser) field() *ast.Field {
	name := p.identifier()
	p.expect(token.Colon)
	typ := p.expr()
	return &ast.Field{
		Name: name,
		Type: typ,
	}
}

func (p *Parser) ifExpr() *ast.IfExpr {
	p.expect(token.If)
	cond := p.expr()
	body := p.blockExpr()

	return &ast.IfExpr{Cond: cond, Block: body}
}

func (p *Parser) sliceOrManyPointer() ast.Expr {
	var manyPointer bool
	var base ast.Expr

	p.expect(token.OpenBracket)
	switch p.t.Type {
	case token.Asterisk:
		p.next()
		p.expect(token.CloseBracket)
		manyPointer = true
	case token.CloseBracket:
		p.next()
	}

	base = p.expr()

	if manyPointer {
		return &ast.ManyPointerExpr{Base: base}
	} else {
		return &ast.SliceExpr{Base: base}
	}
}

func (p *Parser) integer() *ast.Literal {
	tok := p.expect(token.Integer)
	return &ast.Literal{
		Tok:   token.Integer,
		Value: tok.Text,
	}
}

func (p *Parser) string() *ast.Literal {
	tok := p.expect(token.String)
	return &ast.Literal{
		Tok:   token.String,
		Value: tok.Text,
	}
}

func (p *Parser) identifier() *ast.Identifier {
	tok := p.expect(token.Identifier)
	if tok == nil {
		return nil
	}
	return &ast.Identifier{
		NamePos: tok.Pos,
		NameEnd: tok.End,
		Name:    tok.Text,
	}
}

func (p *Parser) expect(tok token.Type) *token.Token {
	if p.t == nil {
		panic(io.ErrUnexpectedEOF)
	}

	if t := p.accept(tok); t != nil {
		return t
	}

	p.error(NewParseError(p.t.Pos, p.t.End, fmt.Errorf("Expected %q but found %q", tok, p.t.Type)))
	p.recover(topLevelRecoveryTokens)

	return nil
}

func (p *Parser) accept(tok token.Type) *token.Token {
	if p.t == nil {
		return nil
	}

	if p.t.Type == tok {
		t := p.t
		p.next()
		return t
	}
	return nil
}

func (p *Parser) next() {
	t, err := p.scn.Scan()
	if err != nil {
		if errors.Is(err, io.EOF) {
			p.t = nil
			return
		}
		panic(err)
	}
	p.t = t
	if p.t.Type == token.Comment {
		p.next()
	}
}

func (p *Parser) unexpected(expected string) {
	var end token.Pos
	problem := p.t
	recovered := p.recover(topLevelRecoveryTokens)
	if recovered != nil {
		end = recovered.End
	}
	p.error(NewParseError(problem.Pos, end, fmt.Errorf("expected %s but found %q", expected, problem.Type)))
}

func (p *Parser) error(err error) {
	p.errs = append(p.errs, err)
}

func (p *Parser) recover(recoveryTokens []token.Type) *token.Token {
	for !slices.Contains(recoveryTokens, p.t.Type) {
		p.next()
		if p.t == nil {
			return nil
		}
	}
	t := p.t
	p.next()
	return t
}

func New(scn Scanner) *Parser {
	return &Parser{scn: scn}
}

func NewFromReader(rd io.Reader) *Parser {
	return New(scanner.New(rd))
}

func ParseBytes(name string, src []byte) (*ast.Module, error) {
	return NewFromReader(bytes.NewReader(src)).Parse(name)
}

type Scanner interface {
	Scan() (*token.Token, error)
}

type ParseError struct {
	pos, end token.Pos
	err      error
}

func NewParseError(pos, end token.Pos, err error) *ParseError {
	return &ParseError{pos, end, err}
}

func (err *ParseError) Error() string {
	return fmt.Sprintf("parse error: %v", err.err)
}

func (err *ParseError) Unwrap() error {
	return err.err
}
