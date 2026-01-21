package qbe

import "strconv"

type ModuleBuilder struct {
	strings     int
	definitions []Definition
}

func (b *ModuleBuilder) StringLiteral(value string) *Global {
	name := NewGlobal("str"+strconv.Itoa(b.strings), Long)
	b.strings++
	// b.strings = append(b.strings, value)
	b.Add(&Data{
		Name:  name,
		Items: []DataItem{StringLiteral(value)},
	})
	return name
}

func (b *ModuleBuilder) Add(def Definition) {
	b.definitions = append(b.definitions, def)
}

func (b *ModuleBuilder) Module() *Module {
	return &Module{Definitions: b.definitions}
}

type FunctionBuilder struct {
	temporaries int
	linkage     Linkage
	returnType  Type
	name        string
	params      []*Param
	blocks      []*Block
}

func (b *FunctionBuilder) Returns(typ Type) {
	b.returnType = typ
}

func (b *FunctionBuilder) Name(name string) {
	b.name = name
}

func (b *FunctionBuilder) Param(name string, typ Type) *Temporary {
	temp := &Temporary{name: name}
	b.params = append(b.params, &Param{
		Type:  typ,
		Ident: temp,
	})
	return temp
}

func (b *FunctionBuilder) Temporary(typ Type) *Temporary {
	temp := NewTemporary("t"+strconv.Itoa(b.temporaries), typ)
	b.temporaries++
	return temp
}

func (b *FunctionBuilder) Add(block *Block) { b.blocks = append(b.blocks, block) }

func (b *FunctionBuilder) Function() *Function {
	return &Function{
		Linkage:    b.linkage,
		ReturnType: b.returnType,
		Name:       b.name,
		Params:     b.params,
		Blocks:     b.blocks,
	}
}

type BlockBuilder struct {
	name         string
	instructions []Instruction
}

func (b *BlockBuilder) Name(name string) { b.name = name }

func (b *BlockBuilder) Add(inst Instruction) { b.instructions = append(b.instructions, inst) }

func (b *BlockBuilder) Block() *Block { return &Block{b.name, b.instructions} }
