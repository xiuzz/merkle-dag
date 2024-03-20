package merkledag

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func TestDag2file(t *testing.T) {
	kv := &HashMap{
		mp: make(map[string][]byte),
	}
	h := sha256.New()
	// a folder
	kv = &HashMap{
		mp: make(map[string][]byte),
	}
	h.Reset()
	path := "/home/xiuuix/go"
	files, _ := ioutil.ReadDir(path)
	dir := &TestDir{
		list: make([]Node, len(files)),
		name: "/",
	}
	for i, fi := range files {
		newPath := path + "/" + fi.Name()
		if fi.IsDir() {
			context := search(newPath)
			context.name = fi.Name()
			dir.list[i] = context
		} else {
			context, err := os.ReadFile(newPath)
			if err != nil {
				t.Fatal(err)
			}
			file := &TestFile{
				name: fi.Name(),
				data: context,
			}
			dir.list[i] = file
		}
	}
	root := Add(kv, dir, h)
	fmt.Printf("%x\n", root)
	buffer_go := Hash2File(kv, root, "/pkg/mod/bazil.org/fuse@v0.0.0-20200117225306-7b5117fecadc/buffer.go", nil)
	fmt.Println(string(buffer_go))
}
