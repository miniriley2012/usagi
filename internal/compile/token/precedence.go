package token

type Precedence int

const (
	PrecedenceNone Precedence = iota
	PrecedenceAssignment
	PrecedenceRelational
	PrecedenceAddition
	PrecedenceCall
)

func (t Type) Precedence() Precedence {
	switch t {
	case Assign:
		return PrecedenceAssignment
	case Less:
		return PrecedenceRelational
	case Minus, Plus:
		return PrecedenceAddition
	case OpenParen, Dot, OpenBracket:
		return PrecedenceCall
	default:
		return PrecedenceNone
	}
}
