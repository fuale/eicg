package python

import (
	"fmt"
	"github.com/fuale/eicg/internal/parser"
	"log"
	"strings"
)

type Printer struct {
	usingAssocBuiltin bool
	usingPrintBuiltin bool
}

func (p *Printer) String(ast parser.Statement) string {
	st := p.printStatement(ast)
	if p.usingAssocBuiltin {
		st = fmt.Sprintf("%s\n%s", p.printAssocBuiltin(), st)
	}
	if p.usingPrintBuiltin {
		st = fmt.Sprintf("%s\n%s", p.printPrintBuiltin(), st)
	}
	return st
}

func (p *Printer) printStatement(s parser.Statement) string {
	switch s := s.(type) {
	case parser.BlockStatement:
		expressions := make([]string, 0)
		for _, ee := range s.Expressions {
			expressions = append(expressions, p.printExpression(ee))
		}
		return strings.Join(expressions, "\n")
	default:
		return "<unknown>"
	}
}

func (p *Printer) printExpression(e parser.Expression) string {
	switch e := e.(type) {
	case parser.CallExpression:
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
				if a, ok := e.Args[i].(parser.AssignmentExpression); ok {
					variable := a.Lhs.(parser.VariableReferenceExpression)
					value := p.printExpression(a.Rhs)
					params = append(params, fmt.Sprintf("%s = %s", variable.Value, value))
				}
				if v, ok := e.Args[i].(parser.VariableReferenceExpression); ok {
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
			if defname, ok := e.Args[0].(parser.VariableReferenceExpression); ok {
				if len(e.Args) > 2 {
					params := make([]string, 0)
					if paramDef, ok := e.Args[1].(parser.CallExpression); ok && paramDef.Call == "Args" {
						for _, arg := range paramDef.Args {
							if argname, ok := arg.(parser.VariableReferenceExpression); ok {
								params = append(params, argname.Value)
							} else if subargs, ok := arg.(parser.CallExpression); ok && subargs.Call == "Args" {
								subparams := make([]string, 0)
								for _, ee := range subargs.Args {
									subparams = append(subparams, p.printExpression(ee))
								}
								params = append(params, subparams...)
							} else if subargs, ok := arg.(parser.CallExpression); ok && subargs.Call == "HashMap" {
								if len(subargs.Args) > 1 {
									log.Fatalf("HashMap currently accept only one argument")
								} else {
									params = append(
										params,
										fmt.Sprintf("%s = dict()", subargs.Args[0].(parser.VariableReferenceExpression).Value),
									)
								}
							} else if a, ok := arg.(parser.AssignmentExpression); ok {
								params = append(
									params,
									fmt.Sprintf("%s = %s", a.Lhs.(parser.VariableReferenceExpression).Value, p.printExpression(a.Rhs)),
								)
							}
						}
					}

					return fmt.Sprintf("%s = lambda %s: %s", defname.Value, strings.Join(params, ", "), p.printExpression(e.Args[2]))
				}
			}

			if a, ok := e.Args[0].(parser.AssignmentExpression); ok {
				return fmt.Sprintf("%s = %s", a.Lhs.(parser.VariableReferenceExpression).Value, p.printExpression(a.Rhs))
			}
		}

		if e.Call == "Inc" {
			for i := range args {
				args[i] += "+1"
			}
			return strings.Join(args, ",")
		}

		return fmt.Sprintf("%s(%s)", e.Call, strings.Join(args, ","))
	case parser.LiteralNumberExpression:
		return e.Value
	case parser.VariableReferenceExpression:
		return e.Value
	}

	return "<unknown>"
}

func (p *Printer) printAssocBuiltin() string {
	return "def builtin__assoc(k, v, obj):\n  obj[k] = v\n  return obj\n"
}

func (p *Printer) printPrintBuiltin() string {
	return "def builtin__print(*args, **kwargs):\n  print(*args, **kwargs)\n  return args[0]\n"
}
