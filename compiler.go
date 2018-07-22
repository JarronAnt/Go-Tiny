package main

import (
	"fmt"
	"log"
	"strings"
)

//Helper functions

//helper function to determine if we have a number or not
func isNumber(char string) bool {
	if char == "" {
		return false
	}
	num := []rune(char)[0]
	if num >= '0' && num <= '9' {
		return true
	}
	return false
}

//helper function to determine if we have a letter or not
func isLetter(char string) bool {
	if char == "" {
		return false
	}
	letter := []rune(char)[0]
	if letter >= 'a' && letter <= 'z' {
		return true
	}
	return false
}

// ----------------The Tokanizer----------------
type Token struct {
	kind  string
	value string
}

func tokenizer(input string) []Token {
	//add a new line to the program
	input += "\n"

	//curr var to track positiion in the code
	curr := 0

	//add slice of our 'Token' to append tokens to
	tokens := []Token{}

	//for loop to go through the string and tokenize it

	for curr < len([]rune(input)) {
		//get the character of the input in a char variable
		char := string([]rune(input)[curr])

		//if we get a () add it to the tokens list
		if char == "(" {
			tokens = append(tokens, Token{kind: "paren", value: "("})
			curr++
			continue
		}
		if char == ")" {
			tokens = append(tokens, Token{kind: "paren", value: ")"})
			curr++
			continue
		}
		//if there is a space do nothing
		if char == " " {
			curr++
			continue
		}
		//if its a number or letter composite it togther then send to tokens list
		if isNumber(char) {
			val := ""

			for isNumber(char) {
				val += char
				curr++
				char = string([]rune(input)[curr])
			}

			tokens = append(tokens, Token{kind: "number", value: val})
			continue
		}

		if isLetter(char) {
			val := ""

			for isLetter(char) {
				val += char
				curr++
				char = string([]rune(input)[curr])
			}

			tokens = append(tokens, Token{kind: "name", value: val})
			continue
		}
		break
	}

	return tokens
}

//Parser (turn the array of tokens into an AST)

type node struct {
	kind       string
	value      string
	name       string
	callee     *node
	expression *node
	body       []node
	params     []node
	arguments  *[]node
	context    *[]node
}

//alias the node
type ast node

//program counter
var pc int

//place to store token slices
var pt []Token

//parser function that acceot the token slice
func parser(tokens []Token) ast {
	//set the counter and tokens
	pc = 0
	pt = tokens

	//create our AST with a root of Program
	ast := ast{
		kind: "Program",
		body: []node{},
	}

	//push nodes to the ast.body slice in a loop
	for pc < len(pt) {
		ast.body = append(ast.body, walk())
	}

	return ast
}

func walk() node {

	//grab current token
	token := pt[pc]

	//split into various paths based on the token

	//if we get a number return a node with that number
	if token.kind == "number" {
		pc++
		return node{
			kind:  "NumberLiteral",
			value: token.value,
		}
	}
	//look for call expressions
	if token.kind == "paren" && token.value == "(" {
		//skip the parenthesis and get the next toke
		pc++
		token = pt[pc]

		n := node{
			kind:   "CallExpression",
			name:   token.value,
			params: []node{},
		}

		//skip the name token so we can look for the closing parenthesis
		//this is done because of nested call expressions
		pc++
		token = pt[pc]

		//for loop to find a closing parameter
		//until it finds one we append the token as a param
		for token.kind != "paren" || (token.kind == "paren" && token.value != ")") {
			//call the walk to push a node to params
			n.params = append(n.params, walk())
			token = pt[pc]
		}

		//increment curr to skip the closing parenthesis
		pc++

		//add the node
		return n
	}
	//if its an unknown token throw an error
	log.Fatal(token.kind)
	return node{}
}

//Traverser

//visitior maps a string to an assosiated function
type visitor map[string]func(n *node, p node)

//func accepts and ast and a visitior
func traverser(a ast, v visitor) {
	//call traverse node without a parent cause this is the top layer
	traverseNode(node(a), node{}, v)
}

func traverseArray(a []node, p node, v visitor) {
	for _, child := range a {
		traverseNode(child, p, v)
	}
}

//accept a node and a parrent node so it can go through our visitor
func traverseNode(n, p node, v visitor) {

	//test if the visitor has methods with a matching type
	for k, va := range v {
		if k == n.kind {
			va(&n, p)
		}
	}

	switch n.kind {
	case "Program":
		//traverse into the child nodes of Program cause its the top lvl
		traverseArray(n.body, n, v)
		break
	case "CallExpression":
		//traverse into the params of CallExpression
		traverseArray(n.params, n, v)
		break
	case "NumberLiteral":
		//number literal has no child nodes
		break
	default:
		log.Fatal(n.kind)
	}

}

//Transformer - Take the AST pass it through the traverse and generate a new AST
func transformer(a ast) ast {
	//create a new ast
	nast := ast{
		kind: "Program",
		body: []node{},
	}

	//context is a refrence from the old ast to the new one
	a.context = &nast.body

	//call the traverser with the ast and a visitor
	traverser(a, map[string]func(n *node, p node){

		//fist visitor method takes number literlas
		"NumberLiteral": func(n *node, p node) {

			//push a number literal node to the parent context
			*p.context = append(*p.context, node{
				kind:  "NumberLiteral",
				value: n.value,
			})
		},

		"CallExpression": func(n *node, p node) {

			//create new call expression node
			e := node{
				kind: "CallExpression",
				callee: &node{
					kind: "Identifier",
					name: n.name,
				},
				arguments: new([]node),
			}

			//define a new context that takes expressions arguments
			n.context = e.arguments

			//check if parent is call expression node
			if p.kind != "CallExpression" {
				//if not wrap call expression in an expression statment
				es := node{
					kind:       "ExpressionStatment",
					expression: &e,
				}

				//push the call expression to parent context
				*p.context = append(*p.context, es)
			} else {
				*p.context = append(*p.context, e)
			}
		},
	})

	return nast
}

//Code Generator - prints each node of the tree into a giant string

func codeGen(n node) string {
	//break down node types
	switch n.kind {
	case "Program":
		//recursivly run through code gen and add to the string array
		var r []string
		for _, no := range n.body {
			r = append(r, codeGen(no))
		}
		return strings.Join(r, "\n")
	case "ExpressionStatment":
		return codeGen(*n.expression) + ";"
	case "CallExpression":
		//run code gen through the arguments array
		//then append them alltogether and add the open and closing parenthesis
		var ra []string
		c := codeGen(*n.callee)

		for _, no := range *n.arguments {
			ra = append(ra, codeGen(no))
		}

		r := strings.Join(ra, ", ")
		return c + "(" + r + ")"
	case "Identifier":
		return n.name
	case "NumberLiteral":
		return n.value
	default:
		log.Fatal("ERROR")
		return "ERROR"
	}
}

//COMPILATION

func compiler(input string) string {
	//run the inital string through each step of the compiler
	tokens := tokenizer(input)
	ast := parser(tokens)
	nast := transformer(ast)
	out := codeGen(node(nast))

	return out
}

func main() {
	program := "(add 2 3 4)"
	output := compiler(program)
	fmt.Println(output)
}
