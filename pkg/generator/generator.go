package generator

import (
	"bytes"
	"fmt"
	"gopy/pkg/ast"
	"strings"
)

type Generator struct {
	// Мы будем хранить сгенерированные функции отдельно от основного кода
	functions bytes.Buffer
	mainBody  bytes.Buffer
	// Нам все еще нужно отслеживать переменные, чтобы использовать = или :=
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
		return "", fmt.Errorf("неподдерживаемый тип узла: %T", node)
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
	out.WriteString(g.functions.String()) // Сначала все функции
	out.WriteString("func main() {\n")
	out.WriteString(g.mainBody.String()) // Затем тело main
	out.WriteString("}\n")

	return out.String(), nil
}

func (g *Generator) generateStatement(stmt ast.Statement) error {
	switch stmt := stmt.(type) {
	case *ast.AssignmentStatement:
		switch name := stmt.Name.(type) {
		case *ast.DotExpression:
			left, err := g.generateExpression(name.Left)
			if err != nil {
				return err
			}
			right := name.Right.Value
			val, err := g.generateExpressionWithCast(stmt.Value, false, false)
			if err != nil {
				return err
			}
			g.mainBody.WriteString(fmt.Sprintf("\t%s.%s = %s\n", left, right, val))
			return nil
		case *ast.Identifier:
			// как раньше
			if call, ok := stmt.Value.(*ast.CallExpression); ok {
				if ident, ok := call.Function.(*ast.Identifier); ok && len(call.Arguments) == 0 && g.isClass(ident.Value) {
					if !g.declaredVariables[name.Value] {
						g.declaredVariables[name.Value] = true
						g.mainBody.WriteString(fmt.Sprintf("\t%s := &%s{}\n", name.Value, ident.Value))
					} else {
						g.mainBody.WriteString(fmt.Sprintf("\t%s = &%s{}\n", name.Value, ident.Value))
					}
					return nil
				}
			}
			val, err := g.generateExpressionWithCast(stmt.Value, false, false)
			if err != nil {
				return err
			}
			if !g.declaredVariables[name.Value] {
				g.declaredVariables[name.Value] = true
				g.mainBody.WriteString(fmt.Sprintf("\t%s := %s\n", name.Value, val))
			} else {
				g.mainBody.WriteString(fmt.Sprintf("\t%s = %s\n", name.Value, val))
			}
			return nil
		}
	case *ast.ClassStatement:
		return g.generateClass(stmt)
	case *ast.LetStatement:
		// Если присваиваем класс, создаём объект через &Class{}
		if call, ok := stmt.Value.(*ast.CallExpression); ok {
			if ident, ok := call.Function.(*ast.Identifier); ok && len(call.Arguments) == 0 && g.isClass(ident.Value) {
				g.declaredVariables[stmt.Name.Value] = true
				g.mainBody.WriteString(fmt.Sprintf("\t%s := &%s{}\n", stmt.Name.Value, ident.Value))
				return nil
			}
		}
		// Если мы присваиваем функцию, генерируем ее отдельно
		if fn, ok := stmt.Value.(*ast.FunctionLiteral); ok {
			return g.generateFunction(stmt.Name.Value, fn)
		}
		// Иначе это обычная переменная
		val, err := g.generateExpressionWithCast(stmt.Value, false, false)
		if err != nil {
			return err
		}
		g.declaredVariables[stmt.Name.Value] = true
		g.mainBody.WriteString(fmt.Sprintf("\t%s := %s\n", stmt.Name.Value, val))

	case *ast.ForStatement:
		return g.generateForStatement(stmt)

	case *ast.ExpressionStatement:
		// d.bark() — CallExpression с DotExpression
		if call, ok := stmt.Expression.(*ast.CallExpression); ok {
			if dot, ok := call.Function.(*ast.DotExpression); ok {
				left, err := g.generateExpressionWithCast(dot.Left, false, true)
				if err != nil {
					return err
				}
				right := dot.Right.Value
				args := []string{}
				for _, a := range call.Arguments {
					astr, err := g.generateExpressionWithCast(a, false, true)
					if err != nil {
						return err
					}
					args = append(args, astr)
				}
				g.mainBody.WriteString(fmt.Sprintf("\t%s.%s(%s)\n", left, right, strings.Join(args, ", ")))
				return nil
			}
		}
		// d.name — DotExpression
		if dot, ok := stmt.Expression.(*ast.DotExpression); ok {
			left, err := g.generateExpressionWithCast(dot.Left, false, false)
			if err != nil {
				return err
			}
			right := dot.Right.Value
			g.mainBody.WriteString(fmt.Sprintf("\t%s.%s\n", left, right))
			return nil
		}
		exprStr, err := g.generateExpressionWithCast(stmt.Expression, false, false)
		if err != nil {
			return err
		}
		g.mainBody.WriteString("\t" + exprStr + "\n")
		return nil

	default:
		// Другие инструкции (return, if и т.д.) пока будут работать только внутри main
		str, err := g.generateSimpleStatement(stmt)
		if err != nil {
			return err
		}
		g.mainBody.WriteString("\t" + str + "\n")
	}
	return nil
}

// generateFunction генерирует код для функции верхнего уровня
func (g *Generator) generateFunction(name string, fn *ast.FunctionLiteral) error {
	var params []string
	for _, p := range fn.Parameters {
		// В Go нам нужны типы. Пока что для простоты используем interface{}
		params = append(params, fmt.Sprintf("%s interface{}", p.Value))
	}

	// Начинаем объявление функции
	g.functions.WriteString(fmt.Sprintf("func %s(%s) interface{} {\n", name, strings.Join(params, ", ")))

	// Генерируем тело функции
	body, err := g.generateBlockStatementWithCast(fn.Body, true)
	if err != nil {
		return err
	}
	g.functions.WriteString(body)

	// Если в функции нет return, Go требует его для функций, возвращающих значение
	if !strings.Contains(body, "return") {
		g.functions.WriteString("\treturn nil\n")
	}

	g.functions.WriteString("}\n\n")
	return nil
}

// generateSimpleStatement - это старый generateStatement для простых случаев
func (g *Generator) generateSimpleStatement(stmt ast.Statement) (string, error) {
	switch stmt := stmt.(type) {
	case *ast.ReturnStatement:
		val, err := g.generateExpressionWithCast(stmt.ReturnValue, false, false)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("return %s", val), nil
	// Другие простые инструкции можно добавить сюда
	default:
		return "", fmt.Errorf("неподдерживаемый тип инструкции для generateSimpleStatement: %T", stmt)
	}
}

func (g *Generator) generateExpression(expr ast.Expression) (string, error) {
	return g.generateExpressionWithCast(expr, false, false)
}

// generateExpressionWithCast добавляет приведение к int64 только если inFunction==true
func (g *Generator) generateExpressionWithCast(expr ast.Expression, inFunction bool, isFunctionCall bool) (string, error) {
	switch expr := expr.(type) {
	case *ast.Identifier:
		return expr.Value, nil
	case *ast.IntegerLiteral:
		return fmt.Sprintf("%d", expr.Value), nil
	case *ast.StringLiteral:
		return fmt.Sprintf("\"%s\"", expr.Value), nil
	case *ast.Boolean:
		return fmt.Sprintf("%t", expr.Value), nil
	case *ast.PrefixExpression:
		right, err := g.generateExpressionWithCast(expr.Right, inFunction, isFunctionCall)
		if err != nil {
			return "", err
		}
		if expr.Operator == "not" {
			return fmt.Sprintf("(!%s)", right), nil
		}
		return fmt.Sprintf("(%s%s)", expr.Operator, right), nil
	case *ast.InfixExpression:
		left, err := g.generateExpressionWithCast(expr.Left, inFunction, false)
		if err != nil {
			return "", err
		}
		right, err := g.generateExpressionWithCast(expr.Right, inFunction, false)
		if err != nil {
			return "", err
		}
		if inFunction {
			left = fmt.Sprintf("(%s.(int64))", left)
			right = fmt.Sprintf("(%s.(int64))", right)
		}
		// Обработка логических операторов
		switch expr.Operator {
		case "and":
			return fmt.Sprintf("(%s && %s)", left, right), nil
		case "or":
			return fmt.Sprintf("(%s || %s)", left, right), nil
		default:
			return fmt.Sprintf("(%s %s %s)", left, expr.Operator, right), nil
		}
	case *ast.ArrayLiteral:
		elements := []string{}
		for _, el := range expr.Elements {
			str, err := g.generateExpressionWithCast(el, inFunction, false)
			if err != nil {
				return "", err
			}
			elements = append(elements, str)
		}
		return fmt.Sprintf("[]interface{}{%s}", strings.Join(elements, ", ")), nil
	case *ast.IndexExpression:
		left, err := g.generateExpressionWithCast(expr.Left, inFunction, false)
		if err != nil {
			return "", err
		}
		index, err := g.generateExpressionWithCast(expr.Index, inFunction, false)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("%s[%s]", left, index), nil
	case *ast.CallExpression:
		var args []string
		for _, arg := range expr.Arguments {
			a, err := g.generateExpressionWithCast(arg, inFunction, false)
			if err != nil {
				return "", err
			}
			args = append(args, a)
		}
		// Специальный случай для нашей встроенной функции print
		if expr.Function.String() == "print" {
			return fmt.Sprintf("fmt.Println(%s)", strings.Join(args, ", ")), nil
		}
		// Обычный вызов функции
		return fmt.Sprintf("%s(%s)", expr.Function.String(), strings.Join(args, ", ")), nil
	case *ast.IfExpression:
		condition, err := g.generateExpressionWithCast(expr.Condition, inFunction, false)
		if err != nil {
			return "", err
		}
		consequence, err := g.generateBlockStatementWithCast(expr.Consequence, inFunction)
		if err != nil {
			return "", err
		}
		code := fmt.Sprintf("if %s {\n%s}", condition, consequence)
		if expr.Alternative != nil {
			alternative, err := g.generateBlockStatementWithCast(expr.Alternative, inFunction)
			if err != nil {
				return "", err
			}
			code += fmt.Sprintf(" else {\n%s}", alternative)
		}
		return code, nil
	case *ast.DotExpression:
		left, err := g.generateExpressionWithCast(expr.Left, inFunction, false)
		if err != nil {
			return "", err
		}
		if expr.Right == nil {
			return "", fmt.Errorf("DotExpression: отсутствует поле/метод после точки")
		}
		// Если DotExpression — часть CallExpression, добавим ()
		if isFunctionCall {
			return fmt.Sprintf("%s.%s()", left, expr.Right.Value), nil
		}
		return fmt.Sprintf("%s.%s", left, expr.Right.Value), nil
	default:
		return "", fmt.Errorf("неподдерживаемый тип выражения: %T", expr)
	}
}

func (g *Generator) generateBlockStatement(block *ast.BlockStatement) (string, error) {
	var out bytes.Buffer
	for _, stmt := range block.Statements {
		switch s := stmt.(type) {
		case *ast.ReturnStatement:
			str, err := g.generateSimpleStatement(s)
			if err != nil {
				return "", err
			}
			out.WriteString("\t" + str + "\n")
		case *ast.LetStatement:
			val, err := g.generateExpression(s.Value)
			if err != nil {
				return "", err
			}
			out.WriteString(fmt.Sprintf("\t%s := %s\n", s.Name.Value, val))
		case *ast.AssignmentStatement:
			switch name := s.Name.(type) {
			case *ast.DotExpression:
				left, err := g.generateExpression(name.Left)
				if err != nil {
					return "", err
				}
				right := name.Right.Value
				val, err := g.generateExpression(s.Value)
				if err != nil {
					return "", err
				}
				out.WriteString(fmt.Sprintf("\t%s.%s = %s\n", left, right, val))
			default:
				val, err := g.generateExpression(s.Value)
				if err != nil {
					return "", err
				}
				out.WriteString(fmt.Sprintf("\t%s = %s\n", name.String(), val))
			}
		case *ast.ExpressionStatement:
			// d.bark() — CallExpression с DotExpression
			if call, ok := s.Expression.(*ast.CallExpression); ok {
				if dot, ok := call.Function.(*ast.DotExpression); ok {
					left, err := g.generateExpression(dot.Left)
					if err != nil {
						return "", err
					}
					right := dot.Right.Value
					args := []string{}
					for _, a := range call.Arguments {
						astr, err := g.generateExpression(a)
						if err != nil {
							return "", err
						}
						args = append(args, astr)
					}
					out.WriteString(fmt.Sprintf("\t%s.%s(%s)\n", left, right, strings.Join(args, ", ")))
					continue
				}
			}
			// d.name — DotExpression
			if dot, ok := s.Expression.(*ast.DotExpression); ok {
				left, err := g.generateExpression(dot.Left)
				if err != nil {
					return "", err
				}
				right := dot.Right.Value
				out.WriteString(fmt.Sprintf("\t%s.%s\n", left, right))
				continue
			}
			exprStr, err := g.generateExpressionWithCast(s.Expression, false, false)
			if err != nil {
				return "", err
			}
			out.WriteString("\t" + exprStr + "\n")
		default:
			return "", fmt.Errorf("неподдерживаемый тип инструкции в блоке: %T", stmt)
		}
	}
	return out.String(), nil
}

// generateBlockStatementWithCast аналогично generateBlockStatement, но с флагом inFunction
func (g *Generator) generateBlockStatementWithCast(block *ast.BlockStatement, inFunction bool) (string, error) {
	var out bytes.Buffer
	for _, stmt := range block.Statements {
		switch s := stmt.(type) {
		case *ast.ReturnStatement:
			str, err := g.generateSimpleStatementWithCast(s, inFunction, false)
			if err != nil {
				return "", err
			}
			out.WriteString("\t" + str + "\n")
		case *ast.LetStatement:
			val, err := g.generateExpressionWithCast(s.Value, inFunction, false)
			if err != nil {
				return "", err
			}
			out.WriteString(fmt.Sprintf("\t%s := %s\n", s.Name.Value, val))
		case *ast.AssignmentStatement:
			switch name := s.Name.(type) {
			case *ast.DotExpression:
				left, err := g.generateExpressionWithCast(name.Left, inFunction, false)
				if err != nil {
					return "", err
				}
				right := name.Right.Value
				val, err := g.generateExpressionWithCast(s.Value, inFunction, false)
				if err != nil {
					return "", err
				}
				out.WriteString(fmt.Sprintf("\t%s.%s = %s\n", left, right, val))
			default:
				val, err := g.generateExpressionWithCast(s.Value, inFunction, false)
				if err != nil {
					return "", err
				}
				out.WriteString(fmt.Sprintf("\t%s = %s\n", name.String(), val))
			}
		case *ast.ExpressionStatement:
			// self.bark() — CallExpression с DotExpression
			if call, ok := s.Expression.(*ast.CallExpression); ok {
				if dot, ok := call.Function.(*ast.DotExpression); ok {
					left, err := g.generateExpressionWithCast(dot.Left, inFunction, true)
					if err != nil {
						return "", err
					}
					right := dot.Right.Value
					args := []string{}
					for _, a := range call.Arguments {
						astr, err := g.generateExpressionWithCast(a, inFunction, true)
						if err != nil {
							return "", err
						}
						args = append(args, astr)
					}
					out.WriteString(fmt.Sprintf("\t%s.%s(%s)\n", left, right, strings.Join(args, ", ")))
					continue
				}
			}
			// self.name — DotExpression
			if dot, ok := s.Expression.(*ast.DotExpression); ok {
				left, err := g.generateExpressionWithCast(dot.Left, inFunction, false)
				if err != nil {
					return "", err
				}
				right := dot.Right.Value
				out.WriteString(fmt.Sprintf("\t%s.%s\n", left, right))
				continue
			}
			exprStr, err := g.generateExpressionWithCast(s.Expression, inFunction, false)
			if err != nil {
				return "", err
			}
			out.WriteString("\t" + exprStr + "\n")
		default:
			return "", fmt.Errorf("неподдерживаемый тип инструкции в блоке: %T", stmt)
		}
	}
	return out.String(), nil
}

// generateSimpleStatementWithCast аналогично generateSimpleStatement, но с флагом inFunction
func (g *Generator) generateSimpleStatementWithCast(stmt ast.Statement, inFunction bool, isFunctionCall bool) (string, error) {
	switch stmt := stmt.(type) {
	case *ast.ReturnStatement:
		val, err := g.generateExpressionWithCast(stmt.ReturnValue, inFunction, isFunctionCall)
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("return %s", val), nil
	default:
		return "", fmt.Errorf("неподдерживаемый тип инструкции для generateSimpleStatement: %T", stmt)
	}
}

// generateClass генерирует Go-структуру и методы для класса
func (g *Generator) generateClass(class *ast.ClassStatement) error {
	// Структура с полями
	fields := []string{}
	for _, f := range class.Fields {
		fields = append(fields, fmt.Sprintf("%s interface{}", f.Value))
	}
	g.functions.WriteString(fmt.Sprintf("type %s struct{%s}\n\n", class.Name.Value, strings.Join(fields, "; ")))
	g.declaredClasses[class.Name.Value] = true
	// Методы
	for _, m := range class.Methods {
		if err := g.generateMethod(class.Name.Value, m); err != nil {
			return err
		}
	}
	return nil
}

// generateMethod генерирует Go-метод для структуры
func (g *Generator) generateMethod(className string, m *ast.MethodStatement) error {
	var params []string
	for i, p := range m.Parameters {
		if i == 0 {
			continue // self
		}
		params = append(params, fmt.Sprintf("%s interface{}", p.Value))
	}
	// self всегда первый параметр
	g.functions.WriteString(fmt.Sprintf("func (self *%s) %s(%s) interface{} {\n", className, m.Name.Value, strings.Join(params, ", ")))
	body, err := g.generateBlockStatementWithCast(m.Body, true)
	if err != nil {
		return err
	}
	g.functions.WriteString(body)
	if !strings.Contains(body, "return") {
		g.functions.WriteString("\treturn nil\n")
	}
	g.functions.WriteString("}\n\n")
	return nil
}

// generateForStatement генерирует Go-код для цикла for
func (g *Generator) generateForStatement(stmt *ast.ForStatement) error {
	iterable, err := g.generateExpressionWithCast(stmt.Iterable, false, false)
	if err != nil {
		return err
	}

	// Предполагаем, что Iterable - это IntegerLiteral для range(N)
	// В будущем здесь будет более сложная логика для итерируемых объектов
	// Пока что генерируем простой for-цикл Go
	g.mainBody.WriteString(fmt.Sprintf("\tfor %s := 0; %s < %s; %s++ {\n", stmt.Iterator.Value, stmt.Iterator.Value, iterable, stmt.Iterator.Value))

	body, err := g.generateBlockStatementWithCast(stmt.Body, false)
	if err != nil {
		return err
	}
	g.mainBody.WriteString(body)
	g.mainBody.WriteString("\t}\n")

	return nil
}

// isClass проверяет, объявлен ли класс с таким именем
func (g *Generator) isClass(name string) bool {
	return g.declaredClasses[name]
}


