package uuid

import (
	"context"
	"sync"

	slog "github.com/DynamoGraph/syslog"
	"github.com/DynamoGraph/util"
)

type ndAlias = string // rdf blank-node-id e.g. _:a subject entry in rdf file
type nodeMap map[ndAlias]util.UID

var (
	nodeUID nodeMap
	ReqCh   chan Request
	SaveCh  chan Key
	RespCh  chan util.UID
)

func init() {
	// maps
	nodeUID = make(nodeMap)
	// channels
	ReqCh = make(chan Request)
	SaveCh = make(chan Key)
	RespCh = make(chan util.UID)

}

type Request struct {
	SName  ndAlias
	RespCh chan util.UID
}

type Key struct {
	SName ndAlias
	UID   util.UID
}

func PowerOn(ctx context.Context, wp *sync.WaitGroup, wgEnd *sync.WaitGroup) {
	defer wgEnd.Done()
	var (
		err error
		ok  bool
		req Request
		uid util.UID
	)

	slog.Log("rdfuuid: ", "Powering on...")
	wp.Done()

	for {

		select {

		case req = <-ReqCh:

			//slog.Log("rdfuuid: ", fmt.Sprintf("Request received.. %#v", req))
			if uid, ok = nodeUID[req.SName]; !ok {
				if req.SName == "__" {
					// for dummy uid-pred entry - when node is first created
					uid = []byte(req.SName)
				} else {
					uid, err = util.MakeUID()
					if err != nil {
						panic(err) //TODO - handle errors somehow
					}
				}
				nodeUID[req.SName] = uid
			}

			req.RespCh <- uid

		case <-ctx.Done():

			slog.Log("rdfuuid: ", "Powering down...")
			return

		}
	}
}
