package ast

import "codeberg.org/rileyq/usagi/internal/compile/token"

type Node interface {
	Pos() token.Pos
	End() token.Pos

	astNode()
}

type Expr interface {
	Node

	astExpr()
}

type Stmt interface {
	Node

	astStmt()
}

type Decl interface {
	Node

	astDecl()
}

type Module struct {
	Name  string
	Decls []Decl
}

func (m *Module) Pos() token.Pos { return token.NoPos }
func (m *Module) End() token.Pos { return token.NoPos }

func (m *Module) astNode() {}

type BindingMode int

const (
	ModeExport BindingMode = 1 << iota
	ModeConst
)

func (m BindingMode) Export() bool { return m&ModeExport != 0 }
func (m BindingMode) Const() bool  { return m&ModeConst != 0 }

type Binding struct {
	Token token.Type
	Mode  BindingMode
	Name  *Identifier
	Type  Expr
	Value Expr
}

func (b *Binding) Pos() token.Pos { return b.Name.Pos() }
func (b *Binding) End() token.Pos { panic("TODO") }

func (*Binding) astNode() {}
func (*Binding) astDecl() {}

type Identifier struct {
	NamePos token.Pos
	NameEnd token.Pos
	Name    string
}

func (id *Identifier) Pos() token.Pos { return id.NamePos }
func (id *Identifier) End() token.Pos { return id.NameEnd }

func (id *Identifier) astNode() {}
func (id *Identifier) astExpr() {}

type CallExpr struct {
	Base Expr
	Args []Expr
}

func (expr *CallExpr) Pos() token.Pos { return expr.Base.Pos() }
func (expr *CallExpr) End() token.Pos { return expr.Base.End() }

func (expr *CallExpr) astNode() {}
func (expr *CallExpr) astExpr() {}

type Literal struct {
	Tok   token.Type
	Value string
}

func (expr *Literal) Pos() token.Pos { panic("TODO") }
func (expr *Literal) End() token.Pos { panic("TODO") }

func (expr *Literal) astNode() {}
func (expr *Literal) astExpr() {}

type FuncExpr struct {
	Params     []*Param
	ReturnType Expr
	Body       *BlockExpr
}

func (expr *FuncExpr) Pos() token.Pos { panic("TODO") }
func (expr *FuncExpr) End() token.Pos { panic("TODO") }

func (expr *FuncExpr) astNode() {}
func (expr *FuncExpr) astExpr() {}

type Param struct {
	Name *Identifier
	Type Expr
}

func (p *Param) Pos() token.Pos { panic("TODO") }
func (p *Param) End() token.Pos { panic("TODO") }

func (p *Param) astNode() {}

type BlockExpr struct {
	List []Stmt
}

func (expr *BlockExpr) Pos() token.Pos { panic("TODO") }
func (expr *BlockExpr) End() token.Pos { panic("TODO") }

func (*BlockExpr) astNode() {}
func (*BlockExpr) astExpr() {}

type ReturnExpr struct {
	Value Expr
}

func (expr *ReturnExpr) Pos() token.Pos { panic("TODO") }
func (expr *ReturnExpr) End() token.Pos { panic("TODO") }

func (*ReturnExpr) astNode() {}
func (*ReturnExpr) astExpr() {}

type BinaryExpr struct {
	Left  Expr
	Op    token.Type
	Right Expr
}

func (expr *BinaryExpr) Pos() token.Pos { return expr.Left.Pos() }
func (expr *BinaryExpr) End() token.Pos { return expr.Right.End() }

func (*BinaryExpr) astNode() {}
func (*BinaryExpr) astExpr() {}

type UnaryExpr struct {
	Op   token.Type
	Base Expr
}

func (*UnaryExpr) Pos() token.Pos { panic("TODO") }
func (*UnaryExpr) End() token.Pos { panic("TODO") }

func (*UnaryExpr) astNode() {}
func (*UnaryExpr) astExpr() {}

type ExprStmt struct{ X Expr }

func (stmt *ExprStmt) Pos() token.Pos { return stmt.X.Pos() }
func (stmt *ExprStmt) End() token.Pos { return stmt.X.End() }

func (*ExprStmt) astNode() {}
func (*ExprStmt) astStmt() {}

type MemberExpr struct {
	Base   Expr
	Member *Identifier
}

func (expr *MemberExpr) Pos() token.Pos { return expr.Base.Pos() }
func (expr *MemberExpr) End() token.Pos { return expr.Member.End() }

func (*MemberExpr) astNode() {}
func (*MemberExpr) astExpr() {}

type SliceExpr struct {
	Base Expr
}

func (expr *SliceExpr) Pos() token.Pos { panic("TODO") }
func (expr *SliceExpr) End() token.Pos { return expr.Base.End() }

func (*SliceExpr) astNode() {}
func (*SliceExpr) astExpr() {}

type ManyPointerExpr struct {
	Base Expr
}

func (expr *ManyPointerExpr) Pos() token.Pos { panic("TODO") }
func (expr *ManyPointerExpr) End() token.Pos { return expr.Base.End() }

func (*ManyPointerExpr) astNode() {}
func (*ManyPointerExpr) astExpr() {}

type VarArgExpr struct{}

func (*VarArgExpr) Pos() token.Pos { panic("TODO") }
func (*VarArgExpr) End() token.Pos { panic("TODO") }

func (*VarArgExpr) astNode() {}
func (*VarArgExpr) astExpr() {}

type IfExpr struct {
	Cond  Expr
	Block *BlockExpr
}

func (*IfExpr) Pos() token.Pos { panic("TODO") }
func (*IfExpr) End() token.Pos { panic("TODO") }

func (*IfExpr) astNode() {}
func (*IfExpr) astExpr() {}

type BadExpr struct{}

func (*BadExpr) Pos() token.Pos { panic("TODO") }
func (*BadExpr) End() token.Pos { panic("TODO") }

func (*BadExpr) astNode() {}
func (*BadExpr) astExpr() {}

type StructExpr struct {
	Members []*Field
}

func (*StructExpr) Pos() token.Pos { panic("TODO") }
func (*StructExpr) End() token.Pos { panic("TODO") }

func (*StructExpr) astNode() {}
func (*StructExpr) astExpr() {}

type Field struct {
	Name *Identifier
	Type Expr
}

func (*Field) Pos() token.Pos { panic("TODO") }
func (*Field) End() token.Pos { panic("TODO") }

func (*Field) astNode() {}

type NamedArg struct {
	Name  *Identifier
	Value Expr
}

func (*NamedArg) Pos() token.Pos { panic("TODO") }
func (*NamedArg) End() token.Pos { panic("TODO") }

func (*NamedArg) astNode() {}
func (*NamedArg) astExpr() {}

type TraitExpr struct {
	Traits  []Expr
	Members []*Binding
}

func (*TraitExpr) Pos() token.Pos { panic("TODO") }
func (*TraitExpr) End() token.Pos { panic("TODO") }

func (*TraitExpr) astNode() {}
func (*TraitExpr) astExpr() {}

type ImplDecl struct {
	Type        Expr
	Traits      []Expr
	Definitions []*Binding
}

func (*ImplDecl) Pos() token.Pos { panic("TODO") }
func (*ImplDecl) End() token.Pos { panic("TODO") }

func (*ImplDecl) astNode() {}
func (*ImplDecl) astDecl() {}

type ExistentialExpr struct {
	Base Expr
}

func (*ExistentialExpr) Pos() token.Pos { panic("TODO") }
func (*ExistentialExpr) End() token.Pos { panic("TODO") }

func (*ExistentialExpr) astNode() {}
func (*ExistentialExpr) astExpr() {}

type DeclStmt struct {
	X Decl
}

func (stmt *DeclStmt) Pos() token.Pos { return stmt.X.Pos() }
func (stmt *DeclStmt) End() token.Pos { return stmt.X.End() }

func (*DeclStmt) astNode() {}
func (*DeclStmt) astStmt() {}
