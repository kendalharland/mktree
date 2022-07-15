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
)

var keywords = map[string]TokenKind{
	"dir":  DirTokenKind,
	"file": FileTokenKind,
}

var EOF = &Token{EofTokenKind, "", -1}

type Token struct {
	Kind  TokenKind
	Value string
	Pos   int
}

func (t Token) String() string {
	return fmt.Sprintf("(%v, %q, %d)", t.Kind, t.Value, t.Pos)
}

type Config struct {
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

func Parse(r io.Reader) (*Config, error) {
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

func (p *Parser) Parse(r io.Reader) (*Config, error) {
	if err := p.init(r); err != nil {
		return nil, err
	}
	c := parseConfig(p)
	return c, p.e
}

func parseConfig(p *Parser) *Config {
	c := &Config{}
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
	case DirTokenKind, FileTokenKind, AttributeTokenKind, StringTokenKind, NumberTokenKind:
		nextToken(p)
		return &Literal{Token: t}
	}
	unexpectedTokenErr(p)
	return &Literal{Token: t}
}

func consume(p *Parser, k TokenKind) {
	if match(p, k) {
		nextToken(p)
		return
	}
	unexpectedTokenErr(p)
}

func ignore(p *Parser, kinds ...TokenKind) {
OuterLoop:
	for {
		for _, k := range kinds {
			if match(p, k) {
				nextToken(p)
				continue OuterLoop
			}
		}
		return
	}
}

func makeToken(p *Parser, k TokenKind) {
	p.t = &Token{
		Kind:  k,
		Value: p.b.String(),
		Pos:   p.p - p.b.Len(),
	}
	p.b.Reset()
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
		readKeyword(p)
		return
	}
}

func readAttribute(p *Parser) {
	nextChar(p) // @
	for !isEOF(p.r) {
		c := peekChar(p.r)
		if isWhiteSpace(c) || c == '\n' {
			break
		}
		nextChar(p)
	}
	makeToken(p, AttributeTokenKind)
}

func readComment(p *Parser) {
	for !isEOF(p.r) && peekChar(p.r) != '\n' {
		nextChar(p)
	}
	makeToken(p, CommentTokenKind)
}

// TODO: Handle escaped quotes.
func readString(p *Parser) {
	skipChar(p) // "
	for !(isEOF(p.r) || peekChar(p.r) == '"') {
		nextChar(p)
	}
	makeToken(p, StringTokenKind)
	skipChar(p) // "
}

func readNumber(p *Parser) {
	for !isEOF(p.r) && isDigit(peekChar(p.r)) {
		nextChar(p)
	}
	makeToken(p, NumberTokenKind)
}

func readKeyword(p *Parser) {
	nextChar(p)
	for isAlpha(peekChar(p.r)) {
		nextChar(p)
	}

	source := buffer(p)
	if kind, ok := keywords[source]; ok {
		makeToken(p, kind)
		return
	}

	syntaxErr(p, "invalid keyword: %q", source)
}

func buffer(p *Parser) string {
	return p.b.String()
}

func nextChar(p *Parser) {
	p.b.WriteByte(skipChar(p))
}

func skipChar(p *Parser) byte {
	if isEOF(p.r) {
		panic("nextChar: EOF")
	}

	b, _ := p.r.ReadByte()
	p.p += 1
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

func isWhiteSpace(b byte) bool {
	switch b {
	case ' ', '\t', '\r':
		return true
	}
	return false
}

//
// Errors
//

func perror(p *Parser, e error) {
	line, col, source := tokenContext(p, p.t.Pos)
	err := fmt.Errorf("%w at line %d col %d:\n%s", e, line, col, source)
	if p.e == nil {
		p.e = err
	}
	fmt.Fprintln(p.Stderr, err)
	fmt.Fprintln(p.Stderr, string(debug.Stack()))
}

func parseErr(p *Parser, format string, args ...interface{}) {
	if match(p, ErrTokenKind) {
		// We already reported this as a syntax error.
		return
	}
	perror(p, Errorf(ErrParse, format, args...))
}

func syntaxErr(p *Parser, format string, args ...interface{}) {
	makeToken(p, ErrTokenKind)
	perror(p, Errorf(ErrSyntax, format, args...))
}

func unexpectedTokenErr(p *Parser) {
	parseErr(p, "unexpected token %v", p.t)
	nextToken(p)
}

func tokenContext(p *Parser, pos int) (int, int, string) {
	before := p.s[:pos]
	after := p.s[pos:]
	lineStart := bytes.LastIndexByte(before, '\n')
	if lineStart < 0 {
		lineStart = 0
	}
	lineEnd := bytes.IndexByte(after, '\n')
	if lineEnd < 0 {
		lineEnd = len(p.s) - 1
	}
	lineSrc := string(p.s[lineStart:lineEnd])
	line := bytes.Count(before, []byte{'\n'}) + 1
	col := pos - lineStart
	arrow := strings.Repeat("-", col) + "^"
	return line, col, lineSrc + "\n" + arrow
}
