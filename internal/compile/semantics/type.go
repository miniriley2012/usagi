package semantics

import (
	"fmt"
	"slices"
	"strconv"
	"strings"
)

type Type interface {
	IsAssignableTo(other Type) bool
	Equal(other Type) bool
}

type IntegerType struct {
	signed bool
	bits   uint16
}

func NewIntegerType(signed bool, bits int) *IntegerType {
	return &IntegerType{signed, uint16(bits)}
}

func NewIntegerTypeFromName(name string) *IntegerType {
	if name == "void" {
		return NewIntegerType(false, 0)
	}

	if len(name) < 2 {
		return nil
	}

	if name[0] != 'i' && name[0] != 'u' {
		return nil
	}

	signed := name[0] == 'i'

	for _, r := range name[1:] {
		if !(r >= '0' && r <= '9') {
			return nil
		}
	}

	bits, err := strconv.ParseUint(name[1:], 10, 16)
	if err != nil {
		panic(err)
	}

	return NewIntegerType(signed, int(bits))
}

func (i *IntegerType) Signed() bool { return i.signed }
func (i *IntegerType) Bits() int    { return int(i.bits) }

func (i *IntegerType) IsAssignableTo(other Type) bool {
	otherInt, isInteger := other.(*IntegerType)
	if !isInteger {
		return false
	}
	return otherInt.Bits() >= i.Bits()
}

func (i *IntegerType) Equal(other Type) bool {
	otherInt, isInt := other.(*IntegerType)
	if !isInt {
		return false
	}
	return i.signed == otherInt.signed && i.bits == otherInt.bits
}

func (i *IntegerType) String() string {
	if i.signed {
		return "i" + strconv.Itoa(i.Bits())
	} else {
		return "u" + strconv.Itoa(i.Bits())
	}
}

type Pointer struct {
	element Type
	many    bool
}

func NewPointer(element Type) *Pointer     { return &Pointer{element: element} }
func NewManyPointer(element Type) *Pointer { return &Pointer{element: element, many: true} }

func (p *Pointer) Element() Type { return p.element }

func (p *Pointer) IsAssignableTo(other Type) bool {
	return false
}

func (p *Pointer) Equal(other Type) bool {
	otherPointer, isPointer := other.(*Pointer)
	if !isPointer {
		return false
	}
	return p.Element().Equal(otherPointer.Element()) && p.many == otherPointer.many
}

func (p *Pointer) String() string {
	if p.many {
		return fmt.Sprintf("[*]%s", p.element)
	}
	return fmt.Sprintf("*%s", p.element)
}

type Signature struct {
	params     []*NameAndType
	returnType Type
}

func NewSignature(params []*NameAndType, returnType Type) *Signature {
	return &Signature{params, returnType}
}

func (sig *Signature) Params() []*NameAndType { return sig.params }
func (sig *Signature) ReturnType() Type       { return sig.returnType }

func (sig *Signature) String() string {
	var b strings.Builder
	b.WriteString("func(")
	for i, param := range sig.params {
		b.WriteString(param.Name())
		b.WriteString(": ")
		fmt.Fprintf(&b, "%s", param.Type())
		if i < len(sig.params)-1 {
			b.WriteString(", ")
		}
	}
	b.WriteString(") ")
	fmt.Fprintf(&b, "%s", sig.ReturnType())
	return b.String()
}

func (sig *Signature) IsAssignableTo(other Type) bool {
	return sig.Equal(other)
}

func (sig *Signature) Equal(other Type) bool {
	otherSig, isSig := other.(*Signature)
	if !isSig {
		return false
	}

	return slices.EqualFunc(sig.Params(), otherSig.Params(), func(a, b *NameAndType) bool {
		return a.Name() == b.Name() && a.Type().Equal(b.Type())
	}) && sig.ReturnType().Equal(otherSig.ReturnType())
}

type SliceType struct {
	element Type
}

func NewSliceType(element Type) *SliceType { return &SliceType{element} }

func (typ *SliceType) Element() Type { return typ.element }

func (typ *SliceType) IsAssignableTo(other Type) bool {
	return typ.Equal(other)
}

func (typ *SliceType) Equal(other Type) bool {
	otherSlice, isSlice := other.(*SliceType)
	if !isSlice {
		return false
	}
	return typ.Element().Equal(otherSlice.Element())
}

func (typ *SliceType) String() string {
	return fmt.Sprintf("[]%s", typ.Element())
}

type ExistentialType struct {
	trait Type
}

func NewExistentialType(trait Type) *ExistentialType { return &ExistentialType{trait} }

func (typ *ExistentialType) Trait() Type { return typ.trait }

func (typ *ExistentialType) IsAssignableTo(other Type) bool {
	return false // TODO
}

type TraitType struct {
	closed       bool
	requirements []*NameAndType
}

func NewTraitType(closed bool, requirements []*NameAndType) *TraitType {
	return &TraitType{closed, requirements}
}

func (typ *TraitType) Closed() bool { return typ.closed }

func (typ *TraitType) Requirements() []*NameAndType { return typ.requirements }

func (typ *TraitType) IsAssignableTo(other Type) bool {
	return false // TODO
}

func (typ *TraitType) Equal(other Type) bool {
	otherTrait, isTrait := other.(*TraitType)
	if !isTrait {
		return false
	}
	return typ.Closed() == otherTrait.Closed() && slices.EqualFunc(typ.requirements, otherTrait.requirements, func(a, b *NameAndType) bool {
		return a.Name() == b.Name() && a.Type().Equal(b.Type())
	})
}

func (typ *TraitType) String() string {
	return "trait"
}

type StructType struct {
	members []*NameAndType
}

func NewStructType(members []*NameAndType) *StructType { return &StructType{members} }

func (typ *StructType) Members() []*NameAndType { return typ.members }

func (typ *StructType) IsAssignableTo(other Type) bool {
	return false
}

func (typ *StructType) Equal(other Type) bool {
	return other == typ
}

func (typ *StructType) String() string {
	members := make([]string, 0, len(typ.members))
	for _, m := range typ.members {
		members = append(members, m.String())
	}
	return fmt.Sprintf("struct(%s)", strings.Join(members, ", "))
}

type NameAndType struct {
	name string
	typ  Type
}

func NewNameAndType(name string, typ Type) *NameAndType {
	return &NameAndType{name, typ}
}

func (nt *NameAndType) Name() string { return nt.name }
func (nt *NameAndType) Type() Type   { return nt.typ }

func (nt *NameAndType) String() string {
	return fmt.Sprintf("%s: %s", nt.Name(), nt.Type())
}
