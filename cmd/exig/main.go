package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"unicode"

	"github.com/davecgh/go-spew/spew"
)

func DebugBlockln(title any, value any) (n int, err error) {
	delim := strings.Repeat("-", 12)
	return fmt.Fprintf(os.Stdout, "%s %s %s\n%s\n", delim, title, delim, value)
}

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	flag.Parse()
	source := flag.Arg(0)

	if source == "" {
		fmt.Printf("Usage: %s <file>\n", os.Args[0])
		os.Exit(22)
	}

	src, err := os.Open(source)
	if err != nil {
		log.Fatalf("fail obtaining resource: %s", err)
	}

	defer src.Close()

	lexer := NewLexer(src)
	parser := NewParser(lexer)
	ast := parser.Parse()
	printer := Printer{Ast: ast}
	python := printer.PrintPython()

	for i := len(source) - 1; i >= 0 && !os.IsPathSeparator(source[i]); i-- {
		if source[i] == '.' {
			os.WriteFile(source[:i]+".py", []byte(python), 0644)
		}
	}

	DebugBlockln("compiled to python", python)
}

type Printer struct {
	Ast Statement
}

type PythonPrinter struct {
	usingAssocBuiltin bool
	usingPrintBuiltin bool
}

func (p *PythonPrinter) printStatement(s Statement) string {
	switch s := s.(type) {
	case BlockStatement:
		expressions := make([]string, 0)
		for _, ee := range s.Expressions {
			expressions = append(expressions, p.printExpression(ee))
		}
		return strings.Join(expressions, "\n")
	default:
		return "<unknown>"
	}
}

func (p *PythonPrinter) printExpression(e Expression) string {
	switch e := e.(type) {
	case CallExpression:
		args := make([]string, 0)
		for _, a := range e.Args {
			args = append(args, p.printExpression(a))
		}

		if e.Call == "Print" {
			p.usingPrintBuiltin = true
			return fmt.Sprintf("builtin__print(%s)", strings.Join(args, ","))
		}

		if e.Call == "Let" {
			params := make([]string, 0)
			l := len(e.Args) - 1
			for i := 0; i < l; i++ {
				if a, ok := e.Args[i].(AssignmentExpression); ok {
					variable := a.Lhs.(VariableReferenceExpression)
					value := p.printExpression(a.Rhs)
					params = append(params, fmt.Sprintf("%s = %s", variable.Value, value))
				}
				if v, ok := e.Args[i].(VariableReferenceExpression); ok {
					params = append(params, v.Value)
				}
			}

			return fmt.Sprintf("lambda %s: %s", strings.Join(params, ", "), p.printExpression(e.Args[len(e.Args)-1]))
		}

		if e.Call == "HashMap" {
			return "dict()"
		}

		if e.Call == "Map" {
			return fmt.Sprintf("map(%s, %s)", args[0], strings.Join(args[1:], ", "))
		}

		if e.Call == "List" {
			return fmt.Sprintf("[%s]", strings.Join(args, ", "))
		}

		if e.Call == "Call" {
			return fmt.Sprintf("((%s)(%s))", args[0], strings.Join(args[1:], ","))
		}

		if e.Call == "Assoc" {
			p.usingAssocBuiltin = true
			return fmt.Sprintf("builtin__assoc(%s, %s, %s)", args[0], args[1], args[2])
		}

		if e.Call == "Has" {
			p.usingAssocBuiltin = true
			return fmt.Sprintf("(%s.get(%s, None) != None)", args[1], args[0])
		}

		if e.Call == "Get" {
			p.usingAssocBuiltin = true
			return fmt.Sprintf("(%s.get(%s))", args[1], args[0])
		}

		if e.Call == "Cond" {
			return fmt.Sprintf("%s if %s else %s", p.printExpression(e.Args[1]), p.printExpression(e.Args[0]), p.printExpression(e.Args[2]))
		}

		if e.Call == "Def" {
			if defname, ok := e.Args[0].(VariableReferenceExpression); ok {
				if len(e.Args) > 2 {
					params := make([]string, 0)
					if paramDef, ok := e.Args[1].(CallExpression); ok && paramDef.Call == "Args" {
						for _, arg := range paramDef.Args {
							if argname, ok := arg.(VariableReferenceExpression); ok {
								params = append(params, argname.Value)
							} else if subargs, ok := arg.(CallExpression); ok && subargs.Call == "Args" {
								subparams := make([]string, 0)
								for _, ee := range subargs.Args {
									subparams = append(subparams, p.printExpression(ee))
								}
								params = append(params, subparams...)
							} else if subargs, ok := arg.(CallExpression); ok && subargs.Call == "HashMap" {
								if len(subargs.Args) > 1 {
									log.Fatalf("HashMap currently accept only one argument")
								} else {
									params = append(
										params,
										fmt.Sprintf("%s = dict()", subargs.Args[0].(VariableReferenceExpression).Value),
									)
								}
							} else if a, ok := arg.(AssignmentExpression); ok {
								params = append(
									params,
									fmt.Sprintf("%s = %s", a.Lhs.(VariableReferenceExpression).Value, p.printExpression(a.Rhs)),
								)
							}
						}
					}

					return fmt.Sprintf("%s = lambda %s: %s", defname.Value, strings.Join(params, ", "), p.printExpression(e.Args[2]))
				}
			}

			if a, ok := e.Args[0].(AssignmentExpression); ok {
				return fmt.Sprintf("%s = %s", a.Lhs.(VariableReferenceExpression).Value, p.printExpression(a.Rhs))
			}
		}

		if e.Call == "Inc" {
			for i := range args {
				args[i] += "+1"
			}
			return strings.Join(args, ",")
		}

		return fmt.Sprintf("%s(%s)", e.Call, strings.Join(args, ","))
	case LiteralNumberExpression:
		return e.Value
	case VariableReferenceExpression:
		return e.Value
	}

	return "<unknown>"
}

func (p PythonPrinter) printAssocBuiltin() string {
	return "def builtin__assoc(k, v, obj):\n  obj[k] = v\n  return obj\n"
}

func (p PythonPrinter) printPrintBuiltin() string {
	return "def builtin__print(*args, **kwargs):\n  print(*args, **kwargs)\n  return args[0]\n"
}

func (p *Printer) PrintPython() string {
	pp := PythonPrinter{}
	st := pp.printStatement(p.Ast)
	if pp.usingAssocBuiltin {
		st = fmt.Sprintf("%s\n%s", pp.printAssocBuiltin(), st)
	}
	if pp.usingPrintBuiltin {
		st = fmt.Sprintf("%s\n%s", pp.printPrintBuiltin(), st)
	}
	return st
}

var ErrTokenNotExcpected = errors.New("token not expected")

func (p *Parser) expectToken(tokenType TokenType) (token Token, err error) {
	token, err = p.lexer.Next()

	if err != nil {
		return UnknownToken, err
	}

	if token.Typ == tokenType {
		return token, nil
	} else {
		return UnknownToken, fmt.Errorf("%w: expected: %s, given %s at %s", ErrTokenNotExcpected, tokenType.String(), token.Typ.String(), token.Location.String())
	}
}

type Expression interface {
	IsExpression() bool
}

type VariableReferenceExpression struct {
	Value string
}

type LiteralNumberExpression struct {
	Value string
}

type CallExpression struct {
	Args []Expression
	Call string
}

type AssignmentExpression struct {
	Lhs Expression
	Rhs Expression
}

func (VariableReferenceExpression) IsExpression() bool { return true }
func (LiteralNumberExpression) IsExpression() bool     { return true }
func (CallExpression) IsExpression() bool              { return true }
func (AssignmentExpression) IsExpression() bool        { return true }

func (p *Parser) parseCall() (Expression, error) {
	called, err := p.expectToken(TokenName)
	if err != nil {
		return nil, err
	}

	p.expectToken(TokenSquareBracketOpen)

	args, err := p.parseArgs()
	if err != nil {
		return nil, err
	}

	p.expectToken(TokenSquareBracketClose)

	return CallExpression{
		Call: called.Value,
		Args: args,
	}, nil
}

func (p *Parser) parseAssignment() (Expression, error) {
	lhs, err := p.expectToken(TokenName)
	if err != nil {
		return nil, err
	}

	p.expectToken(TokenEquals)

	rhs, err := p.parseExpression()
	if err != nil {
		return nil, err
	}

	return AssignmentExpression{
		Lhs: VariableReferenceExpression{
			Value: lhs.Value,
		},
		Rhs: rhs,
	}, nil
}

func (p *Parser) parseExpression() (Expression, error) {
	token, err := p.lexer.Peek(1)
	if err != nil {
		return nil, err
	}

	if token.Typ == TokenName {
		if t, err := p.lexer.Peek(2); err == nil && t.Typ == TokenSquareBracketOpen {
			return p.parseCall()
		}

		if t, err := p.lexer.Peek(2); err == nil && t.Typ == TokenEquals {
			return p.parseAssignment()
		}

		p.lexer.Consume()

		return VariableReferenceExpression{
			Value: token.Value,
		}, nil
	}

	if token.Typ == TokenNumber {
		p.lexer.Consume()

		return LiteralNumberExpression{token.Value}, nil
	}

	return nil, fmt.Errorf("failed to parse expression, token %s", spew.Sdump(token))
}

func (p *Parser) parseArgs() ([]Expression, error) {
	args := make([]Expression, 0)

	token := p.lexer.MustPeek(1)
	if token.Typ != TokenSquareBracketClose {
		e, err := p.parseExpression()
		if err != nil {
			return nil, err
		}

		args = append(args, e)
		for {
			token, err = p.lexer.Peek(1)
			if err != nil {
				return nil, err
			}
			if token.Typ == TokenComma {
				p.lexer.Consume()
				e, err := p.parseExpression()
				if err != nil {
					return nil, err
				}
				args = append(args, e)
			} else {
				break
			}
		}
	}

	return args, nil
}

type Statement interface {
	IsStatement() bool
}

type BlockStatement struct {
	Expressions []Expression
}

func (BlockStatement) IsStatement() bool { return true }

func (p *Parser) Parse() Statement {
	block := BlockStatement{
		Expressions: make([]Expression, 0),
	}

	for {
		e, err := p.parseCall()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Fatal(err)
		}

		DebugBlockln("AST", spew.Sdump(e))
		block.Expressions = append(block.Expressions, e)
	}

	return block
}

type Parser struct {
	lexer *Lexer
}

func NewParser(lexer *Lexer) *Parser {
	return &Parser{
		lexer: lexer,
	}
}

type Lexer struct {
	line       int
	col        int
	source     *bufio.Reader
	tokenQueue []TokenResult
}

type TokenType int

func (t TokenType) String() string {
	switch t {
	case TokenSquareBracketOpen:
		return "open square bracket"
	case TokenSquareBracketClose:
		return "close square bracket"
	case TokenName:
		return "name"
	case TokenNumber:
		return "number"
	case TokenComma:
		return "literal comma"
	case TokenSlash:
		return "slash"
	case TokenEquals:
		return "equals sign"
	default:
		log.Fatal("unreachable: trying to print null-token")
	}

	return ""
}

const (
	TokenUnknown = iota
	TokenSquareBracketOpen
	TokenSquareBracketClose
	TokenComma
	TokenName
	TokenNumber
	TokenSlash
	TokenEquals
)

var UnknownToken = Token{Typ: TokenUnknown, Value: "<unknown>", Location: Location{}}

type Location struct {
	Col  int
	Row  int
	File string
}

func (l Location) String() string {
	return fmt.Sprintf("%s:%d:%d", l.File, l.Row, l.Col)
}

type TokenResult struct {
	Token Token
	Error error
}

type Token struct {
	Typ      TokenType
	Value    string
	Location Location
}

func NewLexer(source io.Reader) *Lexer {
	return &Lexer{
		source: bufio.NewReader(source),
	}
}

func (l *Lexer) Consume() {
	if len(l.tokenQueue) < 1 {
		log.Fatal("consume called with empty queue")
	}

	l.tokenQueue = l.tokenQueue[1:]
}

func (l *Lexer) Peek(count int) (Token, error) {
	for i := len(l.tokenQueue); i < count; i += 1 {
		token, err := l.lognext()
		l.tokenQueue = append(l.tokenQueue, TokenResult{Token: token, Error: err})
	}

	token := l.tokenQueue[count-1]

	return token.Token, token.Error
}

func (l *Lexer) MustPeek(count int) Token {
	for i := len(l.tokenQueue); i < count; i += 1 {
		token, err := l.lognext()
		l.tokenQueue = append(l.tokenQueue, TokenResult{Token: token, Error: err})
	}

	token := l.tokenQueue[count-1]

	if token.Error != nil {
		log.Fatal(token.Error)
	}

	return token.Token
}

func (l *Lexer) MustNext() Token {
	if len(l.tokenQueue) > 0 {
		t := l.tokenQueue[0]
		if t.Error != nil {
			log.Fatal(t.Error)
		}

		l.tokenQueue = l.tokenQueue[1:]

		return t.Token
	}

	t, err := l.Next()
	if err != nil {
		log.Fatal(err)
	}
	return t
}

func (l *Lexer) Next() (Token, error) {
	if len(l.tokenQueue) > 0 {
		t := l.tokenQueue[0]
		l.tokenQueue = l.tokenQueue[1:]
		return t.Token, t.Error
	}

	return l.lognext()
}

func (l *Lexer) lognext() (Token, error) {
	t, e := l.next()
	if e == nil {
		fmt.Printf("TOKEN: [%+v, %+v]\n", t, e)
	}
	return t, e
}

func (l *Lexer) next() (Token, error) {
	name := make([]rune, 0)
	number := make([]rune, 0)
	searchName := false
	searchNumber := false
	maybeComment := false

	for {
		r, _, err := l.source.ReadRune()
		if err != nil {
			if err == io.EOF {
				return UnknownToken, err
			}
			log.Fatal(err)
		}

		switch true {
		case searchName:
			if unicode.IsDigit(r) || unicode.IsLetter(r) {
				name = append(name, r)
				l.col += 1
				continue
			} else {
				l.source.UnreadRune()
				return Token{
					Typ:      TokenName,
					Value:    string(name),
					Location: Location{Row: l.line, Col: l.col - len(name), File: ""},
				}, nil
			}
		case searchNumber:
			if unicode.IsDigit(r) {
				number = append(number, r)
				l.col += 1
				continue
			} else {
				l.source.UnreadRune()
				return Token{
					Typ:      TokenNumber,
					Value:    string(number),
					Location: Location{Row: l.line, Col: l.col - len(number), File: ""},
				}, nil
			}
		}

		if unicode.IsLetter(r) {
			name = append(name, r)
			l.col += 1
			searchName = true
			continue
		}

		if unicode.IsDigit(r) {
			number = append(number, r)
			l.col += 1
			searchNumber = true
			continue
		}

		switch r {
		case '=':
			return Token{
				Typ:      TokenEquals,
				Value:    "=",
				Location: Location{},
			}, nil
		case '\n':
			l.line += 1
			l.col = 0
			continue
		case ' ', '\t':
			l.col += 1
			continue
		case '/':
			if maybeComment {
				_, err = l.source.ReadBytes('\n')
				if err != nil {
					return UnknownToken, err
				}

				maybeComment = false
				continue
			}

			maybeComment = true
			continue
		case '[':
			return Token{
				Typ:   TokenSquareBracketOpen,
				Value: "[",
				Location: Location{
					Row:  l.line,
					Col:  l.col,
					File: "",
				},
			}, nil
		case ']':
			return Token{
				Typ:   TokenSquareBracketClose,
				Value: "]",
				Location: Location{
					Row:  l.line,
					Col:  l.col,
					File: "",
				},
			}, nil
		case ',':
			return Token{
				Typ:   TokenComma,
				Value: ",",
				Location: Location{
					Row:  l.line,
					Col:  l.col,
					File: "",
				},
			}, nil
		}

		break
	}

	return UnknownToken, io.EOF
}
