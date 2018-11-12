package smol

import (
	"io/ioutil"
	"os"

	"github.com/davecgh/go-spew/spew"

	"github.com/fabulousduck/smol/ast"
	"github.com/fabulousduck/smol/interpreter"
	"github.com/fabulousduck/smol/lexer"
)

//Smol : Defines the global attributes of the interpreter
type Smol struct {
	Tokens   []*lexer.Token
	HadError bool //TODO: use this
}

//NewSmol : Creates a new Smol instance
func NewSmol() *Smol {
	return new(Smol)
}

//RunFile : Interprets a given file
func (smol *Smol) RunFile(filename string) {
	file, err := ioutil.ReadFile(filename)
	if err != nil {
		panic(err)
	}
	smol.Run(string(file), filename)
	if smol.HadError {
		os.Exit(65)
	}
}

//Run exectues a given script
func (smol *Smol) Run(sourceCode string, filename string) {
	l := lexer.NewLexer(filename)
	l.Lex(sourceCode)
	spew.Dump(l)
	p := ast.NewParser(filename)
	//We can ignore the second return value here as it is the amount of tokens consumed.
	//We do not need this here
	p.Ast, _ = p.Parse(l.Tokens)
	i := interpreter.NewInterpreter()
	i.Interpret(p.Ast)
}
