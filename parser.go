package main

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
	LParenTokenKind    TokenKind = "LParen"
	RParenTokenKind    TokenKind = "RParen"
	AttributeTokenKind TokenKind = "Attribute"
	StringTokenKind    TokenKind = "String"
	NumberTokenKind    TokenKind = "Number"
	// Keywords
	DirTokenKind  TokenKind = "dir"
	FileTokenKind TokenKind = "file"

	NewlineTokenKind    TokenKind = "Newline"
	WhiteSpaceTokenKind TokenKind = "WhiteSpace"
	EofTokenKind        TokenKind = "EOF"
)

var keywords = map[string]*Token{
	"dir":  &Token{DirTokenKind, "dir"},
	"file": &Token{FileTokenKind, "file"},
}

var EOF = &Token{EofTokenKind, ""}

type Token struct {
	Kind  TokenKind
	Value string
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
}

// TODO: makeToken
func (p *Parser) Parse(r io.Reader) (*Config, error) {
	src, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	p.src = src
	p.r = bufio.NewReader(bytes.NewReader(p.src))
	p.t = EOF
	p.lineEndings = []int{0}
	c := parseConfig(p)
	return c, p.err
}

// config    = sexpr*
// sexpr     = '(' literal arg* ')'
// arg       = sexpr
//           | literal
// literal   = KEYWORD
//           | ATTRIBUTE
//           | STRING
//			 | NUMBER
// KEYWORD   = 'dir' | 'file'
// ATTRIBUTE = '@' [a-zA-Z0-9_-]+
// STRING    = '"'[^"]*'"'
// NUMBER    = [0-9]+
func parseConfig(p *Parser) *Config {
	c := &Config{}
	nextToken(p)
	for peekToken(p).Kind != EofTokenKind {
		c.SExprs = append(c.SExprs, parseSExpr(p))
		ignoreNewlines(p)
	}
	return c
}

func parseSExpr(p *Parser) *SExpr {
	consume(p, LParenTokenKind)
	ignoreNewlines(p)

	literal := parseLiteral(p)
	ignoreNewlines(p)

	var args []*Arg
	t := peekToken(p)
	for t != EOF && t.Kind != RParenTokenKind {
		args = append(args, parseArg(p))
		ignoreNewlines(p)
		t = peekToken(p)
	}

	consume(p, RParenTokenKind)
	return &SExpr{Literal: literal, Args: args}
}

func parseArg(p *Parser) *Arg {
	if peekToken(p).Kind == LParenTokenKind {
		e := parseSExpr(p)
		return &Arg{SExpr: e}
	}
	l := parseLiteral(p)
	return &Arg{Literal: l}
}

func parseLiteral(p *Parser) *Literal {
	t := peekToken(p)
	switch t.Kind {
	case DirTokenKind, FileTokenKind, AttributeTokenKind, StringTokenKind, NumberTokenKind:
		nextToken(p)
		return &Literal{Token: t}
	}
	unexpectedTokenErr(p, "parseLiteral", t)
	return nil
}

func consume(p *Parser, k TokenKind) {
	if peekToken(p).Kind != k {
		unexpectedTokenErr(p, "consume", p.t)
	}
	nextToken(p)
}

func ignoreNewlines(p *Parser) {
	for peekToken(p).Kind == NewlineTokenKind {
		nextToken(p)
	}
}

type ParseError struct {
	err string
}

func (e ParseError) Error() string {
	return e.err
}

func unexpectedTokenErr(p *Parser, caller string, t *Token) {
	around := surroundingText(p)
	arrow := strings.Repeat("-", p.col) + "^"
	err := fmt.Errorf(`
%s: got unexpected token (%v, %q) at line %d col %d
%s
%s
%s
`, caller, t.Kind, t.Value, p.line+1, p.col+1, around, arrow, string(debug.Stack()))

	parseErr(p, err)
}

func parseErr(p *Parser, e error) {
	p.err = e
	fmt.Fprint(os.Stderr, e.Error())
}

func surroundingText(p *Parser) []byte {
	startLine := p.line - 2
	if startLine < 0 {
		startLine = 0
	}
	start := p.lineEndings[startLine]
	end := p.pos + 20
	if end >= len(p.src) {
		end = len(p.src)
	}
	return p.src[start:end]
}

//
// Lexer
//

func peekToken(p *Parser) *Token {
	return p.t
}

func nextToken(p *Parser) {
	for {
		if isEOF(p.r) {
			p.t = EOF
			return
		}
		switch peekChar(p.r) {
		case '(':
			nextChar(p)
			p.t = &Token{LParenTokenKind, "("}
			return
		case ')':
			nextChar(p)
			p.t = &Token{RParenTokenKind, ")"}
			return
		case '@':
			p.t = readAttribute(p)
			return
		case '\n':
			nextChar(p)
			p.t = &Token{NewlineTokenKind, ""}
			return
		case '"':
			p.t = readString(p)
			return
		case ' ':
			nextChar(p)
			continue
		}
		if isDigit(peekChar(p.r)) {
			p.t = readNumber(p)
			return
		}
		p.t = readKeyword(p)
		return
	}
}

func readAttribute(p *Parser) *Token {
	value := string(nextChar(p))
	for !isEOF(p.r) {
		c := peekChar(p.r)
		if isWhiteSpace(c) || c == '\n' {
			break
		}
		value += string(nextChar(p))
	}
	return &Token{AttributeTokenKind, value}
}

// TODO: Handle escaped quotes.
func readString(p *Parser) *Token {
	var b strings.Builder

	nextChar(p)
	for !(isEOF(p.r) || peekChar(p.r) == '"') {
		b.WriteByte(nextChar(p))
	}
	nextChar(p)

	value := b.String()
	if t, ok := keywords[value]; ok {
		return t
	}

	return &Token{StringTokenKind, value}
}

func readNumber(p *Parser) *Token {
	var b strings.Builder
	for !isEOF(p.r) && isDigit(peekChar(p.r)) {
		b.WriteByte(nextChar(p))
	}

	return &Token{NumberTokenKind, b.String()}
}

func readKeyword(p *Parser) *Token {
	var b strings.Builder

	b.WriteByte(nextChar(p))
	for !(isEOF(p.r) || isWhiteSpace(peekChar(p.r))) {
		b.WriteByte(nextChar(p))
	}

	value := b.String()
	if t, ok := keywords[value]; ok {
		return t
	}

	panic(fmt.Sprintf("invalid keyword %q", value))
	// if c == '/' || c == '_' || c == '-' || c == '\n' || isAlpha(c) {
	// 	value += string(nextChar(p))
	// 	continue
	// }
}

func isEOF(r *bufio.Reader) bool {
	_, err := r.Peek(1)
	return errors.Is(err, io.EOF)
}

func nextChar(p *Parser) byte {
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

func isWhiteSpace(b byte) bool {
	switch b {
	case ' ', '\t', '\r':
		return true
	}
	return false
}

// func isAlpha(b byte) bool {
// 	return 'a' <= b && b <= 'z' || 'A' <= b && b <= 'Z' || '1' <= b && b <= '9'
// }

func isDigit(b byte) bool {
	return '0' <= b && b <= '9'
}
