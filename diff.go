package main

//
//import (
//	"bytes"
//	"encoding/gob"
//	"fmt"
//	"github.com/sergi/go-diff/diffmatchpatch"
//)
//
//func main() {
//	dmp := diffmatchpatch.New()
//	text1 := "Hello World"
//	text2 := "Hello Go d"
//
//	// Generate diffs
//	diffs := dmp.DiffMain(text1, text2, false)
//
//	// Create a buffer and an encoder
//	var buf bytes.Buffer
//	enc := gob.NewEncoder(&buf)
//
//	// Serialize diffs using GOB
//	err := enc.Encode(diffs)
//	if err != nil {
//		fmt.Println("Error encoding diffs:", err)
//		return
//	}
//
//	fmt.Println("Serialized Diffs (GOB):", buf.Bytes())
//
//	// To deserialize
//	dec := gob.NewDecoder(&buf)
//	var decodedDiffs []diffmatchpatch.Diff
//	err = dec.Decode(&decodedDiffs)
//	if err != nil {
//		fmt.Println("Error decoding diffs:", err)
//		return
//	}
//
//	fmt.Println("Deserialized Diffs:", decodedDiffs)
//
//	// Create a patch
//	patch := dmp.PatchMake(text1, diffs)
//	fmt.Println("Patch:", patch)
//
//	// Apply the patch
//	patchedText, _ := dmp.PatchApply(patch, text1)
//	fmt.Println("Patched Text:", patchedText)
//}
