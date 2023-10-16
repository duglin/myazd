package main

import (
	"encoding/json"

	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
	//log "github.com/duglin/dlog"
	// "golang.org/x/term"
)

func FlagAsString(cmd *cobra.Command, name string) string {
	val, err := cmd.Flags().GetString(name)
	NoErr(err)
	return val
}

func ToJson(obj interface{}) string {
	data, _ := json.MarshalIndent(obj, "", "  ")
	return string(data)
}

func Prompt(str string) byte { // lowercase letter
	b := []byte{'0'}
	fmt.Printf("%s ", str)

	// oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	// NoErr(err)
	// // defer term.Restore(int(os.Stdin.Fd()), oldState)
	// _, err := os.Stdin.Read(b)
	// term.Restore(int(os.Stdin.Fd()), oldState)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if len(line) > 0 {
		b[0] = line[0]
	}
	// fmt.Printf("\n")

	NoErr(err)

	if b[0] == 0x03 { // ctrl-c
		os.Exit(1)
	}

	if b[0] >= 'A' && b[0] <= 'Z' {
		return b[0] - 'A' + 'a'
	}

	return b[0]
}

func BoolPtr(val bool) *bool       { return &val }
func IntPtr(val int) *int          { return &val }
func StringPtr(str string) *string { return &str }
func NilStringPtr(str string) *string {
	if str == "" {
		return nil
	}
	return &str
}

func QuoteStrings(strs []string) string {
	res := ""
	for i, s := range strs {
		if i > 0 {
			res += " "
		}
		buf := bytes.Buffer{}
		buf.WriteString("\"")
		for ch := range s {
			if ch == '"' {
				buf.WriteString("\\")
			}
			buf.WriteByte(byte(ch))
		}
		buf.WriteString("\"")
		res += buf.String()
	}
	return res
}

func ParseQuotedString(str string) []string {
	words := []string{}

	word := bytes.Buffer{}
	inQuote := false
	esc := false
	for ch := range str {
		if !inQuote && ch != '"' && ch != ' ' {
			panic(fmt.Sprintf("Bad char %q in %q", ch, str))
		}
		if ch == '\\' {
			if esc {
				word.WriteByte('\\')
			}
			esc = !esc
			continue
		}
		if ch == '"' {
			if !inQuote {
				inQuote = true
				continue
			}
			inQuote = false
			words = append(words, word.String())
			word.Reset()
			continue
		}
		if esc {
			word.WriteByte('\\')
			esc = false
		}
		word.WriteByte(byte(ch))
	}

	if len(words) == 0 {
		words = nil
	}
	return words
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

func NoErr(err error, args ...interface{}) {
	if err == nil {
		return
	}
	if len(args) == 0 {
		args = []interface{}{err.Error()}
	}
	ErrStop(args[0].(string), args[1:]...)
}

func ErrStop(format string, args ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
