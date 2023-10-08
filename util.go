package main

import (
	"encoding/json"
	// "fmt"
	"regexp"
	//log "github.com/duglin/dlog"
)

func ToJson(obj interface{}) string {
	data, _ := json.MarshalIndent(obj, "", "  ")
	return string(data)
}

func StringPtr(str string) *string { return &str }

func NotNil(pStr *string) string {
	if pStr == nil {
		return ""
	}
	return *pStr
}

func ShrinkJson(daJson []byte) []byte {
	tmp := json.RawMessage{}

	// Start by serializing it as non-pretty json
	json.Unmarshal(daJson, &tmp)
	daJson, _ = json.Marshal(tmp)

	original := string(daJson)
	re1 := regexp.MustCompile(`([^:])({})`)   // {}
	re2 := regexp.MustCompile(`"[^"]*":\[\]`) // "xxx": []
	re3 := regexp.MustCompile(`"[^"]*":{}`)   // "xxx": {}
	re4 := regexp.MustCompile(`,([\]}])`)     /// ,{}  or  .[]
	for {
		daJson = re1.ReplaceAll(daJson, []byte("$1")) // {}
		daJson = re2.ReplaceAll(daJson, []byte(""))   // {}
		daJson = re3.ReplaceAll(daJson, []byte(""))   // {}
		daJson = re4.ReplaceAll(daJson, []byte("$1"))

		if string(daJson) == original {
			break
		}
		original = string(daJson)
	}

	json.Unmarshal(daJson, &tmp)
	daJson, _ = json.MarshalIndent(tmp, "", "  ")
	return daJson
}
