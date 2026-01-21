package qbe

import (
	"io"
)

type Module struct {
	Definitions []Definition
}

func NewModule() *Module {
	return &Module{}
}

func (m *Module) Add(def Definition) {
	m.Definitions = append(m.Definitions, def)
}

type Definition interface {
	io.WriterTo
}

type RegularType struct {
	Name    string
	Members []Type
}

type Data struct {
	Linkage Linkage
	Name    Value
	Items   []DataItem
}

type DataItem interface {
	io.WriterTo
}

type StringLiteral string

type Function struct {
	Linkage    Linkage
	ReturnType Type
	Name       string
	Params     []*Param
	Blocks     []*Block
}

func NewFunction(linkage Linkage, returnType Type, name string) *Function {
	return &Function{
		Linkage:    linkage,
		ReturnType: returnType,
		Name:       name,
	}
}

func (f *Function) Add(block *Block) {
	f.Blocks = append(f.Blocks, block)
}

func (f *Function) Type() Type { return Long }

type Block struct {
	Name         string
	Instructions []Instruction
}

func NewBlock(name string) *Block {
	return &Block{Name: name}
}

func (b *Block) Add(inst Instruction) {
	b.Instructions = append(b.Instructions, inst)
}

type Instruction interface {
	io.WriterTo
}

type Linkage int

const (
	LinkageExport Linkage = 1 << iota
	LinkageThread
)

func (l Linkage) Export() bool { return l&LinkageExport != 0 }
func (l Linkage) Thread() bool { return l&LinkageThread != 0 }

type Type interface {
	io.WriterTo
}

type Param struct {
	Type  Type
	Ident *Temporary
}

type Temporary struct {
	name string
	typ  Type
}

func NewTemporary(name string, typ Type) *Temporary { return &Temporary{name, typ} }

func (t *Temporary) Type() Type { return t.typ }

type Value interface {
	io.WriterTo

	Type() Type
}

type SimpleType string

const (
	Void   SimpleType = ""
	Word   SimpleType = "w"
	Long   SimpleType = "l"
	Single SimpleType = "s"
	Double SimpleType = "d"
)

type Global struct {
	name string
	typ  Type
}

func NewGlobal(name string, typ Type) *Global { return &Global{name, typ} }

func (g *Global) Type() Type { return g.typ }

type Constant int64

func (c Constant) Type() Type { return Long }
