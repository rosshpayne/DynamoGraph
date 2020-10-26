package ast

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/DynamoGraph/db"
	"github.com/DynamoGraph/ds"
	expr "github.com/DynamoGraph/gql/expression"
	"github.com/DynamoGraph/gql/token"
	"github.com/DynamoGraph/util"
	//"github.com/DynamoGraph/rdf/grmgr"
)

type FargI interface {
	String() string
	farg()
}

type InnerFuncI interface {
	innerFunc()
}

type FilterI interface {
	AssignFilter(*expr.Expression)
	AssignFilterStmt(string)
}

// type InnerArgI interface {
// 	innerArg()
// }

type SelectI interface {
	AssignSelectList(SelectList)
	initialise()
	hasNoData() bool
	assignData(string, ds.ClientNV, index) ds.NVmap
	getData(string) (ds.NVmap, ds.ClientNV, bool)
	getIdx(string) (index, bool)
	genNV() ds.ClientNV
}

type SelectList []*EdgeT

func (sl SelectList) String() string {
	var s strings.Builder
	for _, e := range sl {
		s.WriteString(e.String())
		s.WriteByte('\n')
	}
	return s.String()
}

// func (sl SelectList) GetPredicates() []string {
// 	var ps []string
// 	for _, e := range sl {
// 		ps = append(ps, e.GetPredicates()...)
// 	}
// 	return ps
// }

type EdgeT struct {
	Alias   name_
	VarName name_ // TODO: type should be Variable maybe
	Edge    EdgeI
	//f         aggrFunc - now in predicate (edgeT)
}

func (e EdgeT) String() string {
	var s strings.Builder
	if len(e.Alias.Name) > 0 {
		s.WriteString(e.Alias.Name)
		s.WriteString(" : ")
	}
	if len(e.VarName.Name) > 0 {
		s.WriteString(e.VarName.Name)
		s.WriteString(" as ")
	}
	s.WriteString(e.Edge.String())
	return s.String()
}

func (e *EdgeT) AssignName(input string, loc token.Pos) {
	//ValidateName(input, err, Loc)
	e.Alias = name_{Name: input, Loc: loc}
}

func (e *EdgeT) AssignVarName(input string, loc token.Pos) {
	//ValidateName(input, err, Loc)
	e.VarName = name_{Name: input, Loc: loc}
}

// func (e *EdgeT) GetPredicates() []string {
// 	return e.Edge.GetPredicates()
// }

func (e *EdgeT) JSON() string {
	var s strings.Builder
	if len(e.Alias.Name) != 0 {
		s.WriteString(e.Alias.String())
	}
	if len(e.VarName.Name) != 0 {
		//s.WriteString(GetVarValue(e.VarName))
	} else {
		s.WriteString(e.Edge.String())
	}
	return s.String()
}

type EdgeI interface {
	edge()
	String() string
	//	GetPredicates() []string
	Name() string
}

// type Arg1 interface {
// 	farg()
// }

type ScalarPred struct {
	Name_  name_
	Parent SelectI
}

func (s ScalarPred) edge() {}
func (s ScalarPred) farg() {}

func (s *ScalarPred) AssignName(input string, loc token.Pos) {
	//ValidateName(input, err, Loc)
	s.Name_ = name_{Name: input, Loc: loc}
}

func (s ScalarPred) String() string {
	return s.Name_.Name
}

func (s ScalarPred) Name() string {
	return s.Name_.Name
}

type NdNv map[util.UIDb64s]ds.ClientNV
type NdNvMap map[util.UIDb64s]ds.NVmap
type NdIdx map[util.UIDb64s]index

// func (s ScalarPred) GetPredicates() []string {
// 	return []string{s.Name_.Name}
// }
// type Data struct {
// 	ScKey ScalarKey
// 	Nd    map[util.UIDb64s]ds.ClientNV
// }
type UidPred struct {
	//
	// meta data description
	//
	Name_      name_
	Printed    bool    // false - not printed, true - has been printed
	Parent     SelectI // *RootStmt, *UidPred
	Filter     *expr.Expression
	filterStmt string
	Select     SelectList
	//
	// data
	//
	lvl    int     // depth of grap
	nodes  NdNvMap // scalar nodes including PKey associated with each nodes belonging to this edge.
	nodesc NdNv
	nodesi NdIdx // nodes's index into parent uid-pred's UL data. e.g. to get Age of this nodes - nv:=nodes.parent.nodes[uid]; age:= nv["Age"].([][]int); age[nodesi.i][nodesi.j]

	// scalar nodes for nodes containing this uid-pred is contained in the parent.

}

func (p UidPred) edge() {}
func (u *UidPred) AssignName(input string, loc token.Pos) {
	//ValidateName(input, err, Loc)
	u.Name_ = name_{Name: input, Loc: loc}
}

func (u *UidPred) AssignFilter(e *expr.Expression) {
	u.Filter = e
}

func (u *UidPred) AssignFilterStmt(e string) {
	u.filterStmt = e
}

func (u *UidPred) GetLvl() int {
	return u.lvl
}

// func (u *UidPred) MakeNVM() {
// 	u.nvm = make(map[string][]ds.ClientNV)
// }

func (p UidPred) Name() string {
	return p.Name_.Name
}

func (u *UidPred) AssignSelectList(s SelectList) {
	u.Select = s
}
func (u *UidPred) initialise() {
	u.nodes = make(NdNvMap)
	u.nodesc = make(NdNv)
	u.nodesi = make(NdIdx)
}
func (u *UidPred) hasNoData() bool {
	return u.nodes == nil
}
func (u *UidPred) getIdx(key string) (index, bool) {
	i, ok := u.nodesi[key]
	return i, ok
}

func (u *UidPred) assignData(key string, nvc ds.ClientNV, idx index) ds.NVmap {
	// make a ds.NVmap from nvc a ds.ClientNV
	nvm := make(ds.NVmap)
	for _, v := range nvc {
		nvm[v.Name] = v
	}
	// add to existing nodes on this edge
	u.nodes[key] = nvm
	//
	fmt.Printf("in assignData  for key %s %d\n", key, len(u.nodes[key]))
	u.nodesc[key] = nvc
	u.nodesi[key] = idx

	fmt.Println("End assignData: ", len(nvm), len(u.nodes), len(nvc), len(u.nodesc), len(u.nodesi))

	return nvm
}
func (u *UidPred) getData(key string) (ds.NVmap, ds.ClientNV, bool) {
	nvm, _ := u.nodes[key]
	nvc, ok := u.nodesc[key]
	fmt.Println("in getData return %v", ok)
	return nvm, nvc, ok
}

// func (u *UidPred) execNode(grl grmgr.Limiter, n util.UID) {...} // see execute.go
// uidp: uid-pred of interest
//func (u *UidPred) genNV(uidp string) (ds.NVmap, ds.ClientNV) {
func (u *UidPred) genNV() ds.ClientNV {
	var nvc ds.ClientNV

	fmt.Printf("\nin uidpred.genNV() .....\n")
	if u.Filter != nil {
		for _, x := range u.Filter.GetPredicates() {
			nv := &ds.NV{Name: x}
			nvc = append(nvc, nv)
		}
	}

	for _, v := range u.Select {
		fmt.Printf("uidpred.select : %s\n", v.Edge.Name())
		switch x := v.Edge.(type) {

		// scalar nodes can be sourced from parent uid-pred's NV nodes in UL cache structure - which may be slow unless we keep {i,j} index into it.
		// other include the below scalar preds which will be stored in the NV structure in uid-pred.nodes
		//
		// case *ScalarPred:
		// 	nv := &ds.NV{Name: x.Name()}
		// 	nvc = append(nvc, nv)

		case *UidPred:
			// if uidp != x.Name() {
			// 	continue // only interested in current uid-pred
			// }
			un := x.Name() + ":"
			fmt.Println("yy genNV un: ", un)
			nv := &ds.NV{Name: un}
			nvc = append(nvc, nv)
			for _, vv := range x.Select {
				switch x := vv.Edge.(type) {
				case *ScalarPred:
					upred := un + x.Name()
					fmt.Println("xx genNV upred: ", upred)
					nv := &ds.NV{Name: upred}
					nvc = append(nvc, nv)
				}
			}
		}

	}
	return nvc
}

// func (u *UidPred) GetPredicates() []string {
// 	var ps []string
// 	ps = append(ps, u.Name())
// 	// 	ps = append(ps, u.Filter.GetPredicates()...)
// 	ps = append(ps, u.Select.GetPredicates()...)
// 	return ps
// }

//func (p *UIDPred) innerArg() {}
func (p *UidPred) aggrArg() {}
func (p *UidPred) cntArg()  {}
func (p UidPred) String() string {
	var s strings.Builder
	s.WriteString(p.Name_.Name)
	// Filter
	if p.Filter != nil {
		s.WriteString("@filter( ")
		s.WriteString(p.filterStmt)
		s.WriteByte(')')
	}
	if p.Select != nil {
		s.WriteString("{\n")
		s.WriteString(p.Select.String())
		s.WriteByte('}')
	}
	return s.String()
}

type Variable struct {
	Name_ name_
}

func (u *Variable) AssignName(input string, loc token.Pos) {
	//ValidateName(input, err, Loc)
	u.Name_ = name_{Name: input, Loc: loc}
}
func (u *Variable) String() string {
	return "var(" + u.Name() + ")"
}

//func (r *Variable) innerArg()  {}
func (r *Variable) edge()   {}
func (r *Variable) cntArg() {}
func (r *Variable) Name() string {
	return r.Name_.Name
}

// not for root func: func (r *Variable) farg() {}

//func (r *Variable) innerFunc() {}
func (r *Variable) aggrArg() {}

// func (r *Variable) GetPredicates() []string {
// 	return nil
// }

//type ValFuncT func(v Variable) ValOut

type AggrArg interface {
	aggrArg()
}
type AggrFunc struct {
	Name name_ // count, avg takes either a variable argument or a uid-pred argument
	Arg  AggrArg
}

func (u *AggrFunc) AssignName(input string, loc token.Pos) {
	//ValidateName(input, err, Loc)
	u.Name = name_{Name: input, Loc: loc}
}

func (e *AggrFunc) edge() {}

//func (e *AggrFunc) innerFunc() {}

type CounterI interface {
	cntArg()
	String() string
	//	GetPredicates() []string
	Name() string
}

type CountFunc struct {
	Arg CounterI // uidPred, UID, Variable
}

func (e *CountFunc) farg() {}
func (e *CountFunc) edge() {}
func (e *CountFunc) String() string {
	var s strings.Builder
	s.WriteString("count(")
	s.WriteString(e.Arg.String())
	s.WriteByte(')')
	return s.String()
}
func (e *CountFunc) Name() string {
	return e.Arg.Name()
}

// func (e *CountFunc) GetPredicates() []string {
// 	return e.Arg.GetPredicates()
// }

type UID struct{}

func (e UID) edge()   {}
func (e UID) cntArg() {}
func (e UID) String() string {
	return "uid"
}
func (e UID) GetPredicates() []string {
	return nil
}

func (e UID) Name() string {
	return ""
}

type Values []interface{} // int,float,string,$var

// =========================  GQLFunc  =============================================

type FuncT func(FargI, interface{}) db.QResult

//type FuncT func(predfunc FargI, value interface{}, nv []ds.NV, ty string) []db.QResult

type GQLFunc struct {
	//	name  name_ // for String() purposes - TODO: check its not used if so remove it
	FName name_ // function name
	F     FuncT
	Farg  FargI // either predicate, count, var
	//	IFarg InnerArgI   // either uidPred, variable
	Value interface{} //  literal value: string,int,float,$var, List of string,int,float,$var
}

func (g *GQLFunc) AssignName(input string, loc token.Pos) {
	//ValidateName(input, err, Loc)
	g.FName = name_{Name: input, Loc: loc}
}

// func (g *GQLFunc) Execute() []db.QResult {
// 	//
// 	return g.F(g.Farg, g.Value)
// }

func (g *GQLFunc) Name() string {
	return g.FName.Name
}

func (f *GQLFunc) String() string {
	var s strings.Builder
	s.WriteString(f.FName.Name)
	s.WriteByte('(')
	s.WriteString(f.Farg.String())
	s.WriteByte(',')
	switch x := f.Value.(type) {
	case string:
		s.WriteByte('"')
		s.WriteString(x)
		s.WriteByte('"')
	case int:
		s.WriteString(strconv.Itoa(x))
	case float64:
		s.WriteString(strconv.FormatFloat(x, 'G', -1, 64))
		// list of literals, list of $varN...
	}
	s.WriteByte(')')
	return s.String()
}

// func (f *GQLFunc) String() string {
// 	//
// 	var s strings.Builder
// 	s.WriteString(f.Name.String())
// 	s.WriteByte('(')
// 	if f.Farg != nil {
// 		switch x := f.Farg.(type) {
// 		case *CountFunc:
// 			s.WriteString("count(")
// 			switch y := x.Arg.(type) {
// 			case *UidPred:
// 				s.WriteString(y.Name())
// 			case UID:
// 				s.WriteString("uid")
// 			}
// 			s.WriteString(")")
// 		case ScalarPred:
// 			s.WriteString(x.Name())
// 			//
// 		}
// 	}
// 	s.WriteByte(',')
// 	switch x := f.Value.(type) {
// 	case string:
// 		s.WriteByte('"')
// 		s.WriteString(x)
// 		s.WriteByte('"')
// 	case int:
// 		s.WriteString(strconv.Itoa(x))
// 	case float64:
// 		s.WriteString(strconv.FormatFloat(x, 'G', -1, 64))
// 		// list of literals, list of $varN...
// 	}
// 	s.WriteString(")")

// 	return s.String()
// }

// ============== Select/edge List ==============

// {
//   ID as var(func: allofterms(name@en, "Steven")) @filter(has(director.film)) {
//     director.film {
//       num_actors as count(starring)
//     }
//     average as avg(val(num_actors))
//   }

//   films(func: uid(ID)) {
//     director_id : uid
//     english_name : name@en
//     average_actors : val(average)
//     num_films : count(director.film)

//     films : director.film {
//       name : name@en
//       english_name : name@en
//       french_name : name@fr
//     }
//   }
// }
// {
//   me(func: eq(name@en, "Steven Spielberg")) @filter(has(director.film)) {
//     name@en
//     director.film @filter(allofterms(name@en, "jones indiana"))  {
//       name@en
//     }
//   }
// }

//func (s *ScalarPred) farg() {}

// ============== RootStmt ==============

type RootStmt struct {
	Name name_
	Var  *Variable
	Lang string
	// (func: eq(name@en, "Steven Spielberg”))
	RootFunc GQLFunc // generates []uid from GSI data io.Writer Write([]byte) (int, error)
	// @filter( has(director.film) )
	filterStmt string           // for printing filter expression
	Filter     *expr.Expression //
	Select     SelectList
	//
	//PredList []string
	// populated during execution phase = contains slice of predicate,value for current nodes and child nodess
	//result []rootResult - executor passes nv results to goroutine collector which formats the results and prints out on request
	nodes  NdNvMap // scalar nodes including PKey associated with each nodes belonging to this edge.
	nodesc NdNv
	nodesi NdIdx
}

func (r *RootStmt) AssignName(input string, loc token.Pos) {
	//ValidateName(input, err, Loc)
	r.Name = name_{Name: input, Loc: loc}
}

func (r *RootStmt) AssignSelectList(s SelectList) {
	r.Select = s
}

func (r *RootStmt) AssignFilter(e *expr.Expression) {
	r.Filter = e
}

func (r *RootStmt) AssignFilterStmt(e string) {
	r.filterStmt = e
}

func (r *RootStmt) AssignVarName(input string, loc token.Pos) {
	//ValidateName(input, err, Loc)
	r.Var.AssignName(input, loc)
}

func (r *RootStmt) hasNoData() bool {
	return r.nodes == nil
}
func (r *RootStmt) initialise() {
	r.nodes = make(NdNvMap)
	r.nodesc = make(NdNv)
	r.nodesi = make(NdIdx)
}

func (r *RootStmt) assignData(key string, nvc ds.ClientNV, idx index) ds.NVmap {
	// create a NVmap
	nvm := make(ds.NVmap)
	for _, v := range nvc {
		nvm[v.Name] = v
	}
	// add to existing nodes on this edge
	r.nodes[key] = nvm
	r.nodesc[key] = nvc
	r.nodesi[key] = idx

	fmt.Println("End assignData: ", len(nvm), len(r.nodes), len(nvc), len(r.nodesc), len(r.nodesi))
	return nvm
}

func (r *RootStmt) getData(key string) (ds.NVmap, ds.ClientNV, bool) {
	nvm, ok := r.nodes[key]
	nvc, ok := r.nodesc[key]
	return nvm, nvc, ok
}

func (r *RootStmt) getIdx(key string) (index, bool) {
	i, ok := r.nodesi[key]
	return i, ok
}

// genNV generates NV nodes based on type (parameter ty) passed in
func (r *RootStmt) genNV() ds.ClientNV {
	var nvc ds.ClientNV

	fmt.Println("In genNV()............")
	if r.Filter != nil {
		s := r.Filter.GetPredicates()
		fmt.Println("r.Filter.GetPredicates():  ", s)
		for _, x := range s {
			nv := &ds.NV{Name: x}
			nvc = append(nvc, nv)
		}
	}

	for _, v := range r.Select {

		fmt.Printf("In genNV()..select %T\n", v.Edge)
		switch x := v.Edge.(type) {

		case *ScalarPred:
			fmt.Println("In genNV()..select scalar ")
			nv := &ds.NV{Name: x.Name()}
			fmt.Println("In genNV()..select scalar ", nv.Name)
			nvc = append(nvc, nv)

		case *UidPred:
			var un string
			un = x.Name() + ":"
			fmt.Println("root genNV un: ", un)
			nv := &ds.NV{Name: un}
			nvc = append(nvc, nv)

			for _, vv := range x.Select {
				fmt.Printf("Found edge for root uid-pred %T\n", vv.Edge)
				switch x := vv.Edge.(type) {
				case *ScalarPred:
					upred := un + x.Name()
					fmt.Println("root genNV upred: ", upred)
					nv := &ds.NV{Name: upred}
					nvc = append(nvc, nv)
				}
			}
		}
	}
	nvc = dedup(nvc)
	return nvc
}

func (r *RootStmt) String() string {
	var s strings.Builder

	s.WriteByte('{')
	s.WriteByte('\n')
	s.WriteString(r.Name.String())
	s.WriteString("(func: ")
	s.WriteString(r.RootFunc.String())
	if r.Filter != nil {
		s.WriteString("@filter( ")
		s.WriteString(r.filterStmt)
		s.WriteByte(')')
	}
	s.WriteByte(')')
	s.WriteString("{\n")
	s.WriteString(r.Select.String())
	s.WriteByte('}')

	s.WriteByte('}')

	return s.String()
}

// Predicates lists all predicates involved in the root stmt i.e. in RootFunc filter, and edges
// func (r *RootStmt) RetrievePredicates() []string {
// 	var s []string
// 	if r.RootFunc.Farg != nil {
// 		switch x := r.RootFunc.Farg.(type) {
// 		case *CountFunc:
// 			switch y := x.Arg.(type) {
// 			case *UidPred:
// 				s = append(s, y.Name.Name)
// 			}
// 		case ScalarPred:
// 			s = append(s, x.Name.Name)
// 		}
// 	}
// 	// s = append(s, r.Filter.GetPredicates()...)
// 	s = append(s, r.Select.GetPredicates()...)

// 	return s
// }

// func dedup(s []string) []string {
// 	var ss []string
// 	var found bool
// 	ss = append(ss, s[0])
// 	for _, e := range s[1:] {
// 		found = false
// 		for _, d := range ss {
// 			if d == e {
// 				found = true
// 				break
// 			}
// 		}
// 		if !found {
// 			ss = append(ss, e)
// 		}
// 	}
// 	return ss
// }

func dedup(s ds.ClientNV) ds.ClientNV {
	var ss ds.ClientNV
	var found bool
	ss = append(ss, s[0])
	for _, e := range s[1:] {
		found = false
		for _, d := range ss {
			if d.Name == e.Name {
				found = true
				break
			}
		}
		if !found {
			ss = append(ss, e)
		}
	}
	return ss
}

// ============== QResult ==============

// type QResult struct {
// 	UID   util.UID
// 	SortK string
// 	Ty    string
// }

// ============== NameI  ========================

type NameAssigner interface {
	AssignName(string, token.Pos)
}

// ===============  NameValue_  =========================

// type NameValue_ string

// func (n NameValue_) String() string {
// 	return string(n)
// }

// func (a NameValue_) Equals(b NameValue_) bool {
// 	return string(a) == string(b)
// }

// func (a NameValue_) EqualString(b string) bool {
// 	return string(a) == b
// }

// ===============  name_  =========================

type name_ struct {
	Name string
	Loc  token.Pos
}

func (n name_) String() string {
	return string(n.Name)
}

func (n name_) AtPosition() string {
	if n.Loc.Col == 0 && n.Loc.Line == 0 {
		//panic(fmt.Errorf("Error in AtPosition(), Loc not set"))
		return "Loc not set"
	}
	return "" + strconv.Itoa(n.Loc.Line) + " " + strconv.Itoa(n.Loc.Col) + "] "
}

func (n name_) Exists() bool {
	if len(n.Name) > 0 {
		return true
	}
	return false
}

// // =========== Loc_ =============================

// type Loc struct {
// 	Line   int
// 	Column int
// }

// func (l Loc) String() string {
// 	return "at line: " + strconv.Itoa(l.Line) + " " + "column: " + strconv.Itoa(l.Column)
// 	//return "" + strconv.Itoa(l.Line) + " " + strconv.Itoa(l.Column) + "] "
// }

// type HasName interface {
// 	AssignName(string, Loc_,)
// }
