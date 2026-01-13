package token

type Pos int

const NoPos Pos = 0

type Token struct {
	Type Type
	Pos  Pos
	End  Pos
	Text string
}
