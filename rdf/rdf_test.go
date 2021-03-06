package rdf

import (
	"fmt"
	"os"
	"testing"
	"time"
)

func TestLoadFile(t *testing.T) {

	f, err := os.Open("person.rdf")
	if err != nil {
		t.Fatal(err)
	}
	t0 := time.Now()
	err = Load(f)
	t1 := time.Now()
	fmt.Println("Duration: ", t1.Sub(t0))
	if err != nil {
		t.Fatal()
	}
	t.Log("Finished...")

	//	time.Sleep(4 * time.Second)
}
