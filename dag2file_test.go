package merkledag

import (
	"fmt"
	"testing"
)

func TestDag2file(t *testing.T) {
	tmp := []string{"曾", "tree", "link"}
	var ans []byte
	for i := 0; i < 3; i++ {
		ans = append(ans, []byte(tmp[i])...)
	}
	// fmt.Println(len(ans))
	for i := 0; i < len(ans); i++ {
		fmt.Printf("%x\n", ans[i])
	}
}
