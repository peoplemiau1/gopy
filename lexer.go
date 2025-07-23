package lexer

import (
	"gopy/token"
)

// Lexer преобразует исходный код в токены
type Lexer struct {
	input        string
	position     int  // текущая позиция в input (указывает на текущий символ)
	readPosition int  // следующая позиция для чтения (после текущего символа)
	ch           byte // текущий символ

	// Для обработки отступов
	indentStack []int // Стек для отслеживания уровней отступов
	pendingTokens []token.Token // Токены, ожидающие выдачи (INDENT/DEDENT)
}

// New создает новый экземпляр Lexer
func New(input string) *Lexer {
	l := &Lexer{input: input, indentStack: []int{0}}
	l.readChar()
	return l
}

func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPosition]
	}
	l.position = l.readPosition
	l.readPosition++
}

// NextToken возвращает следующий токен
func (l *Lexer) NextToken() token.Token {
	// Если есть ожидающие токены (INDENT/DEDENT), выдаем их первыми
	if len(l.pendingTokens) > 0 {
		tok := l.pendingTokens[0]
		l.pendingTokens = l.pendingTokens[1:]
		
		return tok
	}

	l.skipWhitespaceAndComments()

	var tok token.Token

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.EQ, Literal: literal}
		} else {
			tok = newToken(token.ASSIGN, l.ch)
		}
	case '+':
		tok = newToken(token.PLUS, l.ch)
	case '-':
		tok = newToken(token.MINUS, l.ch)
	case '!':
		if l.peekChar() == '=' {
			ch := l.ch
			l.readChar()
			literal := string(ch) + string(l.ch)
			tok = token.Token{Type: token.NOT_EQ, Literal: literal}
		} else {
			tok = newToken(token.BANG, l.ch)
		}
	case '/':
		tok = newToken(token.SLASH, l.ch)
	case '*':
		tok = newToken(token.ASTERISK, l.ch)
	case '<':
		tok = newToken(token.LT, l.ch)
	case '>':
		tok = newToken(token.GT, l.ch)
	case ',':
		tok = newToken(token.COMMA, l.ch)
	case '(':
		tok = newToken(token.LPAREN, l.ch)
	case ')':
		tok = newToken(token.RPAREN, l.ch)
	case '[':
		tok = newToken(token.LBRACKET, l.ch)
	case ']':
		tok = newToken(token.RBRACKET, l.ch)
	case '\n':
		tok = newToken(token.NEWLINE, l.ch)
		l.readChar() // Consume the newline character
		// Now, measure the indent of the next line and queue INDENT/DEDENT tokens
		currentLineStart := l.position
		currentIndent := l.measureIndent(currentLineStart)

		lastIndent := l.indentStack[len(l.indentStack)-1]

		if currentIndent > lastIndent {
			l.indentStack = append(l.indentStack, currentIndent)
			l.pendingTokens = append(l.pendingTokens, newToken(token.INDENT, 0))
		} else if currentIndent < lastIndent {
			for currentIndent < l.indentStack[len(l.indentStack)-1] {
				l.indentStack = l.indentStack[:len(l.indentStack)-1]
				l.pendingTokens = append(l.pendingTokens, newToken(token.DEDENT, 0))
			}
		}
		
		return tok // Always return the NEWLINE token immediately
	case '.':
		tok = newToken(token.DOT, l.ch)
	case '"':
		tok.Type = token.STRING
		tok.Literal = l.readString()
	case 0:
		// At the end of the file, if there are unclosed indents, emit DEDENTs
		for len(l.indentStack) > 1 {
			l.indentStack = l.indentStack[:len(l.indentStack)-1]
			l.pendingTokens = append(l.pendingTokens, newToken(token.DEDENT, 0))
		}
		if len(l.pendingTokens) > 0 {
			tok := l.pendingTokens[0]
			l.pendingTokens = l.pendingTokens[1:]
			
			return tok
		}
		tok.Literal = ""
		tok.Type = token.EOF
		
		return tok
	default:
		if isLetter(l.ch) {
			tok.Literal = l.readIdentifier()
			tok.Type = token.LookupIdent(tok.Literal)
			
			return tok
		} else if isDigit(l.ch) {
			tok.Type = token.INT
			tok.Literal = l.readNumber()
			
			return tok
		} else {
			tok = newToken(token.ILLEGAL, l.ch)
			
		}
	}

	
	return tok
}

func (l *Lexer) skipWhitespaceAndComments() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '#' {
		if l.ch == '#' {
			for l.ch != '\n' && l.ch != 0 {
				l.readChar()
			}
		} else {
			l.readChar()
		}
	}
}

func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readNumber() string {
	position := l.position
	for isDigit(l.ch) {
		l.readChar()
	}
	return l.input[position:l.position]
}

func (l *Lexer) readString() string {
	position := l.position + 1
	for {
		l.readChar()
		if l.ch == '"' || l.ch == 0 {
			break
		}
	}
	return l.input[position:l.position]
}

func (l *Lexer) peekChar() byte {
	if l.readPosition >= len(l.input) {
		return 0
	}
	return l.input[l.readPosition]
}

func (l *Lexer) measureIndent(start int) int {
	indent := 0
	for i := start; i < len(l.input); i++ {
		ch := l.input[i]
		if ch == ' ' {
			indent++
		} else if ch == '\t' {
			indent += 4 // Предполагаем, что табуляция = 4 пробела
		} else {
			break
		}
	}
	return indent
}

func isLetter(ch byte) bool {
	return 'a' <= ch && ch <= 'z' || 'A' <= ch && ch <= 'Z' || ch == '_'
}

func isDigit(ch byte) bool {
	return '0' <= ch && ch <= '9'
}

func newToken(tokenType token.TokenType, ch byte) token.Token {
	// For INDENT/DEDENT, the literal is empty, not a null char
	if tokenType == token.INDENT || tokenType == token.DEDENT {
		return token.Token{Type: tokenType, Literal: ""}
	}
	return token.Token{Type: tokenType, Literal: string(ch)}
}