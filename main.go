package main

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
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

func Eval(node *Node) float64 {
	switch node.Type {
	case "Expression":
		return Eval(node.Children[0])
	case "Sum":
		number := Eval(node.Children[0])
		for _, sum := range node.Children[1].Children {
			if sum.Children[0].Type == "OpAdd" {
				number += Eval(sum.Children[1])
			} else {
				number -= Eval(sum.Children[1])
			}
		}
		return number
	case "Multiplication":
		number := Eval(node.Children[0])
		for _, mult := range node.Children[1].Children {
			if mult.Children[0].Type == "OpMult" {
				number *= Eval(mult.Children[1])
			} else {
				number /= Eval(mult.Children[1])
			}
		}
		return number
	case "Unit":
		return Eval(node.Children[1])
	case "Number":
		number, _ := strconv.ParseFloat(node.Value, 64)
		return number
	default:
		return 0
	}
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
		Skipping(WS, Number))(input)
}

var Number = Regex("Number", regexp.MustCompile("-?[0-9]+"))
var Name = Regex("Number", regexp.MustCompile(`\w*`))
var WS = Regex(Whitespace, regexp.MustCompile(`\s*`))

func main() {
	node, rest, ok := Expression(strings.Repeat("-1*(3*1+2)*11-(12342345+234/2243)/10000-(21)*-1282734*550 /42354345 /2342312394823744", 10000))
	fmt.Println(ok)
	if ok { 
		fmt.Println(rest)
		fmt.Println(node)
		fmt.Println(Eval(node))
	}
}
