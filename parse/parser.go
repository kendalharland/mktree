package parse

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"runtime/debug"
	"strings"
)

//
// Grammar
//

type TokenKind string

const (
	AttributeTokenKind TokenKind = "Attribute"
	CommentTokenKind   TokenKind = "Comment"
	NumberTokenKind    TokenKind = "Number"
	StringTokenKind    TokenKind = "String"
	LParenTokenKind    TokenKind = "LParen"
	RParenTokenKind    TokenKind = "RParen"
	NewlineTokenKind   TokenKind = "Newline"
	EofTokenKind       TokenKind = "EOF"
	ErrTokenKind       TokenKind = "Error"

	// Keywords
	DirTokenKind  TokenKind = "dir"
	FileTokenKind TokenKind = "file"
	LinkTokenKind TokenKind = "link"
)

var keywords = map[string]TokenKind{
	"dir":  DirTokenKind,
	"file": FileTokenKind,
	"link": LinkTokenKind,
}

type Token struct {
	Kind  TokenKind
	Value string
	Pos   int
}

func (t Token) String() string {
	return fmt.Sprintf("(%v, %q, %d)", t.Kind, t.Value, t.Pos)
}

type Tree struct {
	SExprs []*SExpr
}

type SExpr struct {
	Literal *Literal
	Args    []*Arg
}

type Literal struct {
	Token *Token
}

type Arg struct {
	// The first token for this Arg.
	//
	// This matches the token set on either SExpr or Literal.
	Token *Token

	// Only one of these is set.
	SExpr   *SExpr
	Literal *Literal
}

//
// Parser
//

type Parser struct {
	s []byte          // Original source.
	r *bufio.Reader   // Buffered source reader.
	p int             // Source position.
	t *Token          // Current token.
	b strings.Builder // Current token source buffer.
	e error           // Most recent error.

	Stderr io.Writer
}

func Parse(r io.Reader) (*Tree, error) {
	return (&Parser{}).Parse(r)
}

func (p *Parser) init(r io.Reader) error {
	src, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	if p.Stderr == nil {
		p.Stderr = os.Stderr
	}
	p.s = src
	p.r = bufio.NewReader(bytes.NewReader(p.s))
	makeToken(p, EofTokenKind)
	return nil
}

func (p *Parser) Parse(r io.Reader) (*Tree, error) {
	if err := p.init(r); err != nil {
		return nil, err
	}
	c := parseConfig(p)
	return c, p.e
}

func parseConfig(p *Parser) *Tree {
	c := &Tree{}
	nextToken(p)
	for !match(p, EofTokenKind) {
		ignore(p, CommentTokenKind, NewlineTokenKind)
		c.SExprs = append(c.SExprs, parseSExpr(p))
		ignore(p, CommentTokenKind, NewlineTokenKind)
	}
	return c
}

func parseSExpr(p *Parser) *SExpr {
	consume(p, LParenTokenKind)
	ignore(p, NewlineTokenKind)

	literal := parseLiteral(p)
	ignore(p, NewlineTokenKind)

	var args []*Arg
	for !(match(p, EofTokenKind) || match(p, RParenTokenKind)) {
		args = append(args, parseArg(p))
		ignore(p, NewlineTokenKind)
	}

	consume(p, RParenTokenKind)
	return &SExpr{Literal: literal, Args: args}
}

func parseArg(p *Parser) *Arg {
	if match(p, LParenTokenKind) {
		t := peekToken(p)
		e := parseSExpr(p)
		return &Arg{Token: t, SExpr: e}
	}
	l := parseLiteral(p)
	return &Arg{Token: l.Token, Literal: l}
}

func parseLiteral(p *Parser) *Literal {
	t := peekToken(p)
	switch t.Kind {
	case DirTokenKind, FileTokenKind, LinkTokenKind, AttributeTokenKind, StringTokenKind, NumberTokenKind:
		nextToken(p)
		return &Literal{Token: t}
	}
	emitUnexpectedTokenError(p)
	return &Literal{Token: t}
}

func consume(p *Parser, k TokenKind) {
	if match(p, k) {
		nextToken(p)
		return
	}
	emitUnexpectedTokenError(p)
}

func ignore(p *Parser, kinds ...TokenKind) {
OuterLoop:
	for _, k := range kinds {
		if match(p, k) {
			nextToken(p)
			goto OuterLoop
		}
	}
}

//
// Lexer
//

func peekToken(p *Parser) *Token {
	return p.t
}

func match(p *Parser, k TokenKind) bool {
	return peekToken(p).Kind == k
}

func makeToken(p *Parser, k TokenKind) {
	p.t = &Token{
		Kind:  k,
		Value: p.b.String(),
		Pos:   p.p - p.b.Len(),
	}
	p.b.Reset()
}

func nextToken(p *Parser) {
	for {
		if isEOF(p.r) {
			makeToken(p, EofTokenKind)
			return
		}
		switch peekChar(p.r) {
		case '(':
			nextChar(p)
			makeToken(p, LParenTokenKind)
			return
		case ')':
			nextChar(p)
			makeToken(p, RParenTokenKind)
			return
		case '@':
			readAttribute(p)
			return
		case '"':
			readString(p)
			return
		case '\n':
			nextChar(p)
			makeToken(p, NewlineTokenKind)
			return
		case ' ', '\t', '\r':
			skipChar(p)
			continue
		case ';':
			readComment(p)
			return
		}
		if isDigit(peekChar(p.r)) {
			readNumber(p)
			return
		}
		if isAlpha(peekChar(p.r)) {
			readKeyword(p)
			return
		}
		emitSyntaxError(p, "invalid character %s", string(peekChar(p.r)))
		skipChar(p)
		return
	}
}

func readAttribute(p *Parser) {
	nextChar(p) // @
	readWhile(p, isAlpha)
	makeToken(p, AttributeTokenKind)
}

func readComment(p *Parser) {
	readUntil(p, '\n')
	makeToken(p, CommentTokenKind)
}

// TODO: Handle escaped quotes.
func readString(p *Parser) {
	skipChar(p) // "
	readUntil(p, '"')
	makeToken(p, StringTokenKind)
	skipChar(p) // "
}

func readNumber(p *Parser) {
	readWhile(p, isDigit)
	makeToken(p, NumberTokenKind)
}

func readKeyword(p *Parser) {
	nextChar(p)
	readWhile(p, isAlpha)

	source := p.b.String()
	if kind, ok := keywords[source]; ok {
		makeToken(p, kind)
		return
	}

	emitSyntaxError(p, "invalid keyword: %q", source)
}

func readWhile(p *Parser, test interface{}) {
	match := matcher(test)
	for !isEOF(p.r) && match(peekChar(p.r)) {
		nextChar(p)
	}
}

func readUntil(p *Parser, test interface{}) {
	match := matcher(test)
	for !isEOF(p.r) && !match(peekChar(p.r)) {
		nextChar(p)
	}
}

func matcher(test interface{}) func(byte) bool {
	switch t := test.(type) {
	case rune:
		return func(b byte) bool { return b == byte(t) }
	case byte:
		return func(b byte) bool { return b == t }
	case func(byte) bool:
		return t
	default:
		panic("readUntil: test must be byte or func(byte)bool")
	}
}

func nextChar(p *Parser) {
	p.b.WriteByte(skipChar(p))
}

func skipChar(p *Parser) byte {
	if isEOF(p.r) {
		panic("skipChar: EOF")
	}

	p.p += 1
	b, _ := p.r.ReadByte()
	return b
}

func peekChar(r *bufio.Reader) byte {
	if isEOF(r) {
		panic("peekChar: EOF")
	}
	b, _ := r.Peek(1)
	return b[0]
}

func isEOF(r *bufio.Reader) bool {
	_, err := r.Peek(1)
	return errors.Is(err, io.EOF)
}

func isAlpha(b byte) bool {
	return 'a' <= b && b < 'z' || 'A' <= b && b <= 'Z' || isDigit(b)
}

func isDigit(b byte) bool {
	return '0' <= b && b <= '9'
}

//
// Errors
//

func emitError(p *Parser, e error) {
	err := addErrorContext(p, e)
	if p.e == nil {
		p.e = err
	}
	fmt.Fprintln(p.Stderr, err)
	fmt.Fprintln(p.Stderr, string(debug.Stack()))
}

func emitSyntaxError(p *Parser, format string, args ...interface{}) {
	makeToken(p, ErrTokenKind)
	emitError(p, Errorf(ErrSyntax, format, args...))
}

func emitParseError(p *Parser, format string, args ...interface{}) {
	if match(p, ErrTokenKind) { // Already reported as syntax error.
		return
	}
	emitError(p, Errorf(ErrParse, format, args...))
}

func emitUnexpectedTokenError(p *Parser) {
	if p.t.Kind == EofTokenKind {
		emitParseError(p, "unexpected end of input")
	} else {
		emitParseError(p, "unexpected token %v", p.t)
	}
	nextToken(p)
}

func addErrorContext(p *Parser, e error) error {
	pos := p.t.Pos
	before, after := p.s[:pos], p.s[pos:]
	lineStart := bytes.LastIndexByte(before, '\n')
	if lineStart < 0 {
		lineStart = 0
	}
	lineEnd := len(before) + bytes.IndexByte(after, '\n')
	if lineEnd >= len(p.s) {
		lineEnd = len(p.s) - 1
	}

	line := bytes.Count(before, []byte{'\n'}) + 1
	col := pos - lineStart
	b := &bytes.Buffer{}
	b.Write(p.s[lineStart:lineEnd])
	b.WriteByte('\n')
	b.WriteString(strings.Repeat("-", col-1) + "^")

	return fmt.Errorf("%w at line %d col %d:\n%s", e, line, col, b.String())
}
