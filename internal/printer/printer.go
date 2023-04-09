package printer

import (
	"github.com/fuale/eicg/internal/parser"
	"github.com/fuale/eicg/internal/printer/printers/python"
)

type Printer struct {
	Ast parser.Statement
}

func New(ast parser.Statement) *Printer {
	return &Printer{
		Ast: ast,
	}
}

func (p *Printer) PrintPython() string {
	pp := python.Printer{}
	return pp.String(p.Ast)
}
