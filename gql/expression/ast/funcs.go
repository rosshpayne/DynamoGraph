package ast

import (
	"fmt"
	"strings"

	"github.com/DynamoGraph/block"
	"github.com/DynamoGraph/cache"
	//"github.com/DynamoGraph/db"
	"github.com/DynamoGraph/ds"
)

type inEQ uint8

const (
	eq inEQ = iota
	le
	lt
	ge
	gt
)

// eq(predicate, value)
// eq(val(varName), value)
// eq(predicate, val(varName))
// eq(count(predicate), value)
// eq(predicate, [val1, val2, ..., valN])
// eq(predicate, [$var1, "value", ..., $varN])

// type QResultT struct {
// 	PKey  util.UID
// 	SortK string
// 	Ty    string
// }

// ==========================================================================================================

// func Count(p Predicate) int {
// 	// parsing will have confirmed p is a uid-pred only
// 	cnt, err := db.GetCount(p)
// 	if err != nil {
// 		panic(err)
// 	}
// 	return cnt

// }

// type ValOut interface{}

// func Val(v Variable) ValOut {
// 	if v, ok := variable[v]; !ok {
// 		panic(fmt.Errorf("%q does not exist as variable", v))
// 	} else {
// 		return v
// 	}
// 	return nil
// }

// Filter EQ function called during execution-filter phase
// predfunc can be either the expression predicate or a modifying function (val, count)
// value is the predicate value as appears in the expression e.g. eq(<pred>, 5) where 5 is the value.
// nv is the contents of the cache (predicate,value) for the UID returned from the root func
// ty is the type of the cache entry
// i,k index into slice ([][]) of edge interface values (overflow representation of edge values)
// bool dictates whether QResult element (represented by nv argument) will be ignored or displayeda
func EQ(predfunc FargI, value interface{}, nv ds.NVmap, ty string, j, k int) bool {
	return ieq(eq, predfunc, value, nv, ty, j, k)
}
func GT(predfunc FargI, value interface{}, nv ds.NVmap, ty string, j, k int) bool {
	return ieq(gt, predfunc, value, nv, ty, j, k)
}
func GE(predfunc FargI, value interface{}, nv ds.NVmap, ty string, j, k int) bool {
	return ieq(ge, predfunc, value, nv, ty, j, k)
}
func LT(predfunc FargI, value interface{}, nv ds.NVmap, ty string, j, k int) bool {
	return ieq(lt, predfunc, value, nv, ty, j, k)
}
func LE(predfunc FargI, value interface{}, nv ds.NVmap, ty string, j, k int) bool {
	return ieq(le, predfunc, value, nv, ty, j, k)
}

//ieq represents the common logic for inequality functions e.g. eq, lt, gt, ge, le
// ie - inequality value from calling func
// predFunc - argument to inequality function. consists of a predicate or innerfunction like count()
// value - literal value from GQL statement to be compared to data-cache (nv) value
// nv - GQL predicates and their associated node cache data
// ty - type of result item from root query
// j,k - index into node cache map for uid-pred predicates
//
func ieq(ie inEQ, predfunc FargI, value interface{}, nv ds.NVmap, ty string, j, k int) bool {
	var (
		pTy block.TyAttrD
		ok  bool
	)

	if ty == "" {
		panic("Error in Func: expected a type, got nil")
	}

	fmt.Println("In ieq: for - ", ie, ty)
	fmt.Println(strings.Repeat("-", 80))

	switch x := predfunc.(type) {

	case CountFunc:

		switch x.Arg.(type) {

		case *UidPred:
			var nds [][][]byte
			// get data from nv
			data := nv[x.Name()] //data.Value [][][]byte

			if nds, ok = data.Value.([][][]byte); !ok {
				panic(fmt.Errorf("Expression EQ: Expected [][][]byte for NV value"))
			}
			// count child nodes
			n := 0
			for i, k := range nds {
				n += len(k[i])
			}
			if v, ok := value.(int); ok {
				return v == n
			}

			// case Variable: //TODO: implement
		}

	case ScalarPred:
		// find value for this predicate
		var (
			ok bool
		)
		nm := x.Name()
		fmt.Printf("in ScalarPred with %q   ty: %s\n", nm, ty)
		data, ok := nv[nm]
		if !ok {
			panic(fmt.Errorf("Error in inequality func: predicate %q not found in ds.NV", nm))
		}
		//
		// get type of predicate from type info
		if pTy, ok = cache.TypeC.TyAttrC[ty+":"+nm]; !ok {
			// root result type does not contain filter predicate, so root item fails the filter
			return false
		}
		switch pTy.DT {

		case "S":

			switch {

			case j == -1: // scalar value
				var (
					ok      bool
					dataVal string
					exprVal string
				)
				// root query filter expression - node scalar data
				// check data value (from NV) type matches type of scalarPred from GQL query
				if dataVal, ok = data.Value.(string); !ok {
					panic(fmt.Errorf("predicate %q value type %q does not match type for predicate in type %q", x.Name(), dataVal, ty))
				}
				// expression value
				if exprVal, ok = value.(string); !ok {
					panic(fmt.Errorf("value %q for predicate %q is not the correct type for type %q", exprVal, x.Name(), ty))
				}
				// compare values
				switch ie {
				case eq:
					return dataVal == exprVal
				case gt:
					return dataVal > exprVal
				case ge:
					return dataVal >= exprVal
				case lt:
					return dataVal < exprVal
				case le:
					return dataVal <= exprVal
				}

			default: // uid-pred, args ty, i,j populated have non-zero values
				var (
					ok      bool
					dataVal [][]string
					exprVal string
				)
				// uid-pred filter expression - multiple child scalar data (propagated data format)
				// check if null flag set in data
				if data.Null[j][k] {
					return false
				}
				// data value type
				if dataVal, ok = data.Value.([][]string); !ok {
					panic(fmt.Errorf("predicate %q value type %q does not match type for predicate in type %q", x.Name(), data.Value, ty))
				}
				// expression value
				if exprVal, ok = value.(string); !ok {
					panic(fmt.Errorf("value %q for predicate %q is not the correct type for type %q", exprVal, x.Name(), ty))
				}
				switch ie {
				case eq:
					return dataVal[j][k] == exprVal
				case gt:
					return dataVal[j][k] > exprVal
				case ge:
					return dataVal[j][k] >= exprVal
				case lt:
					return dataVal[j][k] < exprVal
				case le:
					return dataVal[j][k] <= exprVal
				}

			}

			return false

		case "I":

			switch {

			case j == -1: // scalar value
				var (
					ok      bool
					dataVal int64
					exprVal int64
				)
				// root query filter expression - node scalar data
				// check data value (from NV) type matches type of scalarPred from GQL query
				if dataVal, ok = data.Value.(int64); !ok {
					panic(fmt.Errorf("predicate %q value type %q does not match type for predicate in type %q", x.Name(), dataVal, ty))
				}
				// expression value
				if exprVal, ok = value.(int64); !ok {
					if y, ok := value.(int); !ok {
						panic(fmt.Errorf("value %q for predicate %q is not the correct type for type %q", exprVal, x.Name(), ty))
					} else {
						exprVal = int64(y)
					}
				}
				// compare values
				switch ie {
				case eq:
					return dataVal == exprVal
				case gt:
					return dataVal > exprVal
				case ge:
					return dataVal >= exprVal
				case lt:
					return dataVal < exprVal
				case le:
					return dataVal <= exprVal
				}

			default: // uid-pred, args ty, i,j populated have non-zero values
				var (
					ok      bool
					dataVal int64
					exprVal int64
				)
				// uid-pred filter expression - multiple child scalar data (propagated data format)
				// check if null flag set in data
				if data.Null[j][k] {
					return false
				}
				// data value type
				if y, ok := data.Value.([][]int64); !ok {
					if z, ok := data.Value.([][]int); !ok {
						panic(fmt.Errorf("value %q for predicate %q is not the correct type for type %q", exprVal, x.Name(), ty))
					} else {
						dataVal = int64(z[j][k])
					}
				} else {
					dataVal = y[j][k]
				}
				// expression value
				if exprVal, ok = value.(int64); !ok {
					if y, ok := value.(int); !ok {
						panic(fmt.Errorf("value %q for predicate %q is not the correct type for type %q", exprVal, x.Name(), ty))
					} else {
						exprVal = int64(y)
					}
				}
				switch ie {
				case eq:
					return dataVal == exprVal
				case gt:
					return dataVal > exprVal
				case ge:
					return dataVal >= exprVal
				case lt:
					return dataVal < exprVal
				case le:
					return dataVal <= exprVal
				}

			}
			// }
		case "F":
		case "Bl":
		}

	}
	return false
}

// func lt() []ast.Result  { return nil }
// func has() []ast.Result { return nil }

// funcs Var(varNm Variable) ValOut {
// 	item:=variable.Get(varNm)
// 	switch x:=item.(type);x {
// 	case ast.EdgeT:
// 	default:

// 	}
// }

func HAS(predfunc FargI, value interface{}, nv ds.NVmap, ty string, j, k int) bool    { return false }
func UID(predfunc FargI, value interface{}, nv ds.NVmap, ty string, j, k int) bool    { return false }
func UID_IN(predfunc FargI, value interface{}, nv ds.NVmap, ty string, j, k int) bool { return false }
func VAL(predfunc FargI, value interface{}, nv ds.NVmap, ty string, j, k int) bool    { return false }
func ANYOFTERMS(predfunc FargI, value interface{}, nv ds.NVmap, ty string, j, k int) bool {
	return false
}
func ALLOFTERMS(predfunc FargI, value interface{}, nv ds.NVmap, ty string, j, k int) bool {
	return false
}
