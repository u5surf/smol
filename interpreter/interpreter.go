package interpreter

import (
	"fmt"
	"math"
	"os"
	"strconv"

	"github.com/fabulousduck/smol/ast"
	"github.com/fabulousduck/smol/errors"
	"github.com/fabulousduck/smol/lexer"
)

type tuple struct {
	key   string
	value string
}

type stack []*tuple

//Stacks is the global scope that hold sub scopes for varianbles
type Stacks []stack

//Heap is not really a heap since it does not hold dynamically sized types, but a good excuse to put my function decls into
type Heap []*ast.Function

//Interpreter contains all data needed to Interpret an AST
type Interpreter struct {
	Stacks Stacks
	Heap   Heap
}

//NewInterpreter provides a new interpreter with empty base stack and heap
func NewInterpreter() *Interpreter {
	i := new(Interpreter)
	i.Stacks = Stacks{}
	i.Heap = Heap{}
	baseStack := stack{}
	i.Stacks = append(i.Stacks, baseStack)
	return i
}

//Interpret will tree walk execute an AST from left to right (topdown)
func (i Interpreter) Interpret(AST []ast.Node) {
	for j := 0; j < len(AST); j++ {
		node := AST[j]
		nodeType := node.GetNodeName()
		switch nodeType {
		case "variable":
			v := node.(*ast.Variable)
			//we can do this since only ints exist in our language
			i.stackAlloc(len(i.Stacks)-1, v)
		case "statement":
			s := node.(*ast.Statement)
			i.execStatement(s)
		case "anb":
			anb := node.(*ast.Anb)
			i.execANB(anb)
		case "function":
			function := node.(*ast.Function)
			i.execFunctionDecl(function)
		case "functionCall":
			fc := node.(*ast.FunctionCall)
			i.execFunctionCall(fc)
		case "setStatement":
			ss := node.(*ast.SetStatement)
			i.setVariableValue(ss)
		case "mathStatement":
			ms := node.(*ast.MathStatement)
			i.execMathStatement(ms)
		case "comparison":
			cm := node.(*ast.Comparison)
			i.execComparison(cm)
		case "switchStatement":
			ss := node.(*ast.SwitchStatement)
			i.execSwitchStatement(ss)
		}
	}
}

func (i *Interpreter) execSwitchStatement(ss *ast.SwitchStatement) {
	matchValue := ""
	var defaultCase []ast.Node
	matchExecuted := false
	if ss.MatchValue.GetNodeName() == "statVar" {
		scopeLevel, index := i.Stacks.find(ss.MatchValue.(*ast.StatVar).Value)
		matchValue = i.Stacks[scopeLevel][index].value
	} else {
		matchValue = ss.MatchValue.(*ast.NumLit).Value
	}

	for j := 0; j < len(ss.Cases); j++ {
		if ss.Cases[j].GetNodeName() != "switchCase" && ss.Cases[j].GetNodeName() != "end_of_switch" {
			errors.UnknownSwitchNode()
			os.Exit(65)
		}
		if ss.Cases[j].GetNodeName() == "end_of_switch" {
			defaultCase = ss.Cases[j].(*ast.Eos).Body
			continue
		}

		caseMatchValue := ""
		if ss.Cases[j].(*ast.SwitchCase).MatchValue.GetNodeName() == "statVar" {
			scopeLevel, index := i.Stacks.find(ss.Cases[j].(*ast.SwitchCase).MatchValue.(*ast.StatVar).Value)
			caseMatchValue = i.Stacks[scopeLevel][index].value
		} else {
			caseMatchValue = ss.Cases[j].(*ast.SwitchCase).MatchValue.(*ast.NumLit).Value
		}

		if matchValue == caseMatchValue {

			i.Interpret(ss.Cases[j].(*ast.SwitchCase).Body)
			matchExecuted = true
			return
		}
	}

	if !matchExecuted {
		if defaultCase != nil {
			i.Interpret(defaultCase)
		}
	}

	return
}

func (i *Interpreter) execComparison(cm *ast.Comparison) {

	clhs := 0
	crhs := 0
	beforeScopeLevel := len(i.Stacks)
	scopedStack := stack{}

	if cm.LHS.GetNodeName() == "statVar" {
		scopeLevel, index := i.Stacks.find(cm.LHS.(*ast.StatVar).Value)
		clhs, _ = strconv.Atoi(i.Stacks[scopeLevel][index].value)
	} else {
		clhs, _ = strconv.Atoi(cm.LHS.(*ast.NumLit).Value)
	}

	if cm.RHS.GetNodeName() == "statVar" {
		scopeLevel, index := i.Stacks.find(cm.RHS.(*ast.StatVar).Value)
		crhs, _ = strconv.Atoi(i.Stacks[scopeLevel][index].value)
	} else {
		crhs, _ = strconv.Atoi(cm.RHS.(*ast.NumLit).Value)
	}

	// do static analysis on same variable comparisons
	switch cm.Operator {
	case "LT":
		if clhs < crhs {
			i.Stacks = append(i.Stacks, scopedStack)
			i.Interpret(cm.Body)
		}
	case "GT":
		if clhs > crhs {
			i.Stacks = append(i.Stacks, scopedStack)
			i.Interpret(cm.Body)
		}
	case "EQ":
		if clhs == crhs {
			i.Stacks = append(i.Stacks, scopedStack)
			i.Interpret(cm.Body)
		}
	case "NEQ":
		if clhs != crhs {
			i.Stacks = append(i.Stacks, scopedStack)
			i.Interpret(cm.Body)
		}
	}
	i.Stacks = i.Stacks[:beforeScopeLevel]
	return
}

func (i *Interpreter) execMathStatement(ms *ast.MathStatement) {
	operator := ms.LHS
	if ms.MHS.GetNodeName() != "statVar" {
		errors.MathInvalidReceiverError()
	}

	receiverVariableName := ms.MHS.(*ast.StatVar).Value
	receiverVariableScopeLevel, receiverVariableIndex := i.Stacks.find(receiverVariableName)
	receiverVariableValue := i.Stacks[receiverVariableScopeLevel][receiverVariableIndex].value
	result := ""
	if ms.RHS.GetNodeName() == "statVar" {
		scopeLevel, index := i.Stacks.find(ms.RHS.(*ast.StatVar).Value)
		RHS := i.Stacks[scopeLevel][index].value
		result = evalMathExpression(operator, receiverVariableValue, RHS)
	} else {
		RHS := ms.RHS.(*ast.NumLit).Value
		result = evalMathExpression(operator, receiverVariableValue, RHS)
	}

	i.Stacks.set(receiverVariableScopeLevel, receiverVariableIndex, result)

}

func evalMathExpression(expressionType string, LHS string, RHS string) string {
	clhs, _ := strconv.Atoi(LHS)
	crhs, _ := strconv.Atoi(RHS)
	switch expressionType {
	case "ADD":
		return strconv.Itoa(clhs + crhs)
	case "SUB":
		return strconv.Itoa(clhs - crhs)
	case "MUL":
		return strconv.Itoa(clhs * crhs)
	case "DIV":
		return strconv.Itoa(clhs / crhs)
	case "POW":
		return strconv.Itoa(int(math.Pow(float64(clhs), float64(crhs))))
	}
	//not sure what to return here
	//TODO: figure above out and apply accordingly
	return RHS
}

func (i *Interpreter) setVariableValue(ss *ast.SetStatement) {
	if ss.MHS.GetNodeName() != "statVar" {
		errors.LitAssignError()
		os.Exit(65)
	}

	scopeLevel, index := i.Stacks.find(ss.MHS.(*ast.StatVar).Value)
	if ss.RHS.GetNodeName() == "statVar" {
		rhsScopeLevel, rhsIndex := i.Stacks.find(ss.RHS.(*ast.StatVar).Value)
		i.Stacks[scopeLevel][index].value = i.Stacks[rhsScopeLevel][rhsIndex].value
		return
	}
	i.Stacks[scopeLevel][index].value = ss.RHS.(*ast.NumLit).Value

}

func (i *Interpreter) execFunctionCall(fc *ast.FunctionCall) {
	functionDecl := i.Heap[i.Heap.find(fc.Name)]
	if len(fc.Args) != len(functionDecl.Params) {

		errors.IncorrectFunctionParamCountError(functionDecl.Name, len(fc.Args), len(functionDecl.Params))
		os.Exit(65)
		return
	}
	beforeScopeLevel := len(i.Stacks)
	scopedStack := stack{}
	for j := 0; j < len(functionDecl.Params); j++ {

		//if the value given is a variable, resolve it
		if lexer.DetermineStringType(fc.Args[j]) == "CHAR" {
			scopeLevel, index := i.Stacks.find(fc.Args[j])
			value := i.Stacks[scopeLevel][index].value
			scopedStack = append(scopedStack, &tuple{key: functionDecl.Params[j], value: value})
			continue
		}

		scopedStack = append(scopedStack, &tuple{key: functionDecl.Params[j], value: fc.Args[j]})
	}
	i.Stacks = append(i.Stacks, scopedStack)
	i.Interpret(functionDecl.Body)
	i.Stacks = i.Stacks[:beforeScopeLevel]
}

func (i *Interpreter) execFunctionDecl(f *ast.Function) {
	i.Heap = append(i.Heap, f)
}

func (i *Interpreter) stackAlloc(scopeLevel int, v *ast.Variable) {
	stackTuple := new(tuple)
	stackTuple.key = v.Name
	if v.Value.GetNodeName() == "statVar" {
		scopeLevel, index := i.Stacks.find(v.Value.(*ast.StatVar).Value)
		stackTuple.value = i.Stacks[scopeLevel][index].value
	} else {
		stackTuple.value = v.Value.(*ast.NumLit).Value
	}
	i.Stacks[scopeLevel] = append(i.Stacks[scopeLevel], stackTuple)
}

func (i *Interpreter) execANB(anb *ast.Anb) {
	var LHS *string
	var RHS *string

	if anb.LHS.GetNodeName() == "statVar" {
		scopeLevel, index := i.Stacks.find(anb.LHS.(*ast.StatVar).Value)
		LHS = &i.Stacks[scopeLevel][index].value
	} else {
		LHS = &anb.LHS.(*ast.NumLit).Value
	}

	if anb.RHS.GetNodeName() == "statVar" {
		scopeLevel, index := i.Stacks.find(anb.RHS.(*ast.StatVar).Value)
		RHS = &i.Stacks[scopeLevel][index].value
	} else {
		RHS = &anb.RHS.(*ast.NumLit).Value
	}
	scopedStack := stack{}
	i.Stacks = append(i.Stacks, scopedStack)
	scopeLevel := len(i.Stacks)
	v, _ := strconv.Atoi(*LHS)
	n, _ := strconv.Atoi(*RHS)
	for v != n {

		i.Interpret(anb.Body)
		v, _ = strconv.Atoi(*LHS)
		n, _ = strconv.Atoi(*RHS)
	}
	//GC the Stacks that were used in the scoped block. ANB in this case
	i.Stacks = i.Stacks[scopeLevel:]
}

func (i *Interpreter) execStatement(s *ast.Statement) {
	switch s.LHS {
	case "BRK":
		fmt.Printf("\n")
		return
	case "PRI":
		if s.RHS.GetNodeName() == "statVar" {
			RHS := s.RHS.(*ast.StatVar)
			//scope level 0 is local block scope, and then works its way up
			scopeLevel, index := i.Stacks.find(RHS.Value)
			fmt.Printf("%s", i.Stacks[scopeLevel][index].value)
			return
		}

		fmt.Printf("%s", s.RHS.(*ast.NumLit).Value)
		return
	case "PRU":
		if s.RHS.GetNodeName() == "statVar" {
			RHS := s.RHS.(*ast.StatVar)
			scopeLevel, index := i.Stacks.find(RHS.Value)
			cast, _ := strconv.Atoi(i.Stacks[scopeLevel][index].value)
			fmt.Printf("%c", cast)
			return
		}

		cast, _ := strconv.Atoi(s.RHS.(*ast.NumLit).Value)
		fmt.Printf("%c", cast)
		return
	case "INC":
		if s.RHS.GetNodeName() != "statVar" {
			errors.LitIncrementError()
			os.Exit(65)
		}

		scopeLevel, index := i.Stacks.find(s.RHS.(*ast.StatVar).Value)
		vc, _ := strconv.Atoi(i.Stacks[scopeLevel][index].value)
		vc++
		i.Stacks.set(scopeLevel, index, strconv.Itoa(vc))
		return
	}
}

func (s Stacks) set(scopeLevel int, index int, value string) {
	s[scopeLevel][index].value = value
}

func (s Stacks) find(key string) (int, int) {
	//reverse stack search so we start at local scope and keep working our way up intill we find something

	for i := len(s) - 1; i > -1; i-- {
		stackIndex := s[i].stackContains(key)
		if stackIndex != -1 {
			//scopeLevel, scopedStackIndex
			return i, stackIndex
		}
	}

	errors.UndefinedVariableError(key)
	os.Exit(65)
	return -1, -1
}

func (s stack) stackContains(key string) int {
	for i := 0; i < len(s); i++ {
		if s[i].key == key {
			return i
		}
	}
	return -1
}

func (h Heap) find(name string) int {
	for i := 0; i < len(h); i++ {
		if h[i].Name == name {
			return i
		}
	}
	errors.UndefinedFunctionReferenceError(name)
	os.Exit(65)
	return -1
}
