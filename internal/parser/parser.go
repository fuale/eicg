package parser

import (
	"errors"
	"fmt"
	"io"
	"log"

	"github.com/davecgh/go-spew/spew"
	"github.com/fuale/eicg/internal"
	"github.com/fuale/eicg/internal/lexer"
)

var ErrTokenNotExpected = errors.New("token not expected")

// Parser - is a deeply recursive algorithm that
// parses a stream of tokens into a tree structure.
// Basically, it turns
//
//	[NAME{x} OPENBRACKET{x} CLOSEBRACKET{x}]
//
// to
//
//	CallExpr { Call: x, Args: [] }
type Parser struct {
	lexer *lexer.Lexer
}

func New(lexer *lexer.Lexer) *Parser {
	return &Parser{
		lexer: lexer,
	}
}

// Main function. Here we create BlockStatement as top level node,
// and then parse calls (only calls allowed in top level in this implementation) one by one.
func (p *Parser) Parse() Statement {
	block := BlockStatement{
		Expressions: make([]Expression, 0),
	}

	for {
		// Here we first calling recursive function to parse a function call.
		// parse<Something> functions usually calls each other and stops when no tokens left.
		e, err := p.parseCall()
		if err == io.EOF {
			// Gracefully handle EOF
			break
		} else if err != nil {
			log.Fatal(err)
		}

		internal.DebugBlock("AST", spew.Sdump(e))
		block.Expressions = append(block.Expressions, e)
	}

	return block
}

// parseCall - for example, tries to parse a function call. :^)
// It consumes a NameToken, which will be the name of the function.
// Then open and close brackets, between which we parse the arguments.
func (p *Parser) parseCall() (Expression, error) {
	called, err := p.expectToken(lexer.TokenName)
	if err != nil {
		return nil, err
	}

	_, _ = p.expectToken(lexer.TokenSquareBracketOpen)

	args, err := p.parseArgs()
	if err != nil {
		return nil, err
	}

	_, _ = p.expectToken(lexer.TokenSquareBracketClose)

	return CallExpression{
		Call: called.Value,
		Args: args,
	}, nil
}

func (p *Parser) parseAssignment() (Expression, error) {
	lhs, err := p.expectToken(lexer.TokenName)
	if err != nil {
		return nil, err
	}

	_, _ = p.expectToken(lexer.TokenEquals)

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

// I think, parseExpression is a the most difficult to program function,
// because there is many conditions and recursive calls
func (p *Parser) parseExpression() (Expression, error) {
	token, err := p.lexer.Peek(1)
	if err != nil {
		return nil, err
	}

	if token.Typ == lexer.TokenName {
		if t, err := p.lexer.Peek(2); err == nil && t.Typ == lexer.TokenSquareBracketOpen {
			return p.parseCall()
		}

		if t, err := p.lexer.Peek(2); err == nil && t.Typ == lexer.TokenEquals {
			return p.parseAssignment()
		}

		p.lexer.Consume()

		return VariableReferenceExpression{
			Value: token.Value,
		}, nil
	}

	if token.Typ == lexer.TokenNumber {
		p.lexer.Consume()

		return LiteralNumberExpression{token.Value}, nil
	}

	return nil, fmt.Errorf("failed to parse expression, token %s", spew.Sdump(token))
}

func (p *Parser) parseArgs() ([]Expression, error) {
	args := make([]Expression, 0)

	token := p.lexer.MustPeek(1)
	if token.Typ != lexer.TokenSquareBracketClose {
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
			if token.Typ == lexer.TokenComma {
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

// expectToken - is a helper function that ensures that the next token is the one we expected.
func (p *Parser) expectToken(tokenType lexer.TokenType) (token lexer.Token, err error) {
	token, err = p.lexer.Next()

	if err != nil {
		return lexer.UnknownToken, err
	}

	if token.Typ == tokenType {
		return token, nil
	} else {
		return lexer.UnknownToken, fmt.Errorf("%w: expected: %s, given %s at %s", ErrTokenNotExpected, tokenType.String(), token.Typ.String(), token.Location.String())
	}
}
