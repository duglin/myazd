package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	//log "github.com/duglin/dlog"

	"golang.org/x/term"
)

func ToJson(obj interface{}) string {
	data, _ := json.MarshalIndent(obj, "", "  ")
	return string(data)
}

func Prompt(str string) byte { // lowercase letter
	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	NoErr(err)
	defer term.Restore(int(os.Stdin.Fd()), oldState)

	fmt.Printf("%s", str)
	b := []byte{'0'}
	_, err = os.Stdin.Read(b)
	fmt.Print("\r\n")
	NoErr(err)

	if b[0] == 0x03 { // ctrl-c
		term.Restore(int(os.Stdin.Fd()), oldState)
		os.Exit(1)
	}

	if b[0] >= 'A' && b[0] <= 'Z' {
		return b[0] - 'A' + 'a'
	}

	return b[0]
}

func StringPtr(str string) *string { return &str }

func QuoteStrings(strs []string) string {
	res := ""
	for i, s := range strs {
		if i > 0 {
			res += " "
		}
		res += fmt.Sprintf("%q", s)
	}
	return res
}

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
