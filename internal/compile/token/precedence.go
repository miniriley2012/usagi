package token

type Precedence int

const (
	PrecedenceNone Precedence = iota
	PrecedenceAddition
	PrecedenceCall
)

func (t Type) Precedence() Precedence {
	switch t {
	case Plus:
		return PrecedenceAddition
	case OpenParen, Dot:
		return PrecedenceCall
	default:
		return PrecedenceNone
	}
}
