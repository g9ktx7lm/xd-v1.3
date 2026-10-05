package main

import (
	"bufio"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
)

// ============================================================
//  TOKEN
// ============================================================
type TokenType int

const (
	TNumber TokenType = iota
	TOperator
	TLParen
	TRParen
	TComma
	TIdent // function name or constant
	TEOF
)

type Token struct {
	Type  TokenType
	Value string
	Num   float64
	Pos   int
}

// ============================================================
//  LEXER
// ============================================================
type Lexer struct {
	input string
	pos   int
}

func NewLexer(input string) *Lexer {
	return &Lexer{input: input}
}

func (l *Lexer) peek() byte {
	if l.pos >= len(l.input) {
		return 0
	}
	return l.input[l.pos]
}

func (l *Lexer) advance() byte {
	c := l.input[l.pos]
	l.pos++
	return c
}

func (l *Lexer) skipWhitespace() {
	for l.pos < len(l.input) && (l.input[l.pos] == ' ' || l.input[l.pos] == '\t') {
		l.pos++
	}
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }
func isLetter(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_'
}

func (l *Lexer) Next() (Token, error) {
	l.skipWhitespace()
	if l.pos >= len(l.input) {
		return Token{Type: TEOF, Pos: l.pos}, nil
	}

	start := l.pos
	c := l.peek()

	// Number (support decimals and scientific notation 1e5, 2.5e-3)
	if isDigit(c) || (c == '.' && l.pos+1 < len(l.input) && isDigit(l.input[l.pos+1])) {
		for l.pos < len(l.input) && (isDigit(l.peek()) || l.peek() == '.') {
			l.advance()
		}
		// scientific notation
		if l.pos < len(l.input) && (l.peek() == 'e' || l.peek() == 'E') {
			save := l.pos
			l.advance()
			if l.pos < len(l.input) && (l.peek() == '+' || l.peek() == '-') {
				l.advance()
			}
			if l.pos < len(l.input) && isDigit(l.peek()) {
				for l.pos < len(l.input) && isDigit(l.peek()) {
					l.advance()
				}
			} else {
				l.pos = save // rollback, 'e' was not part of number
			}
		}
		s := l.input[start:l.pos]
		n, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return Token{}, fmt.Errorf("angka tidak valid %q di posisi %d", s, start)
		}
		return Token{Type: TNumber, Value: s, Num: n, Pos: start}, nil
	}

	// Identifier (function / constant)
	if isLetter(c) {
		for l.pos < len(l.input) && (isLetter(l.peek()) || isDigit(l.peek())) {
			l.advance()
		}
		s := l.input[start:l.pos]
		return Token{Type: TIdent, Value: s, Pos: start}, nil
	}

	// Operators and parens
	switch c {
	case '+', '-', '*', '/', '%', '^':
		l.advance()
		return Token{Type: TOperator, Value: string(c), Pos: start}, nil
	case '(':
		l.advance()
		return Token{Type: TLParen, Value: "(", Pos: start}, nil
	case ')':
		l.advance()
		return Token{Type: TRParen, Value: ")", Pos: start}, nil
	case ',':
		l.advance()
		return Token{Type: TComma, Value: ",", Pos: start}, nil
	}

	return Token{}, fmt.Errorf("karakter tidak dikenal %q di posisi %d", string(c), start)
}

// ============================================================
//  PARSER (Recursive Descent)
//  Grammar (precedence rendah → tinggi):
//    expression = term (('+' | '-') term)*
//    term       = power (('*' | '/' | '%') power)*
//    power      = unary ('^' power)?          // right-assoc
//    unary      = ('+' | '-') unary | primary
//    primary    = number
//               | ident ('(' args ')')?       // function or constant
//               | '(' expression ')'
// ============================================================
type Parser struct {
	lexer   *Lexer
	current Token
	hasCur  bool
	angle   bool // degrees if true (default true)
}

func NewParser(input string, useDegrees bool) (*Parser, error) {
	p := &Parser{lexer: NewLexer(input), angle: useDegrees}
	if err := p.next(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Parser) next() error {
	tok, err := p.lexer.Next()
	if err != nil {
		return err
	}
	p.current = tok
	p.hasCur = true
	return nil
}

func (p *Parser) expect(t TokenType, val string) error {
	if p.current.Type != t {
		return fmt.Errorf("ekspektasi %q, dapat %q di posisi %d", val, p.current.Value, p.current.Pos)
	}
	return p.next()
}

func (p *Parser) Parse() (float64, error) {
	v, err := p.expression()
	if err != nil {
		return 0, err
	}
	if p.current.Type != TEOF {
		return 0, fmt.Errorf("token tak terduga %q di posisi %d", p.current.Value, p.current.Pos)
	}
	return v, nil
}

func (p *Parser) expression() (float64, error) {
	left, err := p.term()
	if err != nil {
		return 0, err
	}
	for p.current.Type == TOperator && (p.current.Value == "+" || p.current.Value == "-") {
		op := p.current.Value
		if err := p.next(); err != nil {
			return 0, err
		}
		right, err := p.term()
		if err != nil {
			return 0, err
		}
		if op == "+" {
			left += right
		} else {
			left -= right
		}
	}
	return left, nil
}

func (p *Parser) term() (float64, error) {
	left, err := p.power()
	if err != nil {
		return 0, err
	}
	for p.current.Type == TOperator &&
		(p.current.Value == "*" || p.current.Value == "/" || p.current.Value == "%") {
		op := p.current.Value
		if err := p.next(); err != nil {
			return 0, err
		}
		right, err := p.power()
		if err != nil {
			return 0, err
		}
		switch op {
		case "*":
			left *= right
		case "/":
			if right == 0 {
				return 0, fmt.Errorf("pembagian dengan nol")
			}
			left /= right
		case "%":
			if right == 0 {
				return 0, fmt.Errorf("modulo dengan nol")
			}
			left = math.Mod(left, right)
		}
	}
	return left, nil
}

func (p *Parser) power() (float64, error) {
	base, err := p.unary()
	if err != nil {
		return 0, err
	}
	if p.current.Type == TOperator && p.current.Value == "^" {
		if err := p.next(); err != nil {
			return 0, err
		}
		// right-associative
		exp, err := p.power()
		if err != nil {
			return 0, err
		}
		return math.Pow(base, exp), nil
	}
	return base, nil
}

func (p *Parser) unary() (float64, error) {
	if p.current.Type == TOperator && (p.current.Value == "+" || p.current.Value == "-") {
		op := p.current.Value
		if err := p.next(); err != nil {
			return 0, err
		}
		v, err := p.unary()
		if err != nil {
			return 0, err
		}
		if op == "-" {
			return -v, nil
		}
		return v, nil
	}
	return p.primary()
}

func (p *Parser) primary() (float64, error) {
	tok := p.current

	// Number
	if tok.Type == TNumber {
		if err := p.next(); err != nil {
			return 0, err
		}
		return tok.Num, nil
	}

	// Identifier: constant or function
	if tok.Type == TIdent {
		name := strings.ToLower(tok.Value)
		if err := p.next(); err != nil {
			return 0, err
		}

		// Constant (no paren)
		if p.current.Type != TLParen {
			switch name {
			case "pi", "π":
				return math.Pi, nil
			case "e":
				return math.E, nil
			case "tau", "τ":
				return 2 * math.Pi, nil
			case "phi", "φ":
				return math.Phi, nil
			}
			return 0, fmt.Errorf("identifier tidak dikenal: %q", name)
		}

		// Function call: consume '('
		if err := p.next(); err != nil {
			return 0, err
		}

		var args []float64
		if p.current.Type != TRParen {
			for {
				arg, err := p.expression()
				if err != nil {
					return 0, err
				}
				args = append(args, arg)
				if p.current.Type == TComma {
					if err := p.next(); err != nil {
						return 0, err
					}
					continue
				}
				break
			}
		}
		if err := p.expect(TRParen, ")"); err != nil {
			return 0, err
		}
		return p.callFunc(name, args)
	}

	// Parentheses
	if tok.Type == TLParen {
		if err := p.next(); err != nil {
			return 0, err
		}
		v, err := p.expression()
		if err != nil {
			return 0, err
		}
		if err := p.expect(TRParen, ")"); err != nil {
			return 0, err
		}
		return v, nil
	}

	return 0, fmt.Errorf("token tak terduga %q di posisi %d", tok.Value, tok.Pos)
}

// ============================================================
//  FUNGSI MATEMATIKA
// ============================================================
func (p *Parser) toRad(x float64) float64 {
	if p.angle {
		return x * math.Pi / 180
	}
	return x
}

func (p *Parser) fromRad(x float64) float64 {
	if p.angle {
		return x * 180 / math.Pi
	}
	return x
}

func (p *Parser) callFunc(name string, args []float64) (float64, error) {
	require := func(n int) error {
		if len(args) != n {
			return fmt.Errorf("%s() butuh %d argumen, dapat %d", name, n, len(args))
		}
		return nil
	}

	switch name {
	// Trigonometri
	case "sin":
		if err := require(1); err != nil { return 0, err }
		return math.Sin(p.toRad(args[0])), nil
	case "cos":
		if err := require(1); err != nil { return 0, err }
		return math.Cos(p.toRad(args[0])), nil
	case "tan":
		if err := require(1); err != nil { return 0, err }
		return math.Tan(p.toRad(args[0])), nil
	case "asin", "arcsin":
		if err := require(1); err != nil { return 0, err }
		return p.fromRad(math.Asin(args[0])), nil
	case "acos", "arccos":
		if err := require(1); err != nil { return 0, err }
		return p.fromRad(math.Acos(args[0])), nil
	case "atan", "arctan":
		if err := require(1); err != nil { return 0, err }
		return p.fromRad(math.Atan(args[0])), nil
	case "atan2":
		if err := require(2); err != nil { return 0, err }
		return p.fromRad(math.Atan2(args[0], args[1])), nil

	// Hiperbolik
	case "sinh":
		if err := require(1); err != nil { return 0, err }
		return math.Sinh(args[0]), nil
	case "cosh":
		if err := require(1); err != nil { return 0, err }
		return math.Cosh(args[0]), nil
	case "tanh":
		if err := require(1); err != nil { return 0, err }
		return math.Tanh(args[0]), nil

	// Akar & pangkat
	case "sqrt":
		if err := require(1); err != nil { return 0, err }
		if args[0] < 0 { return 0, fmt.Errorf("sqrt dari bilangan negatif") }
		return math.Sqrt(args[0]), nil
	case "cbrt":
		if err := require(1); err != nil { return 0, err }
		return math.Cbrt(args[0]), nil
	case "pow":
		if err := require(2); err != nil { return 0, err }
		return math.Pow(args[0], args[1]), nil
	case "hypot":
		if err := require(2); err != nil { return 0, err }
		return math.Hypot(args[0], args[1]), nil

	// Logaritma & eksponen
	case "log": // log base 10
		if err := require(1); err != nil { return 0, err }
		if args[0] <= 0 { return 0, fmt.Errorf("log dari ≤ 0") }
		return math.Log10(args[0]), nil
	case "ln":
		if err := require(1); err != nil { return 0, err }
		if args[0] <= 0 { return 0, fmt.Errorf("ln dari ≤ 0") }
		return math.Log(args[0]), nil
	case "log2":
		if err := require(1); err != nil { return 0, err }
		return math.Log2(args[0]), nil
	case "exp":
		if err := require(1); err != nil { return 0, err }
		return math.Exp(args[0]), nil

	// Pembulatan & nilai absolut
	case "abs":
		if err := require(1); err != nil { return 0, err }
		return math.Abs(args[0]), nil
	case "floor":
		if err := require(1); err != nil { return 0, err }
		return math.Floor(args[0]), nil
	case "ceil":
		if err := require(1); err != nil { return 0, err }
		return math.Ceil(args[0]), nil
	case "round":
		if err := require(1); err != nil { return 0, err }
		return math.Round(args[0]), nil
	case "trunc":
		if err := require(1); err != nil { return 0, err }
		return math.Trunc(args[0]), nil

	// Min / max
	case "min":
		if len(args) == 0 { return 0, fmt.Errorf("min() butuh ≥1 argumen") }
		m := args[0]
		for _, v := range args[1:] {
			if v < m { m = v }
		}
		return m, nil
	case "max":
		if len(args) == 0 { return 0, fmt.Errorf("max() butuh ≥1 argumen") }
		m := args[0]
		for _, v := range args[1:] {
			if v > m { m = v }
		}
		return m, nil
	}

	return 0, fmt.Errorf("fungsi tidak dikenal: %s()", name)
}

// ============================================================
//  EVALUATOR ENTRY
// ============================================================
func Eval(expr string, degrees bool) (float64, error) {
	p, err := NewParser(expr, degrees)
	if err != nil {
		return 0, err
	}
	return p.Parse()
}

// ============================================================
//  FORMAT OUTPUT
// ============================================================
func formatResult(v float64) string {
	if math.IsNaN(v) {
		return "NaN"
	}
	if math.IsInf(v, 1) {
		return "+Inf"
	}
	if math.IsInf(v, -1) {
		return "-Inf"
	}
	// integer?
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		return strconv.FormatFloat(v, 'f', 0, 64)
	}
	// print with up to 12 significant digits, trim trailing zeros
	s := strconv.FormatFloat(v, 'g', 12, 64)
	return s
}

// ============================================================
//  REPL (interactive)
// ============================================================
func repl() {
	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Println("║       KALKULATOR PANJANG (Go)  v1.0          ║")
	fmt.Println("╚══════════════════════════════════════════════╝")
	fmt.Println()
	fmt.Println(" Mendukung  : + - * / % ^ ( ) , serta fungsi matematika")
	fmt.Println(" Fungsi     : sin cos tan asin acos atan atan2")
	fmt.Println("              sinh cosh tanh sqrt cbrt pow hypot")
	fmt.Println("              log ln log2 exp abs floor ceil round trunc")
	fmt.Println("              min max")
	fmt.Println(" Konstanta  : pi, e, tau, phi")
	fmt.Println()
	fmt.Println(" Perintah   : :deg (mode derajat)  :rad (mode radian)")
	fmt.Println("              :help                 :quit / exit")
	fmt.Println()
	fmt.Println(" Contoh     : (2 + 3) * 4 ^ 2 / sqrt(16) - 5")
	fmt.Println()

	degrees := true
	reader := bufio.NewReader(os.Stdin)

	for {
		mode := "DEG"
		if !degrees {
			mode = "RAD"
		}
		fmt.Printf("[%s] > ", mode)

		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println()
			return
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		switch line {
		case ":quit", ":q", "exit", "quit":
			fmt.Println("Sampai jumpa 👋")
			return
		case ":help", "help":
			printHelp()
			continue
		case ":deg":
			degrees = true
			fmt.Println("Mode: DERAJAT")
			continue
		case ":rad":
			degrees = false
			fmt.Println("Mode: RADIAN")
			continue
		}

		result, err := Eval(line, degrees)
		if err != nil {
			fmt.Printf("  ✗ Error: %v\n", err)
			continue
		}
		fmt.Printf("  = %s\n", formatResult(result))
	}
}

func printHelp() {
	fmt.Println()
	fmt.Println("OPERATOR:")
	fmt.Println("  +  -  *  /  %  ^       (^ = pangkat, right-associative)")
	fmt.Println("  (  )                   (kurung buka/tutup)")
	fmt.Println("  ,                      (pemisah argumen fungsi)")
	fmt.Println()
	fmt.Println("FUNGSI:")
	fmt.Println("  Trigonometri : sin(x) cos(x) tan(x)")
	fmt.Println("                 asin(x) acos(x) atan(x) atan2(y,x)")
	fmt.Println("  Hiperbolik   : sinh(x) cosh(x) tanh(x)")
	fmt.Println("  Akar/pangkat : sqrt(x) cbrt(x) pow(x,y) hypot(x,y)")
	fmt.Println("  Logaritma    : log(x)  ln(x)  log2(x)  exp(x)")
	fmt.Println("  Pembulatan   : abs(x) floor(x) ceil(x) round(x) trunc(x)")
	fmt.Println("  Min/max      : min(a,b,...)  max(a,b,...)")
	fmt.Println()
	fmt.Println("KONSTANTA:")
	fmt.Println("  pi  e  tau  phi")
	fmt.Println()
	fmt.Println("PERINTAH:")
	fmt.Println("  :deg   mode derajat (default)")
	fmt.Println("  :rad   mode radian")
	fmt.Println("  :help  tampilkan bantuan ini")
	fmt.Println("  :quit  keluar (atau exit / quit / Ctrl+D)")
	fmt.Println()
}

// ============================================================
//  MAIN
// ============================================================
func main() {
	// Mode argumen: menu "2+3*4"
	if len(os.Args) > 1 {
		// optional flag: --rad untuk radian
		degrees := true
		args := os.Args[1:]
		if args[0] == "--rad" || args[0] == "-r" {
			degrees = false
			args = args[1:]
		} else if args[0] == "--deg" || args[0] == "-d" {
			degrees = true
			args = args[1:]
		}
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Usage: calc [--rad|--deg] <expression>")
			os.Exit(1)
		}
		expr := strings.Join(args, " ")
		result, err := Eval(expr, degrees)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(formatResult(result))
		return
	}

	// Mode interaktif
	repl()
}
