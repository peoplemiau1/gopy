package generator

import (
	"bytes"
	"fmt"
	"gopy/pkg/ast"
	"strings"
)

type Generator struct {
	functions         bytes.Buffer
	mainBody          bytes.Buffer
	declaredVariables map[string]bool
	declaredClasses   map[string]bool
}

func New() *Generator {
	return &Generator{
		declaredVariables: make(map[string]bool),
		declaredClasses:   make(map[string]bool),
	}
}

func (g *Generator) Generate(node ast.Node) (string, error) {
	program, ok := node.(*ast.Program)
	if !ok {
		return "", fmt.Errorf("unsupported node type: %T", node)
	}

	for _, stmt := range program.Statements {
		err := g.generateStatement(stmt)
		if err != nil {
			return "", err
		}
	}

	var out bytes.Buffer
	out.WriteString("package main\n\n")
	out.WriteString("import (\n\t\"fmt\"\n)\n\n")
	out.WriteString(g.functions.String())
	out.WriteString("func main() {\n")
	out.WriteString(g.mainBody.String())
	out.WriteString("}\n")

	return out.String(), nil
}

func (g *Generator) generateStatement(stmt ast.Statement) error {
	switch stmt := stmt.(type) {
	case *ast.LetStatement:
		val, err := g.generateExpression(stmt.Value)
		if err != nil {
			return err
		}
		g.mainBody.WriteString(fmt.Sprintf("\t%s := %s\n", stmt.Name.Value, val))
	case *ast.ExpressionStatement:
		expr, err := g.generateExpression(stmt.Expression)
		if err != nil {
			return err
		}
		g.mainBody.WriteString(fmt.Sprintf("\t%s\n", expr))
	default:
		return fmt.Errorf("unsupported statement type: %T", stmt)
	}
	return nil
}

func (g *Generator) generateExpression(expr ast.Expression) (string, error) {
	switch expr := expr.(type) {
	case *ast.IntegerLiteral:
		return fmt.Sprintf("%d", expr.Value), nil
	case *ast.StringLiteral:
		return fmt.Sprintf("\"%s\"", expr.Value), nil
	case *ast.Identifier:
		return expr.Value, nil
	case *ast.CallExpression:
		if expr.Function.String() == "print" {
			var args []string
			for _, arg := range expr.Arguments {
				a, err := g.generateExpression(arg)
				if err != nil {
					return "", err
				}
				args = append(args, a)
			}
			return fmt.Sprintf("fmt.Println(%s)", strings.Join(args, ", ")), nil
		}
	case *ast.ArrayLiteral:
		var elements []string
		for _, el := range expr.Elements {
			elem, err := g.generateExpression(el)
			if err != nil {
				return "", err
			}
			elements = append(elements, elem)
		}
		return fmt.Sprintf("[]interface{}{%s}", strings.Join(elements, ", ")), nil
	case *ast.IndexExpression:
		left, err := g.generateExpression(expr.Left)
		if err != nil {
			return "", err
		}
		index, err := g.generateExpression(expr.Index)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s.([]interface{})[%s]", left, index), nil
	}
	return "", fmt.Errorf("unsupported expression type: %T", expr)
}
