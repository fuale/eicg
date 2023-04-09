package lexer

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"unicode"
)

type Lexer struct {
	// Row - is the current row in which the cursor is located.
	row int

	// Column - is, respectively, column of the current row.
	col int

	// Source - is the source file reader.
	// here we use a bufio.Scanner, which also buffers input for us
	// and allows to use convenient functions, like `ReadRune`
	source *bufio.Reader

	// TokenQueue - is the queue of tokens that have been read from the source but not yet parsed.
	// It is used to keep tokens, that we peeked, but not yet consumed.
	tokenQueue []TokenResult
}

// Constructs a new Lexer from io.Reader
func New(source io.Reader) *Lexer {
	return &Lexer{
		source: bufio.NewReader(source),
	}
}

// Consume - consumes token from `tokenQueue`
// and not trigger lexer to lex new token. Used for peeking.
func (l *Lexer) Consume() {
	if len(l.tokenQueue) < 1 {
		log.Fatal("consume called with empty queue")
	}

	l.tokenQueue = l.tokenQueue[1:]
}

// Peek - peek the next token at specified position in tokenQueue
func (l *Lexer) Peek(count int) (Token, error) {
	// Make sure we have enough tokens in tokenQueue
	for i := len(l.tokenQueue); i < count; i += 1 {
		token, err := l.lognext()

		// Simply append to queue without checking for error
		l.tokenQueue = append(l.tokenQueue, TokenResult{Token: token, Error: err})
	}

	// If we have enough tokens in tokenQueue,
	// return the token at count-1, which is token index
	token := l.tokenQueue[count-1]

	return token.Token, token.Error
}

// MustPeek - peek the next token at specified position in tokenQueue
// but throws fatal error if there is no token at specified position.
// Used in alghorithms, where must be at least `count` tokens.
func (l *Lexer) MustPeek(count int) Token {
	for i := len(l.tokenQueue); i < count; i += 1 {
		token, err := l.lognext()
		l.tokenQueue = append(l.tokenQueue, TokenResult{Token: token, Error: err})
	}

	token := l.tokenQueue[count-1]

	if token.Error != nil {
		log.Fatal(token.Error)
	}

	// only return the token, because we know that error is nil
	return token.Token
}

// Next - is like `next`, but returns token from queue
// if there is any and then removes it from queue.
func (l *Lexer) Next() (Token, error) {
	// Check `tokenQueue` is not empty
	if len(l.tokenQueue) > 0 {
		// pick it up...
		t := l.tokenQueue[0]
		// ...and remove
		l.tokenQueue = l.tokenQueue[1:]
		return t.Token, t.Error
	}

	return l.lognext()
}

// MustNext - is like `Next`, but throws fatal error if there is no token.
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

// lognext - it is a `next` decorator that logs the next token.
func (l *Lexer) lognext() (Token, error) {
	t, e := l.next()
	if e == nil {
		fmt.Printf("TOKEN: [%+v, %+v]\n", t, e)
	}
	return t, e
}

// `next` - is the primary lexer function that does all the work.
func (l *Lexer) next() (Token, error) {
	// name - array, which we use to collect runes,
	//        which can possibly be a variable name, that means multiple runes
	name := make([]rune, 0)

	// number - is the same as `name`, but for numbers.
	number := make([]rune, 0)

	// searchName - boolean flag, which indicates that we currently lexing `name`
	searchName := false

	// searchNumber - same for numbers
	searchNumber := false

	// flag - which needed for check double slashes for comments
	maybeComment := false

	// Main loop. Tokenization usually performs without recursion,
	//            because tokens is not a recursive structure -
	//            tokens, basically, is just array
	for {
		// start lexing by reading one rune
		r, _, err := l.source.ReadRune()
		if err != nil {
			// check for io.EOF.
			// Need to explicitly handle `io.EOF` for properly handle end of file
			if err == io.EOF {
				return UnknownToken, err
			}

			// Otherwise throw fatal
			log.Fatal(err)
		}

		// `switch true` - it's a trick to replace `if {} else if {}` over
		// booleans with `switch` by a better looking control flow.
		switch true {
		// When we find a rune that represents a letter,
		// we need to continue searching for another letter.
		// That's why we use a Boolean flag for that purpose:
		// in first iteration, in which we find letter,
		// we set `searchName` to true, then here we check if that flag is set to True,
		// and if so, we append the next rune to the array
		// and continue until, finally, we encounter a non-letter rune.
		case searchName:
			// When we looking for a name, the first character should be a letter,
			// while the second and the rest may be also a numbers.
			if unicode.IsDigit(r) || unicode.IsLetter(r) {
				name = append(name, r)
				// When appending a rune, don't forget to increase `col`
				l.col += 1
				continue
			} else {
				// If we encounter a non-letter rune, we need to place
				// it back in `source` buffer, because the last readed
				// rune does not belongs to `name`
				l.source.UnreadRune()
				return Token{
					Typ:      TokenName,
					Value:    string(name),
					Location: Location{Row: l.row, Col: l.col - len(name), File: ""},
				}, nil
			}
			// Searching number is done almost exactly the same
			// but here we searching only for numbers.
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
					Location: Location{Row: l.row, Col: l.col - len(number), File: ""},
				}, nil
			}
		}

		// Here we start scanning for name
		if unicode.IsLetter(r) {
			name = append(name, r)
			l.col += 1
			searchName = true
			continue // using continue, because we want stay in a loop
		}

		// Same for numbers
		if unicode.IsDigit(r) {
			number = append(number, r)
			l.col += 1
			searchNumber = true
			continue
		}

		// Single-rune tokens.
		// Here we construct tokens from one or several runes.
		switch r {
		// For example: if we encounter a equal sign,
		// we immediatly return it as a token
		case '=':
			return Token{
				Typ:      TokenEquals,
				Value:    "=",
				Location: Location{},
			}, nil
		// When encounter a space, we need to strip it,
		// advance position and continue as usual
		case '\n', '\v', '\f', '\r':
			l.row += 1
			l.col = 0
			continue
		case ' ', '\t', 0x85, 0xA0: // Some of weird runes a stolen from go's `unicode.IsSpace` builtin function
			l.col += 1
			continue
		case '/':
			// Same mechanic as with names
			if maybeComment {
				// Here we strip all runes to the end of the line
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
					Row:  l.row,
					Col:  l.col,
					File: "",
				},
			}, nil
		case ']':
			return Token{
				Typ:   TokenSquareBracketClose,
				Value: "]",
				Location: Location{
					Row:  l.row,
					Col:  l.col,
					File: "",
				},
			}, nil
		case ',':
			return Token{
				Typ:   TokenComma,
				Value: ",",
				Location: Location{
					Row:  l.row,
					Col:  l.col,
					File: "",
				},
			}, nil
		}

		// I think, here we should throw an error,
		// because we don't know what kind of rune it is.
		break
	}

	// If we reach here, we have reached the end of the file,
	// or it is probably a bug
	return UnknownToken, io.EOF
}
