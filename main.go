package main

import (
	"fmt"
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
		if i > 0 && child.Type != Char { output += ", " }
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

func As(outType NodeType, parser Parser) Parser {
	return func(input string) (node *Node, rest string, ok bool) {
		node = &Node{Type: outType}
		parserNode, parserRest, parserOk := parser(input)
		node.Children = []*Node{parserNode}
		return parserNode, parserRest, parserOk
	}
}

func Eval(node *Node) int {
	switch node.Type {
	case "Expression":
		if len(node.Children) == 1 {
			return Eval(node.Children[0])
		}
		return Eval(node.Children[1]) + Eval(node.Children[5])
	case "Term":
		if len(node.Children) == 1 {
			return Eval(node.Children[0])
		}
		return Eval(node.Children[1]) * Eval(node.Children[5])
	case "Factor":
		if len(node.Children) == 1 {
			return Eval(node.Children[0])
		}
		return Eval(node.Children[3])
	case "Number":
		number, _ := strconv.Atoi(node.Value)
		return number
	default:
		return 0
	}
}

func Expression(input string) (node *Node, rest string, ok bool) {
	return Or(Then("Expression", WS, Term, WS, As("Operator", Character('+')), WS, Expression), As("Expression", Term))(input)
}

func Term(input string) (node *Node, rest string, ok bool) {
	return Or(Then("Term", WS, Factor, WS, As("Operator", Character('*')), WS, Term), As("Term", Factor))(input)
}

func Factor(input string) (node *Node, rest string, ok bool) {
	return Or(Then("Factor", WS, Character('('), WS, Expression, WS, Character(')')), As("Factor", Number))(input)
}

var Number = Regex("Number", regexp.MustCompile("[0-9]+"))
var WS = Regex(Whitespace, regexp.MustCompile(`\s*`))

func main() {
	node, rest, ok := Expression("(1+2+4+3+2+453*2+4*12*(5*123+123*545)*321)")
	fmt.Println(node)
	fmt.Println(rest)
	fmt.Println(ok)
	fmt.Println(Eval(node))
}
