package semantics

import (
	"fmt"
	"io"
	"iter"
	"maps"
	"slices"
	"strings"

	"codeberg.org/rileyq/usagi/internal/compile/token"
)

type Scope struct {
	pos, end token.Pos
	comment  string

	parent   *Scope
	module   *Module
	children []*Scope
	symbols  map[string]Symbol
}

func NewScope(parent *Scope, pos, end token.Pos, comment string) *Scope {
	s := &Scope{
		pos:      pos,
		end:      end,
		comment:  comment,
		parent:   parent,
		children: []*Scope{},
		symbols:  map[string]Symbol{},
	}
	if parent != nil {
		parent.children = append(parent.children, s)
		s.module = parent.module
	}
	return s
}

func (s *Scope) Insert(symbol Symbol) Symbol {
	name := symbol.Name()
	sym, found := s.symbols[name]
	if found {
		return sym
	}
	s.symbols[name] = symbol
	symbol.setScope(s)
	return nil
}

func (s *Scope) Lookup(name string) Symbol {
	sym, found := s.symbols[name]
	if found {
		return sym
	}
	if s.parent != nil {
		return s.parent.Lookup(name)
	}
	return nil
}

func (s *Scope) Module() *Module { return s.module }

func (s *Scope) Symbols() iter.Seq[Symbol] {
	return func(yield func(Symbol) bool) {
		for _, key := range slices.Sorted(maps.Keys(s.symbols)) {
			if !yield(s.symbols[key]) {
				return
			}
		}
	}
}

func (s *Scope) WriteTo(w io.Writer) (int64, error) {
	return s.writeTo(w, 0)
}

func (s *Scope) writeTo(w io.Writer, depth int) (int64, error) {
	const pad = "  "
	outerPad := strings.Repeat(pad, depth)
	innerPad := strings.Repeat(pad, depth+1)
	var total int64
	n, err := io.WriteString(w, outerPad)
	total += int64(n)
	if err != nil {
		return total, err
	}
	n, err = io.WriteString(w, s.comment)
	total += int64(n)
	if err != nil {
		return total, err
	}
	n, err = io.WriteString(w, " {\n")
	total += int64(n)
	if err != nil {
		return total, err
	}
	sortedNames := slices.Sorted(maps.Keys(s.symbols))
	for _, name := range sortedNames {
		n, err = io.WriteString(w, innerPad)
		total += int64(n)
		if err != nil {
			return total, err
		}
		n, err = fmt.Fprintln(w, s.symbols[name])
		total += int64(n)
		if err != nil {
			return total, err
		}
	}
	for _, child := range s.children {
		n, err := child.writeTo(w, depth+1)
		total += n
		if err != nil {
			return total, err
		}
		n2, err := io.WriteString(w, "\n")
		total += int64(n2)
		if err != nil {
			return total, err
		}
	}
	n, err = io.WriteString(w, outerPad)
	total += int64(n)
	if err != nil {
		return total, err
	}
	n, err = io.WriteString(w, "}")
	total += int64(n)
	if err != nil {
		return total, err
	}
	return total, nil
}

func (s *Scope) String() string {
	var b strings.Builder
	s.WriteTo(&b)
	return b.String()
}

type Symbol interface {
	Name() string
	Type() Type
	Value() Value
	LinkName() string
	Scope() *Scope

	setScope(scope *Scope)
}

type symbol struct {
	scope    *Scope
	name     string
	linkName string
	tv       *TypeAndValue
}

func (sym *symbol) Name() string  { return sym.name }
func (sym *symbol) Type() Type    { return sym.tv.Type() }
func (sym *symbol) Value() Value  { return sym.tv.Value() }
func (sym *symbol) Scope() *Scope { return sym.scope }

func (sym *symbol) QualifiedName() string {
	return fmt.Sprintf("%s.%s", sym.scope.Module().Name(), sym.Name())
}

func (sym *symbol) LinkName() string {
	if len(sym.linkName) > 0 {
		return sym.linkName
	}
	qualName := sym.QualifiedName()
	return fmt.Sprintf("_U%d%s", len(qualName), qualName)
}

func (sym *symbol) setScope(scope *Scope) { sym.scope = scope }

func (sym *symbol) String() string {
	var b strings.Builder
	b.WriteString(sym.Name())
	if sym.Type() != nil {
		b.WriteString(": ")
		fmt.Fprint(&b, sym.Type())
	}
	if sym.Value() != nil {
		b.WriteString(" = ")
		fmt.Fprint(&b, sym.Value())
	}
	return b.String()
}

func NewSymbol(name string, tv *TypeAndValue) *symbol {
	return &symbol{nil, name, "", tv}
}

func NewSymbolFromValue(name string, value Value) *symbol {
	return &symbol{nil, name, "", NewTypeAndValue(value.Type(), value)}
}

type TypeAndValue struct {
	typ Type
	val Value
}

func NewTypeAndValue(typ Type, val Value) *TypeAndValue { return &TypeAndValue{typ, val} }

func (tv *TypeAndValue) Type() Type   { return tv.typ }
func (tv *TypeAndValue) Value() Value { return tv.val }

var Universe *Scope

func init() {
	Universe = NewScope(nil, token.NoPos, token.NoPos, "universe")
	Universe.Insert(NewSymbolFromValue("Type", NewTypeValue(NewTraitType(true, nil))))
	Universe.Insert(NewSymbolFromValue("@import", NewBuiltin(BuiltinImport)))
	Universe.Insert(NewSymbolFromValue("@extern", NewBuiltin(BuiltinExtern)))
}
