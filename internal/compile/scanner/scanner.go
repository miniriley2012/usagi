package scanner

import (
	"bufio"
	"errors"
	"io"
	"unicode"

	"codeberg.org/rileyq/usagi/internal/compile/token"
)

type Scanner struct {
	rd *runeScanner
}

func New(rd io.Reader) *Scanner {
	return &Scanner{rd: newRuneScanner(bufio.NewReader(rd))}
}

func (s *Scanner) Scan() (*token.Token, error) {
	err := s.skipSpace()
	if err != nil {
		return nil, err
	}

	s.rd.Begin()

	r, err := s.next()
	if err != nil {
		return nil, err
	}

	if isIdentifierStart(r) {
		return s.identifier()
	} else if r == '"' {
		return s.string()
	} else if isDigit(r) {
		return s.integer()
	} else if r == '/' {
		r, err = s.next()
		if err != nil {
			return nil, err
		}
		if r == '/' {
			for r != '\n' {
				r, err = s.next()
				if err != nil {
					return nil, err
				}
			}
			return s.token(token.Comment), nil
		}
		panic("TODO")
	}

	node := token.Fixed
retry:
	for _, c := range node.Children {
		if c.Rune == r {
			node = c
			r, err = s.next()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				panic(err)
			}
			goto retry
		}
	}
	if err == nil && node.Type != token.Invalid {
		s.rewind()
	}

	return s.token(node.Type), nil
}

func (s *Scanner) integer() (*token.Token, error) {
	var r rune
	var err error

	for {
		r, err = s.next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if !isDigit(r) {
			break
		}
	}
	if err == nil {
		s.rewind()
	}

	return s.token(token.Integer), nil
}

func (s *Scanner) string() (*token.Token, error) {
	var r rune
	var err error

	for {
		r, err = s.next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = io.ErrUnexpectedEOF
			}
			return nil, err
		}
		if r == '"' {
			break
		}
	}

	return s.token(token.String), nil
}

func (s *Scanner) identifier() (*token.Token, error) {
	var r rune
	var err error

	for {
		r, err = s.next()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		if !isIdentifierContinue(r) {
			break
		}
	}
	if err == nil {
		s.rewind()
	}

	start, end, text := s.rd.End()

	typ := token.Identifier
	node := token.Fixed
outer:
	for _, r := range text {
		for _, c := range node.Children {
			if c.Rune == r {
				node = c
				continue outer
			}
		}
		node = nil
		break
	}
	if node != nil && node.Type != token.Invalid {
		typ = node.Type
	}

	return &token.Token{
		Type: typ,
		Pos:  token.Pos(start + 1),
		End:  token.Pos(end + 1),
		Text: text,
	}, nil
}

func (s *Scanner) token(tok token.Type) *token.Token {
	start, end, text := s.rd.End()
	return &token.Token{
		Type: tok,
		Pos:  token.Pos(start + 1),
		End:  token.Pos(end + 1),
		Text: text,
	}
}

func (s *Scanner) skipSpace() error {
	for {
		r, err := s.next()
		if err != nil {
			return err
		}
		if !unicode.IsSpace(r) {
			s.rewind()
			return nil
		}
	}
}

func (s *Scanner) next() (rune, error) {
	r, _, err := s.rd.ReadRune()
	if err != nil {
		return r, err
	}
	return r, nil
}

func (s *Scanner) rewind() {
	err := s.rd.UnreadRune()
	if err != nil {
		panic(err)
	}
}

func isIdentifierStart(r rune) bool {
	return r == '@' || r == '_' || unicode.In(r, &ID_Start)
}

func isIdentifierContinue(r rune) bool {
	return r == '_' || unicode.In(r, &ID_Continue)
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

type runeScanner struct {
	rd           io.RuneScanner
	off          int
	lastRuneSize int
	recording    bool
	buf          []rune
	start        int
}

func newRuneScanner(rd io.RuneScanner) *runeScanner {
	return &runeScanner{rd: rd, lastRuneSize: -1}
}

func (r *runeScanner) Begin() {
	r.recording = true
	r.buf = r.buf[:0]
	r.start = r.off
}

func (r *runeScanner) End() (int, int, string) {
	r.recording = false
	return r.start, r.off, string(r.buf)
}

func (r *runeScanner) ReadRune() (rune, int, error) {
	ru, sz, err := r.rd.ReadRune()
	if err != nil {
		return ru, sz, err
	}
	r.off += sz
	r.lastRuneSize = sz
	if r.recording {
		r.buf = append(r.buf, ru)
	}
	return ru, sz, err
}

func (r *runeScanner) UnreadRune() error {
	if r.lastRuneSize == -1 {
		return errors.New("invalid use of UnreadRune")
	}
	err := r.rd.UnreadRune()
	if err != nil {
		return err
	}
	r.off -= r.lastRuneSize
	r.lastRuneSize = -1
	if r.recording {
		r.buf = r.buf[:len(r.buf)-1]
	}
	return err
}
