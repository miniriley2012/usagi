package token

type Pos int

type Token struct {
	Type Type
	Pos  Pos
	End  Pos
	Text string
}
