package main

import (
	"fmt"
	"gopy/generator"
	"gopy/lexer"
	"gopy/parser"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Использование: gopy <файл.gopy>")
		os.Exit(1)
	}

	inputFile := os.Args[1]
	content, err := ioutil.ReadFile(inputFile)
	if err != nil {
		fmt.Printf("Ошибка чтения файла %s: %s\n", inputFile, err)
		os.Exit(1)
	}

	l := lexer.New(string(content))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		fmt.Println("Ошибки парсинга:")
		for _, msg := range p.Errors() {
			fmt.Println("\t" + msg)
		}
		os.Exit(1)
	}

	gen := generator.New()
	generatedCode, err := gen.Generate(program)
	if err != nil {
		fmt.Printf("Ошибка генерации кода: %s\n", err)
		os.Exit(1)
	}

	// Сохраняем сгенерированный Go-код во временный файл
	goFile, err := ioutil.TempFile("", "gopy-*.go")
	if err != nil {
		fmt.Printf("Ошибка создания временного файла: %s\n", err)
		os.Exit(1)
	}
	defer os.Remove(goFile.Name()) // Очищаем после себя

	if _, err := goFile.WriteString(generatedCode); err != nil {
		fmt.Printf("Ошибка записи во временный файл: %s\n", err)
		os.Exit(1)
	}
	goFile.Close()

	// Компилируем Go-файл
	outputExe := strings.TrimSuffix(inputFile, filepath.Ext(inputFile)) + ".exe"
	cmd := exec.Command("go", "build", "-o", outputExe, goFile.Name())
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Ошибка компиляции Go-кода: %s\n", err)
		os.Exit(1)
	}

	fmt.Printf("\n--- Gopy: Компиляция %s завершена. Запуск %s ---\n\n", inputFile, outputExe)

	// Запускаем скомпилированный exe
	cmdRun := exec.Command("./" + outputExe)
	cmdRun.Stdout = os.Stdout
	cmdRun.Stderr = os.Stderr
	if err := cmdRun.Run(); err != nil {
		fmt.Printf("\n--- Gopy: Ошибка выполнения %s: %s ---\n", outputExe, err)
		os.Exit(1)
	}

	fmt.Printf("\n--- Gopy: Выполнение %s завершено ---\n", outputExe)
}