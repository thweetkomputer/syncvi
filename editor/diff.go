package editor

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"github.com/sergi/go-diff/diffmatchpatch"
)

func diff(buffer, newBuffer []rune) []byte {
	bufferStr := string(buffer)
	newBufferStr := string(newBuffer)
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(bufferStr, newBufferStr, true)

	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)

	err := enc.Encode(diffs)
	if err != nil {
		fmt.Println("Error encoding diffs:", err)
		return nil
	}

	return buf.Bytes()
}

func patch(buffer []rune, diff []byte) []rune {
	bufferStr := string(buffer)
	dmp := diffmatchpatch.New()

	var buf bytes.Buffer
	buf.Write(diff)
	dec := gob.NewDecoder(&buf)

	var decodedDiffs []diffmatchpatch.Diff
	err := dec.Decode(&decodedDiffs)
	if err != nil {
		fmt.Println("Error decoding diffs:", err)
		return nil
	}

	patch := dmp.PatchMake(bufferStr, decodedDiffs)
	patchedText, _ := dmp.PatchApply(patch, bufferStr)
	return []rune(patchedText)
}
