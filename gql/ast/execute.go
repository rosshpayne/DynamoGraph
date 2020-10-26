package ast

import (
	"fmt"
	"sync"

	blk "github.com/DynamoGraph/block"
	"github.com/DynamoGraph/cache"
	"github.com/DynamoGraph/ds"
	mon "github.com/DynamoGraph/gql/monitor"
	"github.com/DynamoGraph/rdf/grmgr"
	"github.com/DynamoGraph/util"
)

type rootResult struct {
	uid   util.UID    //returned from root query
	tyS   string      //returned from root query
	sortk string      // returned from root query
	path  string      // #root (R) #root#Siblings#Friends - not currently used
	nv    ds.ClientNV // the data from the uid item. Populated during execution phase with results of filter operation.
}

// index into node data UL structures (see UnmarshalNodeCache). Used to get node scalar data.
type index struct {
	i, j int
}

func (r *RootStmt) Execute(grl grmgr.Limiter) {
	//
	// execute root func - get back slice of unfiltered results
	//
	fmt.Printf("About to run root func..argument type: .%T %v\n", r.RootFunc.Farg, r.RootFunc.F)

	result := r.RootFunc.F(r.RootFunc.Farg, r.RootFunc.Value)

	fmt.Printf("result: %#v\n", result)

	if len(result) == 0 {
		return
	}
	var wgRoot sync.WaitGroup

	stat := mon.Stat{Id: mon.Candidate, Value: len(result)}
	mon.StatCh <- stat

	for _, v := range result {

		grl.Ask()
		<-grl.RespCh()

		wgRoot.Add(1)
		result := &rootResult{uid: v.PKey, tyS: v.Ty, sortk: v.SortK, path: "root"}

		go r.filterRootResult(grl, &wgRoot, result) // go 	r.filterRootResult(grl, &wgRoot, result)

	}
	wgRoot.Wait()

	fmt.Println("Execute:  wait. over all done ")

}

func (r *RootStmt) filterRootResult(grl grmgr.Limiter, wg *sync.WaitGroup, result *rootResult) {
	var (
		err error
		nc  *cache.NodeCache
	)
	defer wg.Done()
	defer grl.EndR()
	//
	// save: filter-visit-node uid
	//
	// generate NV from GQL stmt - will also hold data from query response once UmarshalNodeCache is run.
	// query->cache->unmarshal(nv)
	//
	nvc := r.genNV()
	for _, v := range nvc {
		fmt.Printf("nvc: %#v\n", *v)
	}

	for k, v := range nvc {
		fmt.Printf("filterRoot: nvm %s %#v\n", k, *v)
	}
	//
	// generate sortk - determines extent of node data to be loaded into cache. Tries to keep it as norrow (specific) as possible.
	//
	sortkS := cache.GenSortK(nvc, result.tyS)
	for _, v := range sortkS {
		fmt.Println("filterRoot - sortk : ", v)
	}
	//fmt.Printf("GenSortK: %#v\n", sortkS)
	//
	// fetch data - with optimised fetch - perform queries sequentially becuase of mutex lock on node map
	//
	gc := cache.GetCache()
	for _, sortk := range sortkS {
		fmt.Println("filterRoot - FetchNodeNonCache for : ", result.uid, sortk)
		stat := mon.Stat{Id: mon.NodeFetch}
		mon.StatCh <- stat
		nc, _ = gc.FetchNodeNonCache(result.uid, sortk) //, result.tyS)
	}
	//
	// assign cached data to NV
	//
	err = nc.UnmarshalNodeCache(nvc, result.tyS)
	if err != nil {
		panic(err)
	}
	//
	if r.Filter != nil && !r.Filter.RootApply(nvc, result.tyS) {
		// clear node from cache.
		// nc.ClearCache() // TODO: implement
		return
	}
	//
	// save nvm to parent
	//
	if r.hasNoData() {
		r.initialise()
	}

	// save nvm to parent
	//
	nvm := r.assignData(result.uid.String(), nvc, index{0, 0})
	//
	//
	stat := mon.Stat{Id: mon.PassRootFilter}
	mon.StatCh <- stat
	//
	var wgNode sync.WaitGroup

	for _, p := range r.Select {

		switch x := p.Edge.(type) {

		case *ScalarPred:
			// do nothing as in cache

		case *UidPred: // child of child, R.N.N
			var (
				aty blk.TyAttrD
				ok  bool
			)
			x.lvl = 1
			fmt.Println("***filterRoot: root select aty =  ", result.tyS+":"+x.Name())
			if aty, ok = cache.TypeC.TyAttrC[result.tyS+":"+x.Name()]; !ok {
				continue // ignore this attribute as it is not in current type
			}
			fmt.Println("***filterRoot: x.select  for ", x.Name())
			for _, p := range x.Select {

				fmt.Println("***filterRoot: get select  ", p.Edge.Name())
				switch y := p.Edge.(type) {

				case *ScalarPred, *Variable:
					// do nothing as UnmarshalNodeCode has already assigned scalar results in n

				case *UidPred:
					// there is an embedded uid-pred at greater than depth 1 which must be resolved.
					// execute query on each x.Name() item and use the propagated uid-pred data to resolve this uid-pred
					var (
						nds [][][]byte
						idx index
					)

					// get type of the uid-pred
					fmt.Printf("********* XYZ uidPred: %s.   type: %v\n\n", y.Name(), aty)
					if aty, ok = cache.TypeC.TyAttrC[aty.Ty+":"+y.Name()]; !ok {
						fmt.Printf("****** . ignore it\n")
						continue // ignore this attribute as it is not in current type
					}
					// uid=pred not in nv for this depth. Must query for each uid stored in nv[i].Value -> [][][]byte and save result in map[uid]ClientNV and save to current uid-pred.
					// e.g uid-pred, "Friends:" - contains all the child uids (+ scalar data) for Friends edge. Friend edge contains current uid-pred i.e its the paraent of current uid-pred.
					// parent uid-pred -> Friends: -> current-uid-pred
					//  grab each child in Friend: and perform query on child to get data for current uid-pred.
					data, ok := nvm[x.Name()+":"]
					if !ok {
						for k, v := range nvm {
							fmt.Printf("nvm: %s  %#v\n", k, *v)
						}
						panic(fmt.Errorf("%q not in NV map", x.Name()+":"))
					}
					if nds, ok = data.Value.([][][]byte); !ok {
						panic(fmt.Errorf("filterRootResult: data.Value is of wrong type"))
					}

					for i, u := range nds {
						// for each child in outer uid-pred (x.Name)
						for j, uid := range u {

							//sloc = scalarKey{i, j}
							if data.State[i][j] == blk.UIDdetached { // soft delete set
								fmt.Println("continue......due to detached XF entry")
								continue
							}
							// i,j - defined key for looking up child node UID in cache block.
							// grl.EndR()
							// grl.Ask()
							// <-grl.RespCh()

							wgNode.Add(1)
							idx = index{i, j}
							fmt.Printf("*********  ROOTFILTER... **************************** in execNode(), %s - %s Depth: %d\n\n", util.UID(uid), aty.Ty, 1)
							y.execNode(grl, &wgNode, result.uid, util.UID(uid), aty.Ty, 2, y.Name(), idx) // go x.execNode(grl, &wgNode, result.uid, util.UID(uid), aty.Ty, 1, y.Name()) /
						}
					}
				}
			}
		}
	}
	fmt.Println("wait on wgNode")
	wgNode.Wait()

	// for _, v := range nvc {
	// 	fmt.Printf("filterRootResult: Exit %#v\n", *v)
	// }
	//r.nv = append(r.nv, nvc)

}

// execNode takes parent node (depth-1)and performs UmarshalCacheNode on its uid-preds.
// ty   type of parent node
// us is the current uid-pred from filterRootResult
// uidp is uid current node - not used anymore.
func (u *UidPred) execNode(grl grmgr.Limiter, wg *sync.WaitGroup, puid, uid util.UID, ty string, lvl int, uidp string, idx index) {

	var (
		err error
		nc  *cache.NodeCache
		nvm ds.NVmap // where map key is NV.Name
		nvc ds.ClientNV
		ok  bool
	)

	uidstr := uid.String() // TODO: chanve to pass uuid into execNode as string

	fmt.Printf("**************************************************** in execNode() %s, %s Depth: %d  current uidpred: %s\n", uidstr, ty, lvl, uidp)
	defer wg.Done()
	//
	u.lvl = lvl // depth in graph as determined from GQL stmt

	if u.hasNoData() {
		u.initialise() // TODO: make concurrent safe - as multiple goroutines could access parent concurrently. INitialise in parser???
	}

	if nvm, nvc, ok = u.getData(uidstr); !ok {
		//
		// first uid-pred in node to be executed. All other uid-preds in this node can ignore fetching data from db as its data was included in the first uid-pred search.
		//
		// generate NV from GQL stmt - for each uid-pred even though this is not strictly necessary. If nvm, nvc was saved to u then it would only need to be generated once for u.
		// query->cache->unmarshal(nv). Generate from parent of u, as it contains u. The uid is sourced from the parent node's relevant uid-pred attribute.
		// we need to perform a query on each uid as it represents the children containing u.
		//
		//nvm, nvc = u.genNV(uidp) // TODO: remove uidp as it is nolonger used
		nvc = u.Parent.genNV()
		for _, v := range nvc {
			fmt.Printf("execNode -  nvc: %#v\n", *v)
		}

		// Note: the NV will contain all uid-preds associated with current u, not just uidp passed in. TODO: remove uidp
		// fmt.Printf("uidpred nvc: %#v\n", nvc)
		//
		// generate sortk - source from node type and NV - merge of two.
		//                  determines extent of node data to be loaded into cache. Tries to keep it as norrow (specific) as possible to minimise RCUs.
		//                  ty is the type of the node (uid passed in)
		//
		sortkS := cache.GenSortK(nvc, ty)
		for _, v := range sortkS {
			fmt.Println("execNode - sortk : ", v)
		}
		//
		// fetch data - with optimised fetch - perform queries sequentially because of mutex lock on node map
		// uid is sourced from the parent uid-pred containing the current uid-pred (u). The uid represents the child uid contained in the parent-uid's "uid-pred:" NV item, which will contain the current uid-pred (u)
		//
		gc := cache.GetCache()
		for _, sortk := range sortkS {
			fmt.Println("execNode - FetchNodeNonCache for : ", uid, sortk)
			stat := mon.Stat{Id: mon.NodeFetch}
			mon.StatCh <- stat
			nc, _ = gc.FetchNodeNonCache(uid, sortk) //, ty)
		}
		//
		// assign cached data to NV
		//
		err = nc.UnmarshalNodeCache(nvc, ty)
		if err != nil {
			panic(err)
		}
		fmt.Printf("\n**** assign to uid-pred nv:\n")
		for k, v := range nvc {
			fmt.Printf("nvc: %d  %#v\n", k, *v)
		}
		//
		// save nvm
		//
		fmt.Printf("*** AssignData: %d %s\n", len(nvm), uidstr)
		nvm = u.assignData(uidstr, nvc, idx)
		//u.data[uidstr] = nvm

	} else {
		fmt.Printf("about to test filter on data = [%#v]\n", nvm)
	}
	// apply filter condition to this edge for uid data
	if u.Filter != nil && u.Filter.Apply(nvm, ty, u.Name()) {
		// set uid-pred to ignore for this node only
		// nvm, _, ok := u.getData(uidstr) // u.data[uidstr]
		// if !ok {
		// 	panic(fmt.Errorf("Error in execNode: Cannot find uid key in data map"))
		// }
		nvm[u.Name()+":"].Ignore = true
	}
	//
	for _, p := range u.Select {
		//
		switch x := p.Edge.(type) {

		case *ScalarPred, *Variable: // R.p ignore, already processed

		case *UidPred:
			// NV entry contains child UIDs i.e nv[upred].Value -> [][][]byte
			var (
				nds [][][]byte
				aty blk.TyAttrD
				idx index
			)
			// get type of the uid-pred
			if aty, ok = cache.TypeC.TyAttrC[ty+":"+x.Name()]; !ok {
				continue // ignore this attribute as it is not in current type
			}
			// results not in nv for this depth in graph. Must query uids stored in nv[i].Value -> [][][]byte
			data, ok := nvm[u.Name()+":"]
			if !ok {
				for k, v := range nvm {
					fmt.Printf("nvm: %s  %#v\n", k, *v)
				}
				panic(fmt.Errorf("%q not in NV map", x.Name()+":"))
			}
			if nds, ok = data.Value.([][][]byte); !ok {
				panic(fmt.Errorf(": data.Value is of wrong type"))
			}

			for i, k := range nds {
				for j, cUid := range k {

					if data.State[i][j] == blk.UIDdetached {
						continue // soft delete set or failed filter condition
					}

					// grl.EndR()
					// grl.Ask()
					// <-grl.RespCh()

					wg.Add(1)
					fmt.Printf("********* DDD **************************** in execNode() %s, %s Depth: %d\n\n", util.UID(cUid), ty, u.lvl+1)
					idx = index{i, j}
					x.execNode(grl, wg, uid, util.UID(cUid), aty.Ty, u.lvl+1, x.Name(), idx)
				}
			}
		}
	}
}
