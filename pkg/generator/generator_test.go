package generator

import (
	"gopy/lexer"
	"gopy/parser"
	"testing"
)

func TestLetStatementGeneration(t *testing.T) {
	input := `let myVar = 123`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	gen := New()
	generatedCode, err := gen.Generate(program)
	if err != nil {
		t.Fatalf("Code generation failed: %s", err)
	}

	expectedCode := `package main

import (
	"fmt"
)

func main() {
	myVar := 123
}
`
	if generatedCode != expectedCode {
		t.Errorf("Generated code is wrong.\nExpected:\n%s\nGot:\n%s", expectedCode, generatedCode)
	}
}

func TestReturnStatementGeneration(t *testing.T) {
	input := `return 5`

	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	gen := New()
	generatedCode, err := gen.Generate(program)
	if err != nil {
		t.Fatalf("Code generation failed: %s", err)
	}

	expectedCode := `package main

import (
	"fmt"
)

func main() {
	return 5
}
`
	if generatedCode != expectedCode {
		t.Errorf("Generated code is wrong.\nExpected:\n%s\nGot:\n%s", expectedCode, generatedCode)
	}
}

func TestBooleanGeneration(t *testing.T) {
	tests := []struct {
		input        string
		expectedCode string
	}{
		{
			"let x = true",
			`package main

import (
	"fmt"
)

func main() {
	x := true
}
`,
		},
		{
			"let y = false",
			`package main

import (
	"fmt"
)

func main() {
	y := false
}
`,
		},
	}

	for _, tt := range tests {
		l := lexer.New(tt.input)
		p := parser.New(l)
		program := p.ParseProgram()

		gen := New()
		generatedCode, err := gen.Generate(program)
		if err != nil {
			t.Fatalf("Code generation failed: %s", err)
		}

		if generatedCode != tt.expectedCode {
			t.Errorf("Generated code is wrong for input '%s'.\nExpected:\n%s\nGot:\n%s", tt.input, tt.expectedCode, generatedCode)
		}
	}
}

func TestIfElseGeneration(t *testing.T) {
	input := `
if 1 > 0
	print("greater")
else
	print("smaller")
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()

	gen := New()
	generatedCode, err := gen.Generate(program)
	if err != nil {
		t.Fatalf("Code generation failed: %s", err)
	}

	expectedCode := `package main

import (
	"fmt"
)

func main() {
	if (1 > 0) {
	fmt.Println("greater")
} else {
	fmt.Println("smaller")
}
}
`
	if generatedCode != expectedCode {
		t.Errorf("Generated code is wrong.\nExpected:\n%s\nGot:\n%s", expectedCode, generatedCode)
	}
}

func TestFunctionGeneration(t *testing.T) {
	input := `
let add = def(a, b)
	return a + b

let result = add(5, 10)
print(result)
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	checkParserErrors(t, p)

	gen := New()
	generatedCode, err := gen.Generate(program)
	if err != nil {
		t.Fatalf("Code generation failed: %s", err)
	}

	expectedCode := `package main

import (
	"fmt"
)

func add(a interface{}, b interface{}) interface{} {
	return ((a.(int64)) + (b.(int64)))
}

func main() {
	result := add(5, 10)
	fmt.Println(result)
}
`
	if generatedCode != expectedCode {
		t.Errorf("Generated code is wrong.\nExpected:\n%s\nGot:\n%s", expectedCode, generatedCode)
	}
}

func TestClassGeneration(t *testing.T) {
	input := `
class Dog
    def bark(self)
        print("гав!")
let d = Dog()
d.bark()
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	gen := New()
	generatedCode, err := gen.Generate(program)
	if err != nil {
		t.Fatalf("Code generation failed: %s", err)
	}
	expectedCode := `package main

import (
	"fmt"
)

type Dog struct{}

func (self *Dog) bark() interface{} {
	fmt.Println("гав!")
	return nil
}

func main() {
	d := &Dog{}
	d.bark()
}
`
	if generatedCode != expectedCode {
		t.Errorf("Generated code is wrong.\nExpected:\n%s\nGot:\n%s", expectedCode, generatedCode)
	}
}

func TestClassWithFieldsGeneration(t *testing.T) {
	input := `
class Dog name age
    def bark(self)
        print(self.name)
        print(self.age)
let d = Dog()
d.name = "Шарик"
d.age = 5
d.bark()
`
	l := lexer.New(input)
	p := parser.New(l)
	program := p.ParseProgram()
	if len(p.Errors()) > 0 {
		t.Fatalf("Parser errors: %v", p.Errors())
	}
	gen := New()
	generatedCode, err := gen.Generate(program)
	if err != nil {
		t.Fatalf("Code generation failed: %s", err)
	}
	expectedCode := `package main

import (
	"fmt"
)

type Dog struct{name interface{}; age interface{}}

func (self *Dog) bark() interface{} {
	fmt.Println(self.name)
	fmt.Println(self.age)
	return nil
}

func main() {
	d := &Dog{}
	d.name = "Шарик"
	d.age = 5
	d.bark()
}
`
	if generatedCode != expectedCode {
		t.Errorf("Generated code is wrong.\nExpected:\n%s\nGot:\n%s", expectedCode, generatedCode)
	}
}

func checkParserErrors(t *testing.T, p *parser.Parser) {
	errors := p.Errors()
	if len(errors) == 0 {
		return
	}

	t.Errorf("parser has %d errors", len(errors))
	for _, msg := range errors {
		t.Errorf("parser error: %q", msg)
	}
	t.FailNow()
} 