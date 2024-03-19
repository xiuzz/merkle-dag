package merkledag

import (
	"encoding/json"
	"strings"
)

// Hash to file

// example a path : /doc/tmp/temp.txt
func Hash2File(store KVStore, hash []byte, path string, hp HashPool) []byte {
	// 根据hash和path， 返回对应的文件, hash对应的类型是tree
	flag, _ := store.Has(hash)
	if flag {
		objBinary, _ := store.Get(hash)
		var obj Object
		json.Unmarshal(objBinary, &obj)
		pathArr := strings.Split(path, "/")
		cur := 0
		if cur >= len(pathArr) {
			return nil
		}
		index := 0
		for i := range obj.Links {
			objType := string(obj.Data[index:index+8])
			index += 8
			objInfo := obj.Links[i]
			if objType == "tree" {
				objDirBinary, _ := store.Get(objInfo.Hash)
				var objDir Object
				json.Unmarshal(objDirBinary, &objDir)
				ans := getFileByDir(objDir, pathArr, cur+1, store)
				return ans
			} else if objType == "blob" {
				if objInfo.Name == pathArr[cur] {
					ans, _ := store.Get(objInfo.Hash)
					return ans
				}
			} else { //link
				if objInfo.Name == pathArr[cur] {
					objLinkBinary, _ := store.Get(objInfo.Hash)
					var objLink Object
					json.Unmarshal(objLinkBinary, &objLink)
					ans := getFileByLink(objLink, store)
					return ans
				}
			}
		}
	}
	return nil
}

func getFileByLink(obj Object, store KVStore) []byte {
	ans := make([]byte, 0)
	index := 0
	for i := range obj.Links {
		curObjType := string(obj.Data[index:index+8])
		index += 8
		curObjLink := obj.Links[i]
		curObjBinary, _ := store.Get(curObjLink.Hash)
		var curObj Object
		json.Unmarshal(curObjBinary, &curObj)
		if curObjType == "blob" {
			ans = append(ans, curObjBinary...)
		} else {
			tmp := getFileByLink(curObj, store)
			ans = append(ans, tmp...)
		}
	}
	return ans
}

func getFileByDir(obj Object, pathArr []string, cur int, store KVStore) []byte {
	if cur >= len(pathArr) {
		return nil
	}
	index := 0
	for i := range obj.Links {
		objType := string(obj.Data[index:index+8])
		index += 8
		objInfo := obj.Links[i]
		if objType == "tree" {
			objDirBinary, _ := store.Get(objInfo.Hash)
			var objDir Object
			json.Unmarshal(objDirBinary, &objDir)
			ans := getFileByDir(objDir, pathArr, cur+1, store)
			return ans
		} else if objType == "blob" {
			if objInfo.Name == pathArr[cur] {
				ans, _ := store.Get(objInfo.Hash)
				return ans
			}
		} else { //link
			if objInfo.Name == pathArr[cur] {
				objLinkBinary, _ := store.Get(objInfo.Hash)
				var objLink Object
				json.Unmarshal(objLinkBinary, &objLink)
				ans := getFileByLink(objLink, store)
				return ans
			}
		}
	}
	return nil
}
