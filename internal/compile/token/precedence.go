package token

type Precedence int

const (
	PrecedenceNone Precedence = iota
	PrecedenceRelational
	PrecedenceAddition
	PrecedenceCall
)

func (t Type) Precedence() Precedence {
	switch t {
	case Less:
		return PrecedenceRelational
	case Minus, Plus:
		return PrecedenceAddition
	case OpenParen, Dot:
		return PrecedenceCall
	default:
		return PrecedenceNone
	}
}
