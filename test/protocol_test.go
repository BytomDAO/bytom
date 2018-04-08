// +build functional

package test

import (
	"os"
	"testing"
)

// case1:           |------c1(height=7)
// --------(height=5)
//                  |------------c2(height=9)
func TestForkCase1(t *testing.T) {
	c1, err := declChain("chain1", nil, 0, 7)
	defer os.RemoveAll("chain1")
	if err != nil {
		t.Fatal(err)
	}

	c2, err := declChain("chain2", c1, 5, 9)
	defer os.RemoveAll("chain2")
	if err != nil {
		t.Fatal(err)
	}

	bestBlockHash := c2.BestBlockHash()
	if err := merge(c1, c2); err != nil {
		t.Fatal(err)
	}

	if *c1.BestBlockHash() != *bestBlockHash || *c2.BestBlockHash() != *bestBlockHash {
		t.Fatalf("test fork case1 failed")
	}

	if !c1.InMainChain(9, *bestBlockHash) || !c2.InMainChain(9, *bestBlockHash) {
		t.Fatalf("best block is not in main chain")
	}
}

// case2:            |----c1(height=6)
// ---------(height 5)
//                   |----c2(height=6)
func TestForkCase2(t *testing.T) {
	c1, err := declChain("chain1", nil, 0, 6)
	defer os.RemoveAll("chain1")
	if err != nil {
		t.Fatal(err)
	}

	c2, err := declChain("chain2", c1, 5, 6)
	defer os.RemoveAll("chain2")
	if err != nil {
		t.Fatal(err)
	}

	c1BestBlockHash := c1.BestBlockHash()
	c2BestBlockHash := c2.BestBlockHash()
	if err := merge(c1, c2); err != nil {
		t.Fatal(err)
	}

	if *c1.BestBlockHash() != *c1BestBlockHash || *c2.BestBlockHash() != *c2BestBlockHash {
		t.Fatalf("test fork case2 failed")
	}

	if !c1.InMainChain(6, *c1BestBlockHash) || !c2.InMainChain(6, *c2BestBlockHash) {
		t.Fatalf("best block is not in main chain")
	}
}

func TestBlockSync(t *testing.T) {
	c1, err := declChain("chain1", nil, 0, 5)
	defer os.RemoveAll("chain1")
	if err != nil {
		t.Fatal(err)
	}

	c2, err := declChain("chain2", c1, 5, 8)
	defer os.RemoveAll("chain2")
	if err != nil {
		t.Fatal(err)
	}

	bestBlockHash := c2.BestBlockHash()
	if err := merge(c1, c2); err != nil {
		t.Fatal(err)
	}

	if *c1.BestBlockHash() != *bestBlockHash || *c2.BestBlockHash() != *bestBlockHash {
		t.Fatalf("test block sync failed")
	}

	if !c1.InMainChain(8, *bestBlockHash) || !c2.InMainChain(8, *bestBlockHash) {
		t.Fatalf("test block sync failed, best block is not in main chain")
	}
}
