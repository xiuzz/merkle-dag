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

func (obj *Object) appendActionAsTree(key []byte, length int, childObjName string, childObjType string) {
	obj.Links = append(obj.Links, Link{
		Hash: key,
		Size: length,
		Name: childObjName,
	})
	obj.Data = append(obj.Data, []byte(childObjType)...)
}

func (obj *Object) appendActionAsList(key []byte, length int, childObjType string) {
	obj.appendActionAsTree(key, length, "", childObjType)
}

func Add(store KVStore, node Node, h hash.Hash) []byte {
	// TODO 将分片写入到KVStore中，并返回Merkle Root
	switch node.Type() {
	case FILE:
		file := node.(File)
		root := sliceFile(file, store, h)
		return getKey(root, h)
	case DIR:
		dir := node.(Dir)
		root := sliceDir(dir, store, h)
		return getKey(root, h)
	}
	return nil
}

func getKey(obj *Object, h hash.Hash) []byte {
	key, _ := getKeyAndValue(obj, h)
	return key
}

func getKeyAndValue(obj *Object, h hash.Hash) ([]byte, []byte) {
	jsonMarshal, _ := json.Marshal(obj)
	h.Reset()
	h.Write(jsonMarshal)
	return h.Sum(nil), jsonMarshal
}

func put(store KVStore, key []byte, value []byte, objType string) {
	if objType != TREE && len(value) > BLOCK_LIMIT {
		panic("block over the limit")
	}
	store.Put(key, value)
}

func saveObject(obj *Object, h hash.Hash, store KVStore, objType string) {
	key, value := getKeyAndValue(obj, h)
	flag, _ := store.Has(key)
	if !flag {
		put(store, key, value, objType)
	}
}

func saveBlob(blob *Object, h hash.Hash, store KVStore) {
	key := getKey(blob, h)
	flag, _ := store.Has(key)
	if !flag {
		put(store, key, blob.Data, BLOB)
	}
}

func newBlob(data []byte, h hash.Hash, store KVStore) *Object {
	blob := &Object{
		Links: nil,
		Data: data,
	}
	saveBlob(blob, h, store)
	return blob
}

func checkObjIsBlobOrList(obj *Object) string {
	res := LIST
	if obj.Links == nil {
		res = BLOB
	}
	return res
}
func sliceFile(node File, store KVStore, h hash.Hash) *Object {
	nodeData := node.Bytes()
	nodeLen := len(nodeData)
	if nodeLen <= BLOCK_LIMIT {
		return newBlob(nodeData, h, store)
	}
	//list
	linkLen := (nodeLen + (BLOCK_LIMIT - 1)) / BLOCK_LIMIT
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
			key := getKey(tmp, h)
			typeName := checkObjIsBlobOrList(tmp)
			list.appendActionAsList(key, lens, typeName)
		}
		saveObject(list, h, store, LIST)
		return list, lenData
	}
}

func unionBlob(node File, store KVStore, seedId *int, h hash.Hash) (*Object, int) {
	// only 1 blob
	nodeData := node.Bytes()
	nodeLen := len(nodeData)
	if (nodeLen - *seedId) <= BLOCK_LIMIT {
		data := nodeData[*seedId:]
		return newBlob(data, h, store), len(data)
	}
	// > 1 blob
	list := &Object{}
	lenData := 0
	for i := 1; i <= LIST_LIMIT && *seedId < nodeLen; i++ {
		end := *seedId + BLOCK_LIMIT
		if nodeLen < end {
			end = nodeLen
		}
		data := nodeData[*seedId:end]
		blob := newBlob(data, h ,store)
		lenBlob := len(data)
		lenData += lenBlob
		key := getKey(blob, h)
		saveBlob(blob, h, store)
		list.appendActionAsList(key, lenBlob, BLOB)
		*seedId += BLOCK_LIMIT
	}
	saveObject(list, h, store, LIST)
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
			key := getKey(tmp, h)
			typeName := checkObjIsBlobOrList(tmp)
			treeObject.appendActionAsTree(key, int(file.Size()), file.Name(), typeName)

		case DIR:
			dir := node.(Dir)
			tmp := sliceDir(dir, store, h)
			key := getKey(tmp, h)
			typeName := TREE
			treeObject.appendActionAsTree(key, int(dir.Size()), dir.Name(), typeName)
		}
	}
	saveObject(treeObject, h, store, TREE)
	return treeObject
}