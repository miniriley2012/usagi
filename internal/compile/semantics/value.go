package semantics

import "fmt"

type Value interface {
	Type() Type
}

type TypeValue struct{ typ Type }

func NewTypeValue(typ Type) *TypeValue { return &TypeValue{typ} }

func (value *TypeValue) Type() Type { return value.typ }

func (value *TypeValue) String() string { return fmt.Sprintf("type %s", value.Type()) }

type BuiltinID int

const (
	_ BuiltinID = iota
	BuiltinImport
	BuiltinExtern
)

func (id BuiltinID) String() string {
	switch id {
	case BuiltinExtern:
		return "@extern"
	case BuiltinImport:
		return "@import"
	default:
		panic(fmt.Sprintf("unexpected semantics.BuiltinID: %#v", id))
	}
}

type Builtin struct {
	id BuiltinID
}

func NewBuiltin(id BuiltinID) *Builtin { return &Builtin{id} }

func (value *Builtin) Type() Type {
	typeTrait := Universe.Lookup("Type").Type()
	stringLiteral := NewSliceType(NewIntegerType(false, 8))
	switch value.id {
	case BuiltinImport:
		return NewSignature([]*NameAndType{NewNameAndType("name", stringLiteral)}, typeTrait)
	case BuiltinExtern:
		return NewSignature([]*NameAndType{NewNameAndType("linkName", stringLiteral)}, typeTrait)
	default:
		panic("unimplemented builtin")
	}
}

func (value *Builtin) String() string {
	return value.id.String()
}

type StringLiteral struct {
	value string
}

func NewStringLiteral(value string) *StringLiteral { return &StringLiteral{value} }

func (s *StringLiteral) Value() string { return s.value }

func (s *StringLiteral) Type() Type {
	return NewSliceType(NewIntegerType(false, 8))
}

type ExternalSymbol struct{ nt *NameAndType }

func NewExternalSymbol(name string, typ Type) *ExternalSymbol {
	return &ExternalSymbol{NewNameAndType(name, typ)}
}

func (e *ExternalSymbol) Name() string { return e.nt.Name() }
func (e *ExternalSymbol) Type() Type   { return e.nt.Type() }

func (e *ExternalSymbol) String() string {
	return fmt.Sprintf("@extern(%q)", e.Name())
}

type ModuleImport struct {
	module *Module
}

func NewModuleImport(module *Module) *ModuleImport { return &ModuleImport{module} }

func (value *ModuleImport) Module() *Module { return value.Module() }

func (value *ModuleImport) Type() Type {
	return NewStructType(nil)
}
