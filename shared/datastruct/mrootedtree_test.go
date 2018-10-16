package datastruct

import (
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)



func TestBasicTree(t *testing.T) {

	t.Run("add first root", func(t *testing.T) {
		mtr := NewMRootTree()
		mtr.PrependElement(EmptyElement{}, nil)

		equals(t, uint64(0), mtr.Height)
		equals(t, 1, len(mtr.GetRoots()))

		rts := mtr.GetRoots()
		root := rts[0]
		equals(t,0, len(root.Parents))
		assert(t, nil == root.Next(), "should not point to anything")
	})

	t.Run("add root and then, child", func(t *testing.T) {
		mtr := NewMRootTree()
		rt, _ := mtr.PrependElement(EmptyElement{}, nil)
		mtr.PrependElement(EmptyElement{}, rt)

		equals(t, uint64(1), mtr.Height)
		equals(t, 1, len(mtr.GetRoots()))

		rts := mtr.GetRoots()

		child := rts[0].Next()
		equals(t,[]*Node{rts[0]}, child.Parents)
		assert(t, nil == child.Next(), "should not point to anything")

		root := rts[0]
		equals(t,[]*Node{}, root.Parents)
		assert(t, child == root.Next(), "should be pointing to child")
	})

}

func TestComplexTreeInserts(t *testing.T) {
	tests := []struct {
		name string
		height uint64
		roots int
		addOrder []int   // node where we insert first and the number of nodes we insert
		longestChainId int // idx of longest chain
	}{
		{
			name: "single long chain",
			height: 99,
			roots: 1,
			addOrder: []int{0, 99},
			longestChainId: 99,
		},
		{
			name: "2-chain, 2 roots",
			height: 99,
			roots: 2,
			addOrder: []int{0, 99, 5, 20},
			longestChainId: 99,
		},
		{
			name: "7 roots",
			height: 105,
			roots: 7,
			addOrder: []int{
				0, 99,
				5, 100,
				120, 7,
				150, 1,
				100, 20,
				70, 20,
				2, 100},
			longestChainId: 199,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			nds := make([]*Node, 0, 100)
			ee := EmptyElement{}
			mtr := NewMRootTree()

			// create a root
			e, _ :=  mtr.PrependElement(ee, nil)
			nds = append(nds, e)

			for i := 0; i < len(test.addOrder); i+= 2 {
				// grab root and start adding n nodes
				root := nds[test.addOrder[i]]
				for j := 0; j < test.addOrder[i+1]; j++ {
					root, _ = mtr.PrependElement(ee, root)
					nds = append(nds, root)
				}
			}

			// verify sanity of tree
			equals(t, test.roots, len(mtr.roots))
			assert(t, nds[test.longestChainId] == mtr.GetLongestChain(), "Longest chain should match")
			equals(t, test.height, mtr.Height)


			// check if all of the nodes are on the tree
			height := mtr.GetLongestChain().Height
			for root := mtr.GetLongestChain(); root != nil; root = root.Next() {
				if _, ok := mtr.Find(root.Id); !ok {
					t.Fail()
				}
				if height != root.Height {
					t.Fail()
				}
				height -=1
			}
		})
	}
}

// Taken from https://github.com/benbjohnson/testing
// assert fails the test if the condition is false.
func assert(tb testing.TB, condition bool, msg string, v ...interface{}) {
	if !condition {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%str:%d: "+msg+"\033[39m\n\n", append([]interface{}{filepath.Base(file), line}, v...)...)
		tb.FailNow()
	}
}

// equals fails the test if exp is not equal to act.
func equals(tb testing.TB, exp, act interface{}) {
	if !reflect.DeepEqual(exp, act) {
		_, file, line, _ := runtime.Caller(1)
		fmt.Printf("\033[31m%str:%d:\n\n\texp: %#v\n\n\tgot: %#v\033[39m\n\n", filepath.Base(file), line, exp, act)
		tb.FailNow()
	}
}