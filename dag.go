package merkledag

import (
	"encoding/json"
	"hash"
)

const (
	LIST_LIMIT  = 2048
	BLOCK_LIMIT = 256 * 1024
)

const (
	BLOB = "blob"
	LIST = "list"
	TREE = "tree"
)

type Link struct {
	Name string
	Hash []byte
	Size int
}

type Object struct {
	Links []Link
	Data  []byte
}

func Add(store KVStore, node Node, h hash.Hash) []byte {
	// TODO 将分片写入到KVStore中，并返回Merkle Root
	switch node.Type() {
	case FILE:
		file := node.(File)
		tmp := sliceFile(file, store, h)
		jsonMarshal, _ := json.Marshal(tmp)
		h.Reset()
		h.Write(jsonMarshal)
		return h.Sum(nil)
	case DIR:
		dir := node.(Dir)
		tmp := sliceDir(dir, store, h)
		jsonMarshal, _ := json.Marshal(tmp)
		h.Reset()
		h.Write(jsonMarshal)
		return h.Sum(nil)
	}
	return nil
}

func sliceFile(node File, store KVStore, h hash.Hash) *Object {
	if len(node.Bytes()) <= BLOCK_LIMIT {
		data := node.Bytes()
		blob := Object{
			Links: nil,
			Data:  data,
		}
		jsonMarshal, _ := json.Marshal(blob)
		h.Reset()
		h.Write(jsonMarshal)
		flag, _ := store.Has(h.Sum(nil))
		if !flag {
			store.Put(h.Sum(nil), data)
		}
		return &blob
	}
	//list
	linkLen := (len(node.Bytes()) + (BLOCK_LIMIT - 1)) / BLOCK_LIMIT
	hight := 0
	tmp := linkLen
	for {
		hight++
		tmp /= LIST_LIMIT
		if tmp == 0 {
			break
		}
	}
	seedId := 0
	res, _ := dfsForSliceList(hight, node, store, &seedId, h)
	return res
}

func dfsForSliceList(hight int, node File, store KVStore, seedId *int, h hash.Hash) (*Object, int) {
	if hight == 1 {
		return unionBlob(node, store, seedId, h)
	} else { // > 1 depth list
		list := &Object{}
		lenData := 0
		for i := 1; i <= LIST_LIMIT && *seedId < len(node.Bytes()); i++ {
			tmp, lens := dfsForSliceList(hight-1, node, store, seedId, h)
			lenData += lens
			jsonMarshal, _ := json.Marshal(tmp)
			h.Reset()
			h.Write(jsonMarshal)
			list.Links = append(list.Links, Link{
				Hash: h.Sum(nil),
				Size: lens,
			})
			typeName := LIST
			if tmp.Links == nil {
				typeName = BLOB
			}
			list.Data = append(list.Data, []byte(typeName)...)
		}
		jsonMarshal, _ := json.Marshal(list)
		h.Reset()
		h.Write(jsonMarshal)
		flag, _ := store.Has(h.Sum(nil))
		if !flag {
			store.Put(h.Sum(nil), jsonMarshal)
		}
		return list, lenData
	}
}

func unionBlob(node File, store KVStore, seedId *int, h hash.Hash) (*Object, int) {
	// only 1 blob
	if (len(node.Bytes()) - *seedId) <= BLOCK_LIMIT {
		data := node.Bytes()[*seedId:]
		blob := Object{
			Links: nil,
			Data:  data,
		}
		jsonMarshal, _ := json.Marshal(blob)
		h.Reset()
		h.Write(jsonMarshal)
		flag, _ := store.Has(h.Sum(nil))
		if !flag {
			store.Put(h.Sum(nil), data)
		}
		return &blob, len(data)
	}
	// > 1 blob
	list := &Object{}
	lenData := 0
	for i := 1; i <= LIST_LIMIT && *seedId < len(node.Bytes()); i++ {
		end := *seedId + BLOCK_LIMIT
		if len(node.Bytes()) < end {
			end = len(node.Bytes())
		}
		data := node.Bytes()[*seedId:end]
		blob := Object{
			Links: nil,
			Data:  data,
		}
		lenData += len(data)
		jsonMarshal, _ := json.Marshal(blob)
		h.Reset()
		h.Write(jsonMarshal)
		flag, _ := store.Has(h.Sum(nil))
		if !flag {
			store.Put(h.Sum(nil), data)
		}
		list.Links = append(list.Links, Link{
			Hash: h.Sum(nil),
			Size: len(data),
		})
		list.Data = append(list.Data, []byte(BLOB)...)
		*seedId += BLOCK_LIMIT
	}
	jsonMarshal, _ := json.Marshal(list)
	// fmt.Println(node.Name(), len(jsonMarshal) / 1024)
	h.Reset()
	h.Write(jsonMarshal)
	flag, _ := store.Has(h.Sum(nil))
	if !flag {
		store.Put(h.Sum(nil), jsonMarshal)
	}
	return list, lenData
}

func sliceDir(node Dir, store KVStore, h hash.Hash) *Object {
	iter := node.It()
	treeObject := &Object{}
	for iter.Next() {
		node := iter.Node()
		switch node.Type() {
		case FILE:
			file := node.(File)
			tmp := sliceFile(file, store, h)
			jsonMarshal, _ := json.Marshal(tmp)
			h.Reset()
			h.Write(jsonMarshal)
			treeObject.Links = append(treeObject.Links, Link{
				Hash: h.Sum(nil),
				Size: int(file.Size()),
				Name: file.Name(),
			})
			typeName := LIST
			if tmp.Links == nil {
				typeName = BLOB
			}
			treeObject.Data = append(treeObject.Data, []byte(typeName)...)
		case DIR:
			dir := node.(Dir)
			tmp := sliceDir(dir, store, h)
			jsonMarshal, _ := json.Marshal(tmp)
			h.Reset()
			h.Write(jsonMarshal)
			treeObject.Links = append(treeObject.Links, Link{
				Hash: h.Sum(nil),
				Size: int(dir.Size()),
				Name: dir.Name(),
			})
			typeName := TREE
			treeObject.Data = append(treeObject.Data, []byte(typeName)...)
		}
	}
	jsonMarshal, _ := json.Marshal(treeObject)
	h.Reset()
	h.Write(jsonMarshal)
	flag, _ := store.Has(h.Sum(nil))
	if !flag {
		store.Put(h.Sum(nil), jsonMarshal)
	}
	return treeObject
}
