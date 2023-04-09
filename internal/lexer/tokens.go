package lexer

import (
	"fmt"
	"log"
)

// Token - is simple structure that carries information about a single token
type Token struct {
	// Type is one of TokenType enum values
	Typ TokenType

	// Value is string representation of the token
	Value string

	// Location is the location of the token in the source code
	Location Location
}

// Go lacks of enums, that being said, we need to mimic it below
type TokenType int

// TokenType.String - Just returns the string representation of the token type
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
		// When we encounter nil-token or unknown token we need to inform ourselves
		log.Fatal("unreachable: trying to print null-token")
	}

	return "<unknown>"
}

// Here using go's `iota` feature to autoincrement constants
const (
	// first token type need to be 0 (falsy value), which allows us to handle unknown token
	TokenUnknown TokenType = iota
	TokenSquareBracketOpen
	TokenSquareBracketClose
	TokenComma
	TokenName
	TokenNumber
	TokenSlash
	TokenEquals
)

// Dummy token needed for passing it as non-pointer
var UnknownToken = Token{Typ: TokenUnknown, Value: "<unknown>", Location: Location{}}

// Location - is simple location of a token
type Location struct {
	Col  int
	Row  int
	File string
}

func (l Location) String() string {
	return fmt.Sprintf("%s:%d:%d", l.File, l.Row, l.Col)
}

// Using for represents scanned token in tokenQueue
type TokenResult struct {
	Token Token
	Error error
}
