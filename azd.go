package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	// "log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"
)

var Verbose = 1
var PropFile = "azd.config"
var Properties map[string]string = map[string]string{}
var Token string = ""
var Subscription string = ""
var ResourceGroup = ""
var TabWriter = tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)

func Log(depth int, format string, args ...interface{}) {
	if depth > Verbose {
		return
	}
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, format, args...)
}

func ErrStop(format string, args ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func loadProperties() {
	data, err := os.ReadFile(PropFile)
	if err != nil {
		if os.IsNotExist(err) {
			return
		}
		ErrStop("Error reading %s: %s", PropFile, err)
		return
	}
	tmpProps := map[string]string{}

	err = json.Unmarshal(data, &tmpProps)
	if err != nil {
		ErrStop("Error reading property file: %s", err)
	}
	for k, v := range tmpProps {
		Properties[strings.ToUpper(k)] = v
	}

	/*
		aliases := map[string][]string{
			"SUBSCRIPTION":  []string{"SUB"},
			"RESOURCEGROUP": []string{"RG"},
		}

		for k, vs := range aliases {
			if value := Properties[strings.ToUpper(k)]; value != "" {
				for _, alias := range vs {
					Properties[strings.ToUpper(alias)] = value
				}
			}
		}
	*/

	Subscription = Properties["subscription"]
}

func getToken() {
	// az account get-access-token -s $SUB -o tsv | sed 's/\t.*//'
	cmd := exec.Command("az", "account", "get-access-token", "-s", Subscription,
		"-o", "tsv")
	out, err := cmd.CombinedOutput()
	if err != nil {
		ErrStop("Error getting token: %s\n", err)
	}
	Token, _, _ = strings.Cut(string(out), "\t")
	if Token == "" {
		ErrStop("Token is empty something went wrong")
	}
	Log(3, "Token: %s", Token[:5])
}

func setupCmds() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "azd",
		Short: "Demo azd command",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("Show help text\n")
		},
	}
	rootCmd.PersistentFlags().IntVarP(&Verbose, "verbose", "v", 1,
		"Verbose value")

	applyCmd := &cobra.Command{
		Use:   "apply",
		Short: "Apply resource file",
		Run:   CRUDFunc,
	}
	applyCmd.Flags().StringArrayP("file", "f", nil, "List of resource files")
	rootCmd.AddCommand(applyCmd)

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete resources",
		Run:   CRUDFunc,
	}
	deleteCmd.Flags().StringArrayP("file", "f", nil, "List of resource files")
	rootCmd.AddCommand(deleteCmd)

	getCmd := &cobra.Command{
		Use:   "get",
		Short: "Get resources",
		Run:   CRUDFunc,
	}
	getCmd.Flags().StringArrayP("file", "f", nil, "List of resource files")
	rootCmd.AddCommand(getCmd)

	return rootCmd
}

type ResourceDef struct {
	URL      string
	Defaults map[string]string
}

var Resources = map[string]*ResourceDef{
	"ACA.App": &ResourceDef{
		URL: "https://management.azure.com/subscriptions/${SUBSCRIPTION}/resourceGroups/${RESOURCEGROUP}/providers/Microsoft.App/containerApps/${NAME}?api-version=${APIVERSION}",
		Defaults: map[string]string{
			"APIVERSION": "2022-11-01-preview",
		},
	},
	"Redis": &ResourceDef{
		URL: "https://management.azure.com/subscriptions/${SUBSCRIPTION}/resourceGroups/${RESOURCEGROUP}/providers/Microsoft.Cache/redis/${NAME}?api-version=${APIVERSION}",
		Defaults: map[string]string{
			"APIVERSION": "2023-04-01",
		},
	},
}

// Sub recursive history
var history = map[string]bool{}

func doSubs(str string, props map[string]string) string {
	result := strings.Builder{}

	// action: 0=add to result
	// action: 1=add to varName
	// action: 2=add to varName, do subs
	// action: 3=add varName to result
	stateTable := [][]int{
		// ACTION/NextState
		//     $   {   }  else
		[]int{11, 00, 00, 00}, // row 0 - default state, not in var
		[]int{30, 12, 30, 11}, // row 1 - seen $
		[]int{30, 30, 20, 12}, // row 2 - seen ${, in var
	}

	row := 0
	col := 0
	varName := strings.Builder{}
	for _, ch := range str {
		switch ch {
		case '$':
			col = 0
		case '{':
			col = 1
		case '}':
			col = 2
		default:
			col = 3
		}

		state := stateTable[row][col]
		row = state % 10
		switch int(state / 10) {
		case 0:
			result.WriteRune(ch)
		case 1:
			varName.WriteRune(ch)
		case 2:
			v := strings.ToUpper(varName.String()[2:])
			value := props[v]
			if history[v] == true {
				ErrStop("Recurive variable substitution: %s", v)
			}
			history[v] = true
			value = doSubs(value, props)
			delete(history, v)

			// fmt.Printf("VAR: %s -> %s\n", v, value)
			result.WriteString(value)
			varName.Reset()
		case 3:
			varName.WriteRune(ch)
			result.WriteString(varName.String())
			varName.Reset()
		}
	}
	if varName.Len() > 0 {
		result.WriteString(varName.String())
	}

	Log(4, "\nOLD: %s\nNEW: %s", str, result.String())
	return result.String()
}

func readIncludeFile(baseFile string, inc string) ([]byte, error) {
	file := ""
	if baseFile == "-" {
		return os.ReadFile(inc)
	} else if strings.HasPrefix(baseFile, "http") {
		daURL, _ := url.Parse(baseFile)
		path := daURL.Path
		path = strings.TrimRight(path, "/")
		i := strings.LastIndex(path, "/")
		if i == -1 {
			daURL.Path = inc
		} else {
			daURL.Path = path[:i+1] + inc
		}

		res, err := http.Get(daURL.String())
		if err != nil {
			return nil, err
		}
		if res.StatusCode != 200 {
			return nil, fmt.Errorf("Error reading include http file(%s): %s",
				daURL.String(), res.Status)
		}
		return io.ReadAll(res.Body)
	} else {
		i := strings.LastIndex(baseFile, fmt.Sprintf("%c", os.PathSeparator))
		if i == -1 {
			return os.ReadFile(inc)
		}
		return os.ReadFile(baseFile[:i+1] + inc)
	}

	return os.ReadFile(file)
}

func CRUDFunc(cmd *cobra.Command, args []string) {
	getToken()
	files, err := cmd.Flags().GetStringArray("file")
	if err != nil {
		ErrStop(err.Error())
	}

	files = append(files, args...)

	if len(files) == 0 {
		if _, err := os.Stat("resource.json"); err == nil {
			files = []string{"resource.json"}
		} else if _, err := os.Stat("resources.json"); err == nil {
			files = []string{"resources.json"}
		}
	}

	Log(4, "Props:\n%#v\n", Properties)

	if cmd.Use == "get" {
		fmt.Fprintf(TabWriter, "Name\tStatus\n")
	}

	for _, file := range files {
		data := []byte{}

		if file == "-" {
			data, err = io.ReadAll(os.Stdin)
			if err != nil {
				ErrStop("Error reading from stdin: %s", err)
			}
			Properties["file"] = "stdin"
		} else if strings.HasPrefix(file, "http") {
			res, err := http.Get(file)
			if err != nil {
				ErrStop("Error downloading %q: %s", file, err)
			}
			body, _ := io.ReadAll(res.Body)
			if res.StatusCode != 200 {
				ErrStop("Error downloading %q: %s\n%s", file, res.Status,
					string(body))
			}
			data = body
			Properties["file"] = file
		} else {
			Log(2, "Loading %q", file)
			data, err = os.ReadFile(file)
			if err != nil {
				if os.IsNotExist(err) {
					ErrStop("Can't find resource file: %s\n", file)
				}
				ErrStop("Error reading file %s: %s", file, err)
			}

			Properties["file"] = file
		}

		whatIsIt := 0
		for _, ch := range data {
			if ch == '{' {
				whatIsIt = 1 // single resource
				break
			}
			if ch == '[' {
				whatIsIt = 2 // array of resources
				break
			}
			if ch < ' ' {
				continue
			}
		}
		if whatIsIt == 0 {
			ErrStop("Error parsing %s: Not valid JSON. "+
				"Must either be a resource or array of resources\n", file)
		}

		// Either way we'll be using an array for consistency
		resources := []map[string]json.RawMessage{}
		if whatIsIt == 1 {
			res := map[string]json.RawMessage{}
			if err = json.Unmarshal(data, &res); err != nil {
				ErrStop("Error parsing %s: %s", file, err)
			}

			// turn this single resource into an array of resources
			resources = append(resources, res)
		} else {
			if err = json.Unmarshal(data, &resources); err != nil {
				ErrStop("Error parsing %s: %s", file, err)
			}
		}

		// Process "include" statements
		for i, res := range resources {
			inc := string(res["include"])
			json.Unmarshal([]byte(inc), &inc)
			if inc != "" {
				Log(2, "Including: %s", inc)
				// TODO make inc relative to current file
				// data, err := os.ReadFile(inc)
				data, err := readIncludeFile(file, inc)
				if err != nil {
					ErrStop("Error reading include file(%s): %s", inc, err)
				}
				newRes := map[string]json.RawMessage{}
				json.Unmarshal(data, &newRes)
				for k, v := range res {
					if k == "include" {
						continue
					}
					newRes[k] = v
				}
				resources[i] = newRes
			}
		}

		// Now iterate over the array and process each resource, stop on err
		for _, res := range resources {
			if cmd.Use == "apply" {
				err = applyResource(res)
			} else if cmd.Use == "delete" {
				err = deleteResource(res)
			} else if cmd.Use == "get" {
				err = getResource(res)
			} else {
				ErrStop("Unknown cmd: %#v", cmd)
			}
			if err != nil {
				ErrStop("Error process %s: %s", file, err)
			}
		}

	}

	if cmd.Use == "get" {
		TabWriter.Flush()
	}
}

func getAttribute(res map[string]json.RawMessage, attr string, props map[string]string) string {
	js, ok := res[attr]
	if !ok {
		return ""
	}

	value := ""
	err := json.Unmarshal(js, &value)
	if err != nil {
		ErrStop("%q must be a string, not '%s'\n", attr, string(js))
	}

	delete(res, attr)

	props[strings.ToUpper(attr)] = value
	return value
}

func applyResource(res map[string]json.RawMessage) error {
	// Make local copy of Properties
	props := map[string]string{}
	for k, v := range Properties {
		props[k] = v
	}

	resType := getAttribute(res, "type", props)

	Log(3, "resType: %s", resType)
	resDef := Resources[resType]
	resURL := resDef.URL
	for k, v := range props {
		Log(3, "props[%s]=%q", k, v)
	}

	if resDef != nil {
		for k, v := range resDef.Defaults {
			props[k] = v
			Log(3, "%s default: %s=%q", resType, k, v)
		}

		resName := getAttribute(res, "name", props)
		getAttribute(res, "subscription", props)
		getAttribute(res, "apiversion", props)

		data, _ := json.MarshalIndent(res, "", "  ")
		data = []byte(doSubs(string(data), props))
		resURL = doSubs(resURL, props)

		Log(1, "Applying: %s/%s (%s)", resType, resName, props["file"])
		httpRes := doHTTP("PUT", resURL, data)
		if httpRes.ErrorMessage != "" {
			ErrStop(httpRes.ErrorMessage)
		}
	} else {
		ErrStop("What? resURL: %s\n", resURL)
	}

	return nil
}

func deleteResource(res map[string]json.RawMessage) error {
	// Make local copy of Properties
	props := map[string]string{}
	for k, v := range Properties {
		props[k] = v
	}

	resType := getAttribute(res, "type", props)

	Log(3, "resType: %s", resType)
	resDef := Resources[resType]
	resURL := resDef.URL
	for k, v := range props {
		Log(3, "props[%s]=%q", k, v)
	}

	if resDef != nil {
		for k, v := range resDef.Defaults {
			props[k] = v
			Log(3, "%s default: %s=%q", resType, k, v)
		}

		resName := getAttribute(res, "name", props)
		getAttribute(res, "subscription", props)
		getAttribute(res, "apiversion", props)

		data, _ := json.MarshalIndent(res, "", "  ")
		data = []byte(doSubs(string(data), props))
		resURL = doSubs(resURL, props)

		Log(1, "Deleting: %s/%s (%s)", resType, resName, props["file"])
		httpRes := doHTTP("DELETE", resURL, nil) // data)
		if httpRes.ErrorMessage != "" {
			ErrStop(httpRes.ErrorMessage)
		}
	} else {
		ErrStop("What? resURL: %s\n", resURL)
	}

	return nil
}

func getResource(res map[string]json.RawMessage) error {
	// Make local copy of Properties
	props := map[string]string{}
	for k, v := range Properties {
		props[k] = v
	}

	resType := getAttribute(res, "type", props)

	Log(3, "resType: %s", resType)
	resDef := Resources[resType]
	resURL := resDef.URL
	for k, v := range props {
		Log(3, "props[%s]=%q", k, v)
	}

	if resDef != nil {
		for k, v := range resDef.Defaults {
			props[k] = v
			Log(3, "%s default: %s=%q", resType, k, v)
		}

		resName := getAttribute(res, "name", props)
		getAttribute(res, "subscription", props)
		getAttribute(res, "apiversion", props)

		data, _ := json.MarshalIndent(res, "", "  ")
		data = []byte(doSubs(string(data), props))
		resURL = doSubs(resURL, props)

		Log(2, "Getting: %s/%s (%s)", resType, resName, props["file"])
		httpRes := doHTTP("GET", resURL, nil)
		if httpRes.StatusCode == 404 {
			fmt.Fprintf(TabWriter, "%s\t%s\n", resName, "<Not Found>")
		} else {
			if httpRes.ErrorMessage != "" {
				ErrStop(httpRes.ErrorMessage)
			}

			getData := struct {
				ID         string
				Name       string
				Type       string
				Properties map[string]interface{}
			}{}

			err := json.Unmarshal(httpRes.Body, &getData)
			if err != nil {
				ErrStop("Error parsing response: %s\n%s\n", err,
					string(httpRes.Body))
			}

			fmt.Fprintf(TabWriter, "%s\t%s\n",
				resName, getData.Properties["provisioningState"].(string))
		}
	} else {
		ErrStop("What? resURL: %s\n", resURL)
	}

	return nil
}

type HTTPResponse struct {
	RequestVerb string
	RequestURL  string
	Status      string
	StatusCode  int
	Headers     map[string][]string
	Body        []byte

	ErrorMessage string
}

func doHTTP(verb string, URL string, data []byte) *HTTPResponse {
	httpResponse := &HTTPResponse{
		RequestVerb: verb,
		RequestURL:  URL,
	}

	client := &http.Client{}
	req, err := http.NewRequest(verb, URL, bytes.NewReader(data))
	if err != nil {
		httpResponse.ErrorMessage = fmt.Sprintf("Error setting up http "+
			"request: %s\n", err)
		return httpResponse
	}

	req.Header.Add("Authorization", "Bearer "+Token)
	req.Header.Add("Content-Type", "application/json")

	Log(2, "%s %s", verb, URL)
	if len(data) > 0 {
		Log(2, "Data:\n%s", string(data))
	} else {
		Log(2, "Data: <empty>")
	}
	res, err := client.Do(req)
	if err != nil {
		httpResponse.ErrorMessage = fmt.Sprintf("Error sending request: %s",
			err)
		return httpResponse
	}

	body, _ := io.ReadAll(res.Body)
	httpResponse.Status = res.Status
	httpResponse.StatusCode = res.StatusCode
	httpResponse.Body = body
	httpResponse.Headers = res.Header
	Log(2, "Res: %s", res.Status)
	for k, v := range res.Header {
		if len(v) == 1 {
			Log(3, "%s: %v", k, v[0])
		} else {
			Log(3, "%s: %v", k, v)
		}
	}

	tmp := map[string]json.RawMessage{}
	json.Unmarshal(body, &tmp)
	str, _ := json.MarshalIndent(tmp, "", "  ")
	Log(3, "\n%s", string(str))

	if res.StatusCode/100 != 2 {
		// If we can pretty print the error, do so
		msg := ""
		errMsg := struct {
			Error struct {
				Code    string
				Message string
			}
			Errors map[string][]string
		}{}

		err = json.Unmarshal(body, &errMsg)
		if err == nil {
			if errMsg.Error.Message != "" {
				msg = errMsg.Error.Message
			} else {
				e := ""
				for _, v := range errMsg.Errors {
					for _, v1 := range v {
						e += v1 + "\n"
					}
				}
				msg = e
			}
		} else {
			fmt.Println(err)
			// Can't pretty print, so just dump it
			msg = fmt.Sprintf("Error: %s\n%s", res.Status, string(str))
		}

		httpResponse.ErrorMessage = msg
	}

	return httpResponse
}

func main() {
	loadProperties()
	rootCmd := setupCmds()

	if err := rootCmd.Execute(); err != nil {
		ErrStop(err.Error())
	}
}
