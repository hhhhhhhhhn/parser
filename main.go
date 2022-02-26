package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"regexp"
	"strconv"
)

type NodeType string

type Node struct {
	Type     NodeType
	Children []*Node
	Value    string
}

func (node *Node) String() string {
	if node.Type == "Whitespace" {
		return ""
	}
	if node.Value != "" {
		return node.Value
	}
	output := string(node.Type) + "["
	for i, child := range node.Children {
		if i > 0 && child.Type != Char { output += " " }
		output += child.String()
	}
	return output + "]"
}

const (
	Char         NodeType = "Char"
	Whitespace   NodeType = "Whitespace"
)

type Parser func(input string) (node *Node, rest string, ok bool)

func Digit(input string) (node *Node, rest string, ok bool) {
	if len(input) > 0 && input[0] >= '0' && input[0] <= '9' {
		return &Node{Type: Char, Value: input[:1]}, input[1:], true
	}
	return nil, "", false
}

func Character(chr byte) Parser {
	return func(input string) (node *Node, rest string, ok bool) {
		if len(input) > 0 && input[0] == chr {
			return &Node{Type: Char, Value: input[:1]}, input[1:], true
		}
		return nil, "", false
	}
}

func Regex(outType NodeType, regex *regexp.Regexp) Parser {
	return func(input string) (node *Node, rest string, ok bool) {
		indexes := regex.FindStringIndex(input)
		if indexes == nil || indexes[0] != 0 {
			return nil, "", false
		}
		return &Node{Value: input[indexes[0]:indexes[1]], Type: outType}, input[indexes[1]:], true
	}
}

func Some(outType NodeType, parser Parser) Parser {
	return func(input string) (node *Node, rest string, ok bool) {
		node = &Node{Type: outType}
		rest = input
		for {
			parserNode, parserRest, parserOk := parser(rest)

			if !parserOk {
				return node, rest, true
			}

			node.Children = append(node.Children, parserNode)
			rest = parserRest
		}
	}
}

func AtLeast(outType NodeType, minimum int, parser Parser) Parser {
	return func(input string) (node *Node, rest string, ok bool) {
		num := 0
		node = &Node{Type: outType}
		rest = input
		for {
			parserNode, parserRest, parserOk := parser(rest)

			if !parserOk {
				if num >= minimum {
					return node, rest, true
				}
				return nil, "", false
			}

			num++
			node.Children = append(node.Children, parserNode)
			rest = parserRest
		}
	}
}

func Or(parsers... Parser) Parser {
	return func(input string) (node *Node, rest string, ok bool) {
		for _, parser := range parsers {
			parserNode, parserRest, parserOk := parser(input)
			if parserOk {
				return parserNode, parserRest, true
			}
		}
		return nil, "", false
	}
}

func Then(outType NodeType, parsers... Parser) Parser {
	return func(input string) (node *Node, rest string, ok bool) {
		rest = input
		node = &Node{Type: outType}
		for _, parser := range parsers {
			parserNode, parserRest, parserOk := parser(rest)
			if !parserOk {
				return nil, "", false
			}
			node.Children = append(node.Children, parserNode)
			rest = parserRest
		}
		return node, rest, true
	}
}

func ThenSkipping(outType NodeType, skip Parser, parsers... Parser) Parser {
	return func(input string) (node *Node, rest string, ok bool) {
		rest = input
		node = &Node{Type: outType}
		for _, parser := range parsers {
			_, skipRest, skipOk := skip(rest)
			if skipOk {
				rest = skipRest
			}

			parserNode, parserRest, parserOk := parser(rest)
			if !parserOk {
				return nil, "", false
			}
			node.Children = append(node.Children, parserNode)
			rest = parserRest
		}
		return node, rest, true
	}
}

func Skipping(skip Parser, parser Parser) Parser {
	return func(input string) (node *Node, rest string, ok bool) {
		_, skipRest, skipOk := skip(input)
		if skipOk {
			input = skipRest
		}
		return parser(input)
	}
}

func As(outType NodeType, parser Parser) Parser {
	return func(input string) (node *Node, rest string, ok bool) {
		node = &Node{Type: outType}
		parserNode, parserRest, parserOk := parser(input)
		node.Children = []*Node{parserNode}
		return node, parserRest, parserOk
	}
}

/////////////////////////// TEST SECTION //////////////////////////////////////

func Eval(node *Node, memory Memory) float64 {
	switch node.Type {
	case "Expression":
		return Eval(node.Children[0], memory)
	case "Sum":
		number := Eval(node.Children[0], memory)
		for _, sum := range node.Children[1].Children {
			if sum.Children[0].Type == "OpAdd" {
				number += Eval(sum.Children[1], memory)
			} else {
				number -= Eval(sum.Children[1], memory)
			}
		}
		return number
	case "Multiplication":
		number := Eval(node.Children[0], memory)
		for _, mult := range node.Children[1].Children {
			if mult.Children[0].Type == "OpMult" {
				number *= Eval(mult.Children[1], memory)
			} else {
				number /= Eval(mult.Children[1], memory)
			}
		}
		return number
	case "Unit":
		return Eval(node.Children[1], memory)
	case "Number":
		number, _ := strconv.ParseFloat(node.Value, 64)
		return number
	case "Variable":
		return memory.Variables[node.Value]
	case "FunctionCall":
		function := memory.Functions[node.Children[0].Value]
		arguments := []float64{}
		for _, argument := range node.Children[2].Children {
			arguments = append(arguments, Eval(argument.Children[0], memory))
		}
		variableCopy := make(map[string]float64)
		for name, value := range memory.Variables {
			variableCopy[name] = value
		}
		for i, argument := range arguments {
			if i < len(function.Parameters) {
				variableCopy[function.Parameters[i]] = argument
			}
		}
		scopeMemory := Memory{variableCopy, memory.Functions}
		return Eval(function.Expression, scopeMemory)
	default:
		return 0
	}
}

type Memory struct {
	Variables map[string]float64
	Functions map[string]MemoryFunction
}

type MemoryFunction struct {
	Parameters []string
	Expression *Node
}

func Exec(program *Node) {
	memory := Memory{make(map[string]float64), make(map[string]MemoryFunction)}
	for _, node := range program.Children {
		line := node.Children[0]
		switch line.Type {
		case "VariableDeclaration":
			memory.Variables[line.Children[0].Value] = Eval(line.Children[2], memory)
			fmt.Println(line.Children[0].Value, "=", memory.Variables[line.Children[0].Value])
			break
		case "Expression":
			fmt.Println(Eval(line, memory))
			break
		case "FunctionDeclaration":
			parameters := []string{}
			for _, parameter := range line.Children[2].Children {
				parameters = append(parameters, parameter.Children[0].Value)
			}
			memory.Functions[line.Children[0].Value] = MemoryFunction {
				Parameters: parameters,
				Expression: line.Children[5],
			}
			break
		}
	}
}

func Program(input string) (node *Node, rest string, ok bool) {
	return Some("Lines", 
		ThenSkipping("Line", WS,
			Or(
				Declaration,
				Expression),
			LineDelim))(input)
}

func Declaration(input string) (node *Node, rest string, ok bool) {
	return Or(
		VariableDeclaration,
		FunctionDeclaration,
	)(input)
}

func VariableDeclaration(input string) (node *Node, rest string, ok bool) {
	return ThenSkipping("VariableDeclaration", WS,
		Variable,
		Character('='),
		Expression)(input)
}

func FunctionDeclaration(input string) (node *Node, rest string, ok bool) {
	return ThenSkipping("FunctionDeclaration", WS,
		Variable,
		Character('('),
		Some("Parameters", ThenSkipping("Parameter", WS,
			Variable,
			ArguementDelimeter,
			)),
		Character(')'),
		Character('='),
		Expression)(input)
}

func Expression(input string) (node *Node, rest string, ok bool) {
	return As("Expression", Sum)(input)
}

func Sum(input string) (node *Node, rest string, ok bool) {
	return Or(
		ThenSkipping("Sum", WS,
			Multiplication, 
			AtLeast("Terms", 1, ThenSkipping("Term", WS,
				Or(As("OpAdd", Character('+')), As("OpMinus", Character('-'))),
				Multiplication))),
		Skipping(WS, Multiplication))(input)
}

func Multiplication(input string) (node *Node, rest string, ok bool) {
	return Or(
		ThenSkipping("Multiplication", WS,
			Unit,
			AtLeast("Terms", 1, ThenSkipping("Term", WS,
				Or(As("OpMult", Character('*')), As("OpDiv", Character('/'))),
				Unit))),
		Skipping(WS, Unit))(input)
}

func Unit(input string) (node *Node, rest string, ok bool) {
	return Or(
		ThenSkipping("Unit", WS,
			Character('('),
			Expression,
			Character(')')),
		Skipping(WS, FunctionCall),
		Skipping(WS, Variable),
		Skipping(WS, Number))(input)
}

func FunctionCall(input string) (node *Node, rest string, ok bool) {
	return ThenSkipping("FunctionCall", WS,
		Variable,
		Character('('),
		Some("Arguments", ThenSkipping("Argument", WS,
			Expression,
			ArguementDelimeter,
			)),
		Character(')'))(input)
}

var Number = Regex("Number", regexp.MustCompile("-?[0-9]+"))
var Variable = Regex("Variable", regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9]*`))
var ArguementDelimeter = Regex("ArgumentDelimeter", regexp.MustCompile(`,?`))
var WS = Regex(Whitespace, regexp.MustCompile(` *`))
var LineDelim = Regex(Whitespace, regexp.MustCompile(`[\n;]*`))

func main() {
	input, _ := ioutil.ReadAll(os.Stdin)
	node, rest, ok := Program(string(input))
	if ok { 
		fmt.Println("Unprocessed:", "\"" + rest + "\"")
		fmt.Println("Ast:", node)
		fmt.Println("---------------- OUTPUT -------------")
		Exec(node)
	} else {
		fmt.Println("Parser Failed")
	}
}
