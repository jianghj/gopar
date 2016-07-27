package parsec

import (
	"bytes"
	"fmt"
)

type Parser func(*ParseState) (interface{}, error)

var Lowercase = OneOf([]byte("abcdefghijklmnopqrstuvwxyz"))
var Uppercase = OneOf([]byte("ABCDEFGHIJKLMNOPQRSTUVWXYZ"))
var Letter = Either(Lowercase, Uppercase)
var Letters = Many1(Letter)
var Digit = OneOf([]byte("0123456789"))
var Digits = Many1(Digit)
var AlphaNum = Either(Letter, Digit)
var AlphaNums = Many1(AlphaNum)
var HexDigit = OneOf([]byte("0123456789acdefABCDEF"))
var HexDigits = Many1(HexDigit)
var Punctuation = OneOf([]byte("!@#$%^&*()-=+[]{}\\|;:'\",./<>?~`"))
var Space = OneOf([]byte(" \t"))
var Spaces = Skip(Space)
var Newline = OneOf([]byte("\r\n"))
var Eol = Either(Eof, Newline)

type ParseState struct {
	Source string
	Pos    int
	Line   int
}

type ParseErr struct {
	Reason string
	Line   int
}

func (err ParseErr) Error() string {
	return fmt.Sprintf("%s on line %d", err.Reason, err.Line)
}

func (p Parser) Parse(source string) (interface{}, error) {
	st := ParseState{Source: source, Line: 1, Pos: 0}
	return p(&st)
}

func (st *ParseState) next(pred func(byte) bool) (byte, bool) {
	if st.Pos < len(st.Source) {
		if c := st.Source[st.Pos]; pred(c) == false {
			return c, false
		} else {
			st.Pos++
			if c == '\n' {
				st.Line++
			}
			return c, true
		}
	}
	return '\000', false
}

func (st *ParseState) trap(format string, args ...interface{}) ParseErr {
	return ParseErr{Line: st.Line, Reason: fmt.Sprintf(format, args...)}
}

func (p Parser) Bind(f func(interface{}) Parser) Parser {
	return func(st *ParseState) (interface{}, error) {
		if x, err := p(st); err != nil {
			return nil, err
		} else {
			return f(x)(st)
		}
	}
}

func (p1 Parser) Then(p2 Parser) Parser {
	return func(st *ParseState) (interface{}, error) {
		if _, err := p1(st); err != nil {
			return nil, err
		}
		return p2(st)
	}
}

func Return(x interface{}) Parser {
	return func(st *ParseState) (interface{}, error) {
		return x, nil
	}
}

func Fail(msg string) Parser {
	return func(st *ParseState) (interface{}, error) {
		return nil, st.trap(msg)
	}
}

func Either(p1, p2 Parser) Parser {
	return func(st *ParseState) (interface{}, error) {
		oldPos := st.Pos
		x, err := p1(st)
		if err == nil {
			return x, nil
		}
		if st.Pos == oldPos {
			return p2(st)
		}
		return nil, err
	}
}

func (p Parser) Or(p2 Parser) Parser {
	return Either(p, p2)
}

func Try(p Parser) Parser {
	return func(st *ParseState) (interface{}, error) {
		oldPos := st.Pos
		if x, err := p(st); err == nil {
			return x, nil
		} else {
			st.Pos = oldPos
			return nil, err
		}
	}
}

func AnyChar(st *ParseState) (interface{}, error) {
	if c, ok := st.next(func(x byte) bool { return true }); ok {
		return c, nil
	}
	return nil, st.trap("Unexpected end of file")
}

func Eof(st *ParseState) (interface{}, error) {
	if c, ok := st.next(func(x byte) bool { return true }); ok {
		return nil, st.trap("Expected end of file but got '%c'", c)
	}
	return nil, nil
}

func Char(c byte) Parser {
	return func(st *ParseState) (interface{}, error) {
		if x, ok := st.next(func(b byte) bool { return b == c }); ok {
			return x, nil
		} else {
			return nil, st.trap("Expected '%c'", c)
		}
	}
}

func OneOf(set []byte) Parser {
	return func(st *ParseState) (interface{}, error) {
		if x, ok := st.next(func(c byte) bool { return bytes.IndexByte(set, c) >= 0 }); ok {
			return x, nil
		} else {
			return nil, st.trap("Expected one of '%s' but got '%c'", string(set), x)
		}
	}
}

func NoneOf(set []byte) Parser {
	return func(st *ParseState) (interface{}, error) {
		if x, ok := st.next(func(c byte) bool { return bytes.IndexByte(set, c) < 0 }); ok {
			return x, nil
		} else {
			return nil, st.trap("Unexpected '%c'", x)
		}
	}
}

func String(s string) Parser {
	return func(st *ParseState) (interface{}, error) {
		oldPos := st.Pos

		for _, c := range []byte(s) {
			_, ok := st.next(func(b byte) bool { return b == c })

			if ok == false {
				st.Pos = oldPos
				return nil, st.trap("Expected '%s'", s)
			}
		}
		return s, nil
	}
}

func (p Parser) ToString() Parser {
	return p.Bind(func(x interface{}) Parser {
		var bs []byte = make([]byte, len(x.([]interface{})))
		for i, c := range x.([]interface{}) {
			bs[i] = c.(byte)
		}
		return Return(string(bs))
	})
}

func appendx(x, xs interface{}) interface{} {
	return append([]interface{}{x}, xs.([]interface{})...)
}

func Many1(p Parser) Parser {
	return p.Bind(func(x interface{}) Parser {
		return Many(p).Bind(func(xs interface{}) Parser {
			return Return(appendx(x, xs))
		})
	})
}

func Many(p Parser) Parser {
	return Many1(p).Or(Return([]interface{}{}))
}

func ManyTill(p, end Parser) Parser {
	return Either(Try(end).Then(Return([]interface{}{})),
		p.Bind(func(x interface{}) Parser {
			return ManyTill(p, end).Bind(func(xs interface{}) Parser {
				return Return(appendx(x, xs))
			})
		}))
}

func Skip(p Parser) Parser {
	return p.Then(Return(nil)).Or(Return(nil))
}

func SkipMany(p Parser) Parser {
	return Skip(Many(p))
}

func (p Parser) Between(start, end Parser) Parser {
	keep := func(x interface{}) Parser { return end.Then(Return(x)) }
	return start.Then(p.Bind(keep))
}

func (p Parser) SepBy1(sep Parser) Parser {
	return p.Bind(func(x interface{}) Parser {
		return Many(sep.Then(p)).Bind(func(xs interface{}) Parser {
			return Return(appendx(x, xs))
		})
	})
}

func (p Parser) SepBy(sep Parser) Parser {
	return p.SepBy1(sep).Or(Return([]interface{}{}))
}
