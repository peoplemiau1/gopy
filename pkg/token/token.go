package token

// TokenType представляет тип токена/лексемы
type TokenType string

// Token представляет лексему
type Token struct {
	Type    TokenType
	Literal string
}

const (
	ILLEGAL = "ILLEGAL" // Неизвестный токен
	EOF     = "EOF"     // Конец файла

	// Идентификаторы + литералы
	IDENT = "IDENT" // Имена переменных, функций и т.д.
	INT   = "INT"   // Целые числа
	STRING = "STRING" // Строки
	DOT         = "."

	// Операторы
	ASSIGN   = "="
	PLUS     = "+"
	MINUS    = "-"
	BANG     = "!"
	ASTERISK = "*"
	SLASH    = "/"
	LT       = "<"
	GT       = ">"
	EQ       = "=="
	NOT_EQ   = "!="
	AND      = "AND"
	OR       = "OR"
	NOT      = "NOT"

	// Разделители
	COMMA     = ","
	LPAREN    = "("
	RPAREN    = ")"
	LBRACKET  = "["
	RBRACKET  = "]"
	NEWLINE   = "NEWLINE" // Новый разделитель - новая строка

	// Отступы
	INDENT = "INDENT"
	DEDENT = "DEDENT"

	// Ключевые слова
	TRUE     = "TRUE"
	FALSE    = "FALSE"
	IF       = "IF"
	ELSE     = "ELSE"
	RETURN   = "RETURN"
	CLASS    = "CLASS"
	FOR      = "FOR"
	IN       = "IN"
	DEF 	 = "DEF"
	PRINT 	 = "PRINT"
	LET      = "LET"
)

var keywords = map[string]TokenType{
	"def":    DEF,
	"true":   TRUE,
	"false":  FALSE,
	"if":     IF,
	"else":   ELSE,
	"return": RETURN,
	"class":  CLASS,
	"for":    FOR,
	"in":     IN,
	"print":  PRINT,
	"let":    LET,
	"and":    AND,
	"or":     OR,
	"not":    NOT,
}

// LookupIdent проверяет, является ли идентификатор ключевым словом
func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
