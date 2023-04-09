package parser

// Go's type system not allowing using interface{}
// because then we can pass any value to a methods.
type Expression interface {
	// That's why we need the dummy method, which does nothing.
	IsExpression() bool
}

type Statement interface {
	// Same here
	IsStatement() bool
}

// Expression, that references a variable
type VariableReferenceExpression struct {
	Value string
}

// Expression, that represent literal number
type LiteralNumberExpression struct {
	Value string
}

// Expression, that represents a function call
type CallExpression struct {
	// Arguments of that function is array of arbitrary expressions
	Args []Expression

	// Function name
	Call string
}

// Expression, that represents a variable assignment
type AssignmentExpression struct {
	Lhs Expression
	Rhs Expression
}

// Implementing interface
func (VariableReferenceExpression) IsExpression() bool { return true }
func (LiteralNumberExpression) IsExpression() bool     { return true }
func (CallExpression) IsExpression() bool              { return true }
func (AssignmentExpression) IsExpression() bool        { return true }

// Block statement, like in normal languages, carries a bunch of other statements or expressions
type BlockStatement struct {
	Expressions []Expression
}

func (BlockStatement) IsStatement() bool { return true }
