// package expression only used for filter operations
// rootfunc needs only lexer/parser to determine func to execute, which returns []{[]util.UID, Type}, where each item is passed to expression to return true/false

package expression

import (
	"fmt"
	"strings"

	blk "github.com/DynamoGraph/block"
	"github.com/DynamoGraph/ds"
	"github.com/DynamoGraph/gql/expression/ast"
	"github.com/DynamoGraph/gql/expression/token"
	"github.com/DynamoGraph/util"
)

//
// literal Value used in functions to specify search criteria value
//

//
// rootFunc appears at top of GraphQL+- query. Returns initial candidate UID set.
//

type operator = string

// Connectives AND, OR and NOT join filters and can be built into arbitrarily complex filters, such as (NOT A OR B) AND (C AND NOT (D OR E)).
// Note that, NOT binds more tightly than AND which binds more tightly than OR.

type Item struct {
	uid util.UID
	ty  string
}

type node struct {
	ty   string
	j, k int
}

func (e *Expression) rootExecute(nv ds.NVmap, ty string) bool {

	v := node{ty: ty, j: -1, k: -1}
	walk(e, nv, v)
	return e.getResult(nv, v)
}

func (e *Expression) execute(nv ds.NVmap, ty string, j, k int) bool {

	v := node{ty: ty, j: j, k: k}
	walk(e, nv, v)
	return e.getResult(nv, v)
}

func walk(e operand, nv ds.NVmap, v node) {

	if e, ok := e.(*Expression); ok {

		walk(e.left, nv, v)
		walk(e.right, nv, v)

		switch e.opr {

		case token.AND:

			e.result = e.left.getResult(nv, v) && e.right.getResult(nv, v)

		case token.OR:

			e.result = e.left.getResult(nv, v) || e.right.getResult(nv, v)

		case token.NOT:

			e.result = !e.right.getResult(nv, v)

		case token.NOOP:

			e.result = e.left.getResult(nv, v)

		default: //  represents ( )

			e.result = e.right.getResult(nv, v)

		}
		fmt.Println(strings.Repeat("+", 80))
		fmt.Printf("Result: %v %v\n", e.opr, e.result)
	}

}

func walkPreds(e operand, pred []string) []string {

	if e, ok := e.(*Expression); ok {

		walkPreds(e.left, pred)
		walkPreds(e.right, pred)

		pred := e.left.getPredicates(pred)
		fmt.Println("in walkPreds. left...pred = ", pred)
		// if len(p) > 0 {
		// 	pred = append(pred, p...)
		// }
		pred = e.right.getPredicates(pred)
		fmt.Println("in walkPreds. right...pred = ", pred)
		// if len(p) > 0 {
		// 	pred = append(pred, p...)
		// }

		return pred

	}
	return pred
}

func (e *Expression) GetPredicates() []string {

	var pred []string

	pred = walkPreds(e, pred)

	return pred

}

func findRoot(e *Expression) *Expression {

	for e.parent != nil {
		e = e.parent
	}
	return e
}

// operand interface.
// So far type num (integer), Expression satisfy, but this can of course be extended to floats, complex numbers, functions etc.
type operand interface {
	//	getParent() *Expression
	getPredicates([]string) []string
	type_() string
	printName() string
	getResult(ds.NVmap, node) bool
}

type Expression struct { // expr1 and expr2     expr1 or expr2       exp1 or (expr2 and expr3). (expr1 or expr2) and expr3
	id     uint8           // type of Expression. So far used only to identify the NULL Expression, representing the "(" i.e the left parameter or LPARAM in a mathematical Expression
	name   string          // optionally give each Expression a name. Maybe useful for debugging purposes.
	result bool            // store result of "left operator right. Walking the graph will interrogate each operand for its result.
	left   operand         //
	opr    token.TokenType // for Boolean: AND OR NOT NULL (aka "(")
	right  operand         //
	parent *Expression
	//
	depth int8
}

func (e *Expression) getParent() *Expression {
	return e.parent
}
func (e *Expression) gype_() string {
	return "Expression"
}
func (e *Expression) printName() string {
	return e.name
}

func (f *Expression) getPredicates(p []string) []string {
	return p
}

func (e *Expression) type_() string {
	return "expression"
}

func (e *Expression) getResult(nv ds.NVmap, v node) bool {

	return e.result
}

var active bool

// func (e *Expression) expression(nv ds.NVmap) {
// }

// RootApply filters the Root query result for a single PKey only. The result for each predicate
// in the zero level of the graph (first node) are held in nv
func (e *Expression) RootApply(nv ds.ClientNV, ty string) bool {
	nvm := make(ds.NVmap)
	for _, v := range nv {
		nvm[v.Name] = v
	}
	return e.rootExecute(nvm, ty)
}

// Apply will run the filter function over all edges for a uid-pred
// NV value contains all the edges associated with the current uid-pred stored as [][]<type> for each predicate.
// see cache.UnmarshalNodeCache for more details of the data structure.
func (e *Expression) Apply(nvm ds.NVmap, ty string, predicate string) bool {
	//
	// source NV for current uid-pred
	nv := nvm[predicate]
	if x, ok := nv.Value.([][][]byte); !ok {
		panic(fmt.Errorf("Expression: nv.Value not a [][][]byte"))
	} else {
		// apply filter to all edges and set edge to deleted if it fails
		for i, u := range x {
			for k, _ := range u {
				// if k == 0 { // skip first entry
				// 	continue
				// }
				if nv.State[i][k] == blk.UIDdetached {
					// soft deleted
					continue
				}
				return e.execute(nvm, ty, i, k)
				// if !e.execute(nvm, ty, i, k) {
				// 	// false: set corresponding Xf flag to false (soft delete)
				// 	nv.State[i][k] = blk.UIDdetached // soft deleted
				// }
			}
		}
	}
	return false
	//return e.execute(nvm, ty, predicate)
}

// =============================================================================
// ( eq(predicate, value) and eq(..)
// eq(val(varName), value)
// eq(predicate, val(varName))
// eq(count(predicate), value)
// eq(predicate, [val1, val2, , valN])
// eq(predicate, [$var1, "value", , $varN])

//   eq(count(genre), 13))

type FunctionI interface {
	function()
}
type FilterFunc struct {
	//parent *Expression
	//value  bool   // for testing only
	name string // for debug purposes - not used yet
	//predicate string // used to point to particular nv value at execution time - not necessary???
	//nv *ds.NV // data from cache for all predicates
	//
	value bool // maybe useful for dummy expression where roperand is nil a bool rather than a function
	//
	gqlFunc *ast.GQLFunc
}

func (f *FilterFunc) oper() {} // ???

// func (f *FilterFunc) getParent() *Expression {
// 	return f.parent
// }

func (f *FilterFunc) type_() string {
	return "func"
}

func (f *FilterFunc) getResult(nv ds.NVmap, v node) bool {
	gf := f.gqlFunc
	return gf.F(gf.Farg, gf.Value, nv, v.ty, v.j, v.k)

}

func (f *FilterFunc) printName() string {
	return f.name
}

func (f *FilterFunc) getPredicates(pred []string) []string {
	// for dummy expressions the right operand has a nil gqlFunc
	if f.gqlFunc != nil {
		return f.gqlFunc.GetPredicates(pred)
	}
	return pred
}

func (f *FilterFunc) Print() string {
	return f.name
}

func (f *FilterFunc) String() string {
	return f.gqlFunc.String()
}

func makeExpr(l operand, op token.TokenType, r operand) (*Expression, token.TokenType) {

	fmt.Println("MakeExpression: ", op)

	e := &Expression{left: l, opr: op, right: r}

	return e, ""
}

// ExtendRight for Higher Precedence operators or open braces - parsed:   *,/, (
// c - current op node, n is the higer order op we want to extend right
func (c *Expression) extendRight(n *Expression) *Expression {

	c.right = n
	n.parent = c

	fmt.Printf("++++++++++++++++++++++++++ extendRight  FROM %s  -> [%s]  \n", c.opr, n.opr)
	return n
}

func (c *Expression) addParent(n *Expression) *Expression {
	//
	fmt.Println("addParent on ", c.opr, n.opr)
	if c.parent != nil {
		//  current node must now point to the new node being added, and similar the new node must point back to the current node.
		c.parent.right = n
		n.parent = c.parent
	}
	// set old parent to new node
	c.parent = n
	n.left = c

	fmt.Printf("\n++++++++++++++++++++++++++ addParent  %s on %s \n\n", n.opr, c.opr)
	return n
}
