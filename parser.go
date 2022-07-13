package mktree

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
	r   *bufio.Reader
	t   *Token
	err error

	line        int
	col         int
	pos         int
	lineEndings []int
	src         []byte
	buf         strings.Builder
}

func Parse(r io.Reader) (*Config, error) {
	p := &Parser{}
	return p.Parse(r)
}

func (p *Parser) Parse(r io.Reader) (*Config, error) {
	src, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	p.src = src
	p.r = bufio.NewReader(bytes.NewReader(p.src))
	makeToken(p, EofTokenKind)
	p.lineEndings = []int{0}
	c := parseConfig(p)
	return c, p.err
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
	unexpectedTokenErr(p, "parseLiteral")
	return nil
}

func consume(p *Parser, k TokenKind) {
	if !match(p, k) {
		unexpectedTokenErr(p, "consume")
	}
	nextToken(p)
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

type ParseError struct {
	err string
}

func (e ParseError) Error() string {
	return e.err
}

func tokLineCol(p *Parser, t *Token) (int, int) {
	srcSoFar := p.src[:t.Pos]
	line := bytes.Count(srcSoFar, []byte{'\n'}) + 1
	col := bytes.LastIndexByte(srcSoFar, '\n')
	if col < 0 {
		col = p.pos
	}
	return line, col
}

func unexpectedTokenErr(p *Parser, caller string) {
	line, col := tokLineCol(p, p.t)
	around := surroundingText(p)
	arrow := strings.Repeat("-", col) + "^"
	err := fmt.Errorf(`
%s: got unexpected token (%v, %q) at line %d col %d
%s
%s
%s
`, caller, p.t.Kind, p.t.Value, line, col, around, arrow, string(debug.Stack()))

	parseErr(p, err)
}

func makeToken(p *Parser, k TokenKind) {
	p.t = &Token{
		Kind:  k,
		Value: p.buf.String(),
		Pos:   p.pos - p.buf.Len(),
	}
	p.buf.Reset()
}

func surroundingText(p *Parser) []byte {
	// startLine := p.line - 2
	// if startLine < 0 {
	// 	startLine = 0
	// }
	// start := p.lineEndings[startLine]
	// end := p.pos + 20
	// if end >= len(p.src) {
	// 	end = len(p.src)
	// }
	start := p.t.Pos - 10
	end := p.t.Pos + 10
	if start < 0 {
		start = 0
	}
	if end >= len(p.src) {
		end = len(p.src) - 1
	}
	return p.src[start:end]
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

	if kind, ok := keywords[buffer(p)]; ok {
		makeToken(p, kind)
		return
	}

	syntaxErr(p, "invalid keyword: %q", buffer(p))
}

func buffer(p *Parser) string {
	return p.buf.String()
}

func nextChar(p *Parser) {
	p.buf.WriteByte(skipChar(p))
}

func skipChar(p *Parser) byte {
	if isEOF(p.r) {
		panic("nextChar: EOF")
	}

	b, _ := p.r.ReadByte()
	p.pos += 1
	p.col += 1
	if b == '\n' {
		p.col = 0
		p.line += 1
		p.lineEndings = append(p.lineEndings, p.pos)
	}

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

func parseErr(p *Parser, e error) {
	p.err = e
	fmt.Fprint(os.Stderr, e.Error())
}

func syntaxErr(p *Parser, format string, args ...interface{}) {
	p.err = fmt.Errorf("syntax error: "+format, args...)
	fmt.Fprint(os.Stderr, p.err.Error())
}
