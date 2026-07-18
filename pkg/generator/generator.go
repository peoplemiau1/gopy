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
	switch s := stmt.(type) {
	case *ast.LetStatement:
		return g.generateLetStatement(s)
	case *ast.ExpressionStatement:
		return g.generateExpressionStatement(s)
	case *ast.ReturnStatement:
		return g.generateReturnStatement(s)
	case *ast.AssignmentStatement:
		return g.generateAssignmentStatement(s)
	case *ast.ClassStatement:
		return g.generateClassStatement(s)
	case *ast.ForStatement:
		return g.generateForStatement(s)
	default:
		return fmt.Errorf("unsupported statement type: %T", stmt)
	}
}

func (g *Generator) generateLetStatement(stmt *ast.LetStatement) error {
	if funcLit, ok := stmt.Value.(*ast.FunctionLiteral); ok {
		return g.generateFunctionDef(stmt.Name.Value, funcLit)
	}
	val, err := g.generateExpression(stmt.Value)
	if err != nil {
		return err
	}
	g.mainBody.WriteString(fmt.Sprintf("\t%s := %s\n", stmt.Name.Value, val))
	return nil
}

func (g *Generator) generateExpressionStatement(stmt *ast.ExpressionStatement) error {
	expr, err := g.generateExpression(stmt.Expression)
	if err != nil {
		return err
	}
	lines := strings.Split(strings.TrimSuffix(expr, "\n"), "\n")
	for i, line := range lines {
		if i == 0 {
			g.mainBody.WriteString("\t" + line + "\n")
		} else {
			g.mainBody.WriteString(line + "\n")
		}
	}
	return nil
}

func (g *Generator) generateReturnStatement(stmt *ast.ReturnStatement) error {
	if stmt.ReturnValue == nil {
		g.mainBody.WriteString("\treturn\n")
		return nil
	}
	val, err := g.generateExpression(stmt.ReturnValue)
	if err != nil {
		return err
	}
	g.mainBody.WriteString(fmt.Sprintf("\treturn %s\n", val))
	return nil
}

func (g *Generator) generateAssignmentStatement(stmt *ast.AssignmentStatement) error {
	name, err := g.generateExpression(stmt.Name)
	if err != nil {
		return err
	}
	val, err := g.generateExpression(stmt.Value)
	if err != nil {
		return err
	}
	g.mainBody.WriteString(fmt.Sprintf("\t%s = %s\n", name, val))
	return nil
}

func (g *Generator) generateClassStatement(stmt *ast.ClassStatement) error {
	g.declaredClasses[stmt.Name.Value] = true

	// Generate struct type
	var fields []string
	for _, f := range stmt.Fields {
		fields = append(fields, fmt.Sprintf("%s interface{}", f.Value))
	}
	if len(fields) == 0 {
		g.functions.WriteString(fmt.Sprintf("type %s struct{}\n\n", stmt.Name.Value))
	} else {
		g.functions.WriteString(fmt.Sprintf("type %s struct{%s}\n\n", stmt.Name.Value, strings.Join(fields, "; ")))
	}

	// Generate methods
	for _, method := range stmt.Methods {
		if err := g.generateMethodStatement(stmt.Name.Value, method); err != nil {
			return err
		}
	}
	return nil
}

func (g *Generator) generateMethodStatement(className string, method *ast.MethodStatement) error {
	// Receiver is first parameter
	var params []string
	var receiver string
	if len(method.Parameters) > 0 {
		receiver = method.Parameters[0].Value
		for _, p := range method.Parameters[1:] {
			params = append(params, fmt.Sprintf("%s interface{}", p.Value))
		}
	} else {
		receiver = "self"
	}

	funcName := method.Name.Value
	paramStr := strings.Join(params, ", ")
	if paramStr != "" {
		paramStr = ", " + paramStr
	}

	g.functions.WriteString(fmt.Sprintf("func (%s *%s) %s(%s) interface{} {\n", receiver, className, funcName, paramStr))

	// Generate method body
	for _, s := range method.Body.Statements {
		s, err := g.generateStatementString(s)
		if err != nil {
			return err
		}
		lines := strings.Split(s, "\n")
		for _, line := range lines {
			if line != "" {
				g.functions.WriteString("\t" + line + "\n")
			} else {
				g.functions.WriteString("\n")
			}
		}
	}

	g.functions.WriteString("\treturn nil\n")
	g.functions.WriteString("}\n\n")
	return nil
}

func (g *Generator) generateForStatement(stmt *ast.ForStatement) error {
	iter, err := g.generateExpression(stmt.Iterator)
	if err != nil {
		return err
	}
	iterable, err := g.generateExpression(stmt.Iterable)
	if err != nil {
		return err
	}

	g.mainBody.WriteString(fmt.Sprintf("\tfor %s := 0; %s < len(%s); %s++ {\n", iter, iter, iterable, iter))

	// Generate loop body with extra indentation relative to loop
	for _, s := range stmt.Body.Statements {
		s, err := g.generateStatementString(s)
		if err != nil {
			return err
		}
		lines := strings.Split(s, "\n")
		for _, line := range lines {
			if line != "" {
				g.mainBody.WriteString("\t\t" + line + "\n")
			} else {
				g.mainBody.WriteString("\n")
			}
		}
	}

	g.mainBody.WriteString("\t}\n")
	return nil
}

func (g *Generator) generateStatementString(stmt ast.Statement) (string, error) {
	switch s := stmt.(type) {
	case *ast.ExpressionStatement:
		return g.generateExpression(s.Expression)
	case *ast.LetStatement:
		val, err := g.generateExpression(s.Value)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s := %s", s.Name.Value, val), nil
	case *ast.ReturnStatement:
		if s.ReturnValue == nil {
			return "return", nil
		}
		val, err := g.generateExpression(s.ReturnValue)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("return %s", val), nil
	case *ast.AssignmentStatement:
		name, err := g.generateExpression(s.Name)
		if err != nil {
			return "", err
		}
		val, err := g.generateExpression(s.Value)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s = %s", name, val), nil
	case *ast.IfExpression:
		// This shouldn't normally appear as a standalone statement string, but handle it.
		return g.generateExpression(s)
	default:
		return "", fmt.Errorf("unsupported statement type for string generation: %T", stmt)
	}
}

func (g *Generator) generateExpression(expr ast.Expression) (string, error) {
	switch e := expr.(type) {
	case *ast.IntegerLiteral:
		return fmt.Sprintf("%d", e.Value), nil
	case *ast.StringLiteral:
		return fmt.Sprintf("\"%s\"", e.Value), nil
	case *ast.Boolean:
		if e.Value {
			return "true", nil
		}
		return "false", nil
	case *ast.Identifier:
		return e.Value, nil
	case *ast.PrefixExpression:
		right, err := g.generateExpression(e.Right)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("(%s%s)", e.Operator, right), nil
	case *ast.InfixExpression:
		left, err := g.generateExpression(e.Left)
		if err != nil {
			return "", err
		}
		right, err := g.generateExpression(e.Right)
		if err != nil {
			return "", err
		}
		// For arithmetic, cast both sides to int64 to work with interface{} values
		switch e.Operator {
		case "+", "-", "*", "/":
			return fmt.Sprintf("((%s.(int64)) %s (%s.(int64)))", left, e.Operator, right), nil
		default:
			return fmt.Sprintf("(%s %s %s)", left, e.Operator, right), nil
		}
	case *ast.CallExpression:
		funcStr, err := g.generateExpression(e.Function)
		if err != nil {
			return "", err
		}
		if funcStr == "print" {
			var args []string
			for _, arg := range e.Arguments {
				a, err := g.generateExpression(arg)
				if err != nil {
					return "", err
				}
				args = append(args, a)
			}
			return fmt.Sprintf("fmt.Println(%s)", strings.Join(args, ", ")), nil
		}
		var args []string
		for _, arg := range e.Arguments {
			a, err := g.generateExpression(arg)
			if err != nil {
				return "", err
			}
			args = append(args, a)
		}
		return fmt.Sprintf("%s(%s)", funcStr, strings.Join(args, ", ")), nil
	case *ast.ArrayLiteral:
		var elements []string
		for _, el := range e.Elements {
			elem, err := g.generateExpression(el)
			if err != nil {
				return "", err
			}
			elements = append(elements, elem)
		}
		return fmt.Sprintf("[]interface{}{%s}", strings.Join(elements, ", ")), nil
	case *ast.IndexExpression:
		left, err := g.generateExpression(e.Left)
		if err != nil {
			return "", err
		}
		index, err := g.generateExpression(e.Index)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s.([]interface{})[%s]", left, index), nil
	case *ast.IfExpression:
		cond, err := g.generateExpression(e.Condition)
		if err != nil {
			return "", err
		}
		var b bytes.Buffer
		b.WriteString(fmt.Sprintf("if (%s) {\n", cond))
		consequence, err := g.generateBlockStatementString(e.Consequence)
		if err != nil {
			return "", err
		}
		b.WriteString(consequence)
		if e.Alternative != nil {
			alt, err := g.generateBlockStatementString(e.Alternative)
			if err != nil {
				return "", err
			}
			b.WriteString("} else {\n")
			b.WriteString(alt)
			b.WriteString("}\n")
		} else {
			b.WriteString("}\n")
		}
		return b.String(), nil
	case *ast.FunctionLiteral:
		// Function literal as an expression (not assigned to variable)
		// This is unusual; just generate a placeholder or handle as anonymous function.
		// For simplicity, return an error or generate an anonymous Go function.
		// Given the tests, we mainly handle FunctionLiteral via generateLetStatement.
		var params []string
		for _, p := range e.Parameters {
			params = append(params, fmt.Sprintf("%s interface{}", p.Value))
		}
		funcName := "anonymous"
		bodyStr := ""
		if e.Body != nil {
			var bodyBuf bytes.Buffer
			for _, s := range e.Body.Statements {
				str, err := g.generateStatementString(s)
				if err != nil {
					return "", err
				}
				bodyBuf.WriteString("\t" + str + "\n")
			}
			bodyStr = bodyBuf.String()
		}
		paramStr := strings.Join(params, ", ")
		return fmt.Sprintf("func %s(%s) interface{} {\n%s}\n", funcName, paramStr, bodyStr), nil
	case *ast.DotExpression:
		left, err := g.generateExpression(e.Left)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s.%s", left, e.Right.Value), nil
	default:
		return "", fmt.Errorf("unsupported expression type: %T", expr)
	}
}

func (g *Generator) generateBlockStatementString(block *ast.BlockStatement) (string, error) {
	if block == nil {
		return "", nil
	}
	var b bytes.Buffer
	for _, stmt := range block.Statements {
		s, err := g.generateStatementString(stmt)
		if err != nil {
			return "", err
		}
		lines := strings.Split(s, "\n")
		for _, line := range lines {
			if line != "" {
				b.WriteString("\t" + line + "\n")
			} else {
				b.WriteString("\n")
			}
		}
	}
	return b.String(), nil
}

func (g *Generator) generateFunctionDef(name string, funcLit *ast.FunctionLiteral) error {
	var params []string
	for _, p := range funcLit.Parameters {
		params = append(params, fmt.Sprintf("%s interface{}", p.Value))
	}
	paramStr := strings.Join(params, ", ")

	g.functions.WriteString(fmt.Sprintf("func %s(%s) interface{} {\n", name, paramStr))

	if funcLit.Body != nil {
		for _, s := range funcLit.Body.Statements {
			s, err := g.generateStatementString(s)
			if err != nil {
				return err
			}
			lines := strings.Split(s, "\n")
			for _, line := range lines {
				if line != "" {
					g.functions.WriteString("\t" + line + "\n")
				} else {
					g.functions.WriteString("\n")
				}
			}
		}
	}

	g.functions.WriteString("}\n\n")
	return nil
}
