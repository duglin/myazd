package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"time"
	// "log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"text/tabwriter"

	"github.com/itchyny/gojq"
	"github.com/spf13/cobra"
)

var Verbose = 1
var PropFile = "azd.config"
var Properties map[string]string = map[string]string{}
var Token string = ""
var TabWriter = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)

type Resource struct {
	Subscription  string
	ResourceGroup string
	Type          string
	Name          string
	ApiVersion    string

	FromFile   string // filename, URL or stdin(-)
	FromServer bool

	RawData  []byte
	FullJson map[string]json.RawMessage // Json + extra attrs (sub,rg,type,name)
	Json     map[string]json.RawMessage

	DependsOn []string // [[sub:]rg:]type/name[@api]
}

func NewResource() *Resource {
	return &Resource{
		Subscription:  Properties["SUBSCRIPTION"],
		ResourceGroup: Properties["RESOURCEGROUP"],
	}
}

func NewResourceFromFile(file string) (*Resource, error) {
	r := NewResource()

	if err := r.LoadFromFile(file); err != nil {
		return nil, err
	}

	return r, nil
}

func NewResourceFromServer(sub, rg, typ, name, api string) (*Resource, error) {
	r := &Resource{
		Subscription:  sub,
		ResourceGroup: rg,
		Type:          typ,
		Name:          name,
		ApiVersion:    api,
	}

	if err := r.LoadFromServer(); err != nil {
		return nil, err
	}

	return r, nil
}

func (r *Resource) LoadFromServer() error {
	Log(2, "Downloading: %s/%s/%s/%s@%s",
		r.Subscription, r.ResourceGroup, r.Type, r.Name, r.ApiVersion)

	resDef := getResourceDef(r.Type)
	if resDef == nil {
		return fmt.Errorf("Unknown resource type: %s", r.Type)
	}

	if r.Subscription == "" {
		r.Subscription = Properties["SUBSCRIPTION"]
	}

	if r.ResourceGroup == "" {
		r.ResourceGroup = Properties["RESOURCEGROUP"]
	}

	if r.ApiVersion == "" {
		r.ApiVersion = resDef.Defaults["APIVERSION"]
	}

	props := map[string]string{
		"SUBSCRIPTION":  r.Subscription,
		"RESOURCEGROUP": r.ResourceGroup,
		"NAME":          r.Name,
		"APIVERSION":    r.ApiVersion,
	}
	resURL := newDoSubs(resDef.URL, props)

	httpRes := doHTTP("GET", resURL, nil)
	if httpRes.StatusCode == 404 {
		return nil
	}

	if httpRes.ErrorMessage != "" {
		return fmt.Errorf(httpRes.ErrorMessage)
	}

	r.FromServer = true

	return r.LoadFromBytes(httpRes.Body)
}

func (r *Resource) LoadFromFile(file string) error {
	rawData, err := readJsonFile(file)
	if err != nil {
		return err
	}

	r.FromFile = file
	return r.LoadFromBytes(rawData)
}

func (r *Resource) LoadFromBytes(incoming []byte) error {
	r.RawData = incoming

	rawJson := map[string]json.RawMessage{}
	if err := json.Unmarshal(incoming, &rawJson); err != nil {
		return err
	}

	return r.LoadFromJson(rawJson)
}

func (r *Resource) LoadFromJson(incoming map[string]json.RawMessage) error {
	r.FullJson = incoming

	if includeRaw, ok := r.FullJson["include"]; ok {
		delete(r.FullJson, "include")

		include := ""
		json.Unmarshal(includeRaw, &include)
		if include != "" {
			Log(2, "Including: %s", include)
			// TODO make inc relative to current file
			// TODO support include file w/include statement (recursive)
			data, err := readIncludeFile(r.FromFile, include)
			if err != nil {
				return fmt.Errorf("Error reading include file(%s): %s",
					include, err)
			}
			newRes := map[string]json.RawMessage{}
			json.Unmarshal(data, &newRes)
			for k, v := range newRes {
				if _, ok = r.FullJson[k]; ok {
					// Skip attributes that are already defined locally,
					// they're overrides
					continue
				}
				r.FullJson[k] = v
			}
		}

		// Change our RawData to match the included version
		r.RawData, _ = json.Marshal(r.FullJson)
	}

	// Now make "Json" by removing special attributes from the REST API def'n
	r.Json = map[string]json.RawMessage{}
	for k, v := range incoming {
		r.Json[k] = v
	}

	if val := r.FullJson["subscription"]; string(val) != "" {
		r.Subscription = string(val)
		delete(r.Json, "subscription")
	}
	if val := r.FullJson["resourcegroup"]; string(val) != "" {
		r.ResourceGroup = string(val)
		delete(r.Json, "resourcegroup")
	}
	if val := r.FullJson["type"]; string(val) != "" {
		r.Type = string(val)
		delete(r.Json, "type")
	}
	if val := r.FullJson["name"]; string(val) != "" {
		r.Name = string(val)
		delete(r.Json, "name")
	}
	if val := r.FullJson["apiversion"]; string(val) != "" {
		r.ApiVersion = string(val)
		delete(r.Json, "apiversion")
	}

	return nil
}

func ErrStop(format string, args ...interface{}) {
	if !strings.HasSuffix(format, "\n") {
		format += "\n"
	}
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}

func readJsonFile(file string) ([]byte, error) {
	var data []byte
	var err error
	if file == "" {
		data, err = io.ReadAll(os.Stdin)
	} else {
		data, err = os.ReadFile(file)
	}
	if err != nil {
		return nil, err
	}

	return JsonCDecode(data)
}

func loadProperties() {
	data, err := readJsonFile(PropFile)
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
}

func getToken() {
	// az account get-access-token -s $SUB -o tsv | sed 's/\t.*//'
	cmd := exec.Command("az", "account", "get-access-token", "-s",
		Properties["SUBSCRIPTION"], "-o", "tsv")
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

	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Process a resource file",
		Run:   CRUDFunc,
	}
	addCmd.Flags().StringArrayP("file", "f", nil, "List of resource files")
	rootCmd.AddCommand(addCmd)

	deleteCmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete resources",
		Run:   CRUDFunc,
	}
	deleteCmd.Flags().StringArrayP("file", "f", nil, "List of resource files")
	deleteCmd.Flags().BoolP("ignore", "i", false, "Don't stop on error")
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
	Type     string
	URL      string
	Defaults map[string]string
}

var ResourceAliases = map[string]string{
	"App":         "Microsoft.App/containerApps",
	"Env":         "Microsoft.App/managedEnvironments",
	"Environment": "Microsoft.App/managedEnvironments",
	"Redis":       "Microsoft.Cache/redis",
	"DBAccount":   "Microsoft.DocumentDB/databaseAccounts",
}

var Resources = map[string]*ResourceDef{
	"ResourceGroup": &ResourceDef{
		Type: "ResourceGroup",
		URL:  "https://management.azure.com/subscriptions/${SUBSCRIPTION}/resourcegroups/${NAME}?api-version=${APIVERSION}",
		Defaults: map[string]string{
			"APIVERSION": "2021-04-01",
		},
	},

	"Microsoft.App/managedEnvironments": &ResourceDef{
		Type: "Microsoft.App/managedEnvironments",
		URL:  "https://management.azure.com/subscriptions/${SUBSCRIPTION}/resourceGroups/${RESOURCEGROUP}/providers/Microsoft.App/managedEnvironments/${NAME}?api-version=${APIVERSION}",
		Defaults: map[string]string{
			"APIVERSION": "2022-10-01",
			"WAIT":       "true",
		},
	},

	"Microsoft.App/containerApps": &ResourceDef{
		Type: "Microsoft.App/containerApps",
		URL:  "https://management.azure.com/subscriptions/${SUBSCRIPTION}/resourceGroups/${RESOURCEGROUP}/providers/Microsoft.App/containerApps/${NAME}?api-version=${APIVERSION}",
		Defaults: map[string]string{
			"APIVERSION": "2022-11-01-preview",
		},
	},

	"Microsoft.DocumentDB/databaseAccounts": &ResourceDef{
		Type: "Microsoft.DocumentDB/databaseAccounts",
		URL:  "https://management.azure.com/subscriptions/${SUBSCRIPTION}/resourceGroups/${RESOURCEGROUP}/providers/Microsoft.DocumentDB/databaseAccounts/${NAME}?api-version=${APIVERSION}",
		Defaults: map[string]string{
			"APIVERSION": "2021-04-01-preview",
		},
	},

	"Microsoft.Cache/redis": &ResourceDef{
		Type: "Microsoft.Cache/redis",
		URL:  "https://management.azure.com/subscriptions/${SUBSCRIPTION}/resourceGroups/${RESOURCEGROUP}/providers/Microsoft.Cache/redis/${NAME}?api-version=${APIVERSION}",
		Defaults: map[string]string{
			"APIVERSION": "2023-04-01",
		},
	},
}

func getResourceDef(resType string) *ResourceDef {
	tmp, ok := ResourceAliases[resType]
	if ok {
		resType = tmp
	}
	resDef := Resources[resType]
	if resDef == nil {
		ErrStop("Unknown resource type: %s", resType)
	}
	return resDef
}

func parseVariable() {
	re := regexp.MustCompile(`\${(?:(?:(?:(.*):)?(.*):)?([^@}]+)(?:@([^/}]*))?/)?([^\.}]+)(?:\.([^}]+))?}`)
	// re := regexp.MustCompile(`\${(?:(?:(?:(.*):)?(.*):)?([^@]+)(?:@([^/]*))?/)?([^.]+)(?:\.([^}]+))?}`)
	// re := regexp.MustCompile(`\${\s*(?:(?:(.*):)?(.*):)?([^@]+)(@[^/]*)?/([^.]+)(\.[^}]+)?\s*}`)

	//    re := regexp.MustCompile(`\${\s*([^:]*:)?([^:]*:)?([^@]+)(@[^/]*)?/([^.]+)(\.[^}]+)?\s*}`)
	//	re := regexp.MustCompile(`\${\s*((([^:]*):([^:]*):)|([^:]*:))?([^@]+)(@[^/]*)?/([^.]+)(\.[^}]+)?\s*}`)
	fmt.Printf("%q\n", re.FindStringSubmatch("${sub:rg:Microsoft.Apps/containerapps@ver/qwe.e.r.f}"))
	fmt.Printf("%q\n", re.FindStringSubmatch("${rg:Microsoft.Apps/containerapps@ver/qwe.e.r.f}"))
	fmt.Printf("%q\n", re.FindStringSubmatch("${Microsoft.Apps/containerapps@ver/qwe.e.r.f}"))
	fmt.Printf("%q\n", re.FindStringSubmatch("${Microsoft.Apps/containerapps/qwe.e.r.f}"))
	fmt.Printf("%q\n", re.FindStringSubmatch("${Microsoft.Apps/containerapps/qwe}"))
	fmt.Printf("%q\n", re.FindStringSubmatch("${App/qwe}"))
	fmt.Printf("%q\n", re.FindStringSubmatch("${sub:rg:App@ver/qwe.e.r.f}"))
	fmt.Printf("%q\n", re.FindStringSubmatch("${rg:App@ver/qwe.e.r.f}"))

	fmt.Println(re.FindAllStringSubmatchIndex("hello ${rg:App@ver/qwe.e.r.f} world", -1))
	fmt.Println(re.FindAllStringSubmatchIndex("hello ${NAME} world", -1))
}

// Sub recursive history
var history = map[string]bool{}

func newDoSubs(str string, props map[string]string) string {
	re := regexp.MustCompile(`\${(?:(?:(?:(.*):)?(.*):)?([^@}]+)(?:@([^/}]*))?/)?([^\.}]+)(?:\.([^}]+))?}`)
	indexes := re.FindAllStringSubmatchIndex(str, -1)
	nextIndex := 0
	pos := 0
	result := strings.Builder{}

	Log(4, ">SUB OLD: %s\n", str)

	for {
		if nextIndex >= len(indexes) {
			result.WriteString(str[pos:])
			break
		}

		index := indexes[nextIndex]
		result.WriteString(str[pos:index[0]]) // save up to the $
		pos = index[1]                        // skip to char after ${...}

		sub := extract(str, index[2], index[3])
		rg := extract(str, index[4], index[5])
		resType := extract(str, index[6], index[7])
		apiVer := extract(str, index[8], index[9])
		resName := extract(str, index[10], index[11])
		prop := extract(str, index[12], index[13])

		Log(4, "%s -> sub(%s) rg(%s) type(%s) api(%s) name(%s) prop(%s)\n",
			str[index[0]:index[1]], sub, rg, resType, apiVer, resName, prop)

		if resType == "" {
			// Simple ${NAME}
			varName := strings.ToUpper(resName)
			if history[varName] == true {
				ErrStop("Recurive variable substitution: %s", varName)
			}
			value := props[varName]
			history[varName] = true
			Log(4, "Var: %s -> %s", varName, value)
			value = newDoSubs(value, props)
			delete(history, varName)

			result.WriteString(value)
		} else {
			res := getResourceDef(resType)
			if apiVer == "" {
				apiVer = res.Defaults["APIVERSION"]
				if apiVer == "" {
					ErrStop("Can't determine apiVersion for %q", resType)
				}
			}

			data, err := downloadResource(props["SUBSCRIPTION"],
				props["RESOURCEGROUP"], res.Type, resName, apiVer)
			if err != nil {
				ErrStop("Error downloading resource(%s/%s): %s", res.Type,
					resName, err)
			}

			if data == nil {
				ErrStop("Resource '%s/%s'  was not found", res.Type, resName)
			}

			Log(4, "Res json: %s", string(data))

			Log(4, "Prop: .%s", prop)
			query, err := gojq.Parse("." + prop)
			if err != nil {
				ErrStop("Error in prop(%s): %s", prop, err)
			}

			daJson := map[string]any{}
			err = json.Unmarshal(data, &daJson)
			if err != nil {
				ErrStop("Error in parsing resource: %s", err)
			}

			iter := query.Run(daJson)
			value, ok := iter.Next()
			if !ok {
				ErrStop("Can't find value for %q", prop)
			}
			Log(4, "Value: %s", value)

			// result.WriteString(fmt.Sprintf("%s/%s.%s", res.Type, resName, prop))
			result.WriteString(fmt.Sprintf("%v", value))
		}

		nextIndex++
	}

	Log(4, "<SUB NEW: %s", result.String())

	return result.String()
}

func extract(str string, start int, end int) string {
	if start == -1 || end == -1 {
		return ""
	}
	return str[start:end]
}

func downloadResource(sub, rg, resType, resName, api string) ([]byte, error) {
	Log(2, "Download: %s/%s/%s/%s@%s", sub, rg, resType, resName, api)
	res := getResourceDef(resType)
	props := map[string]string{
		"SUBSCRIPTION":  sub,
		"RESOURCEGROUP": rg,
		"APIVERSION":    api,
		"NAME":          resName,
	}
	resURL := newDoSubs(res.URL, props)

	httpRes := doHTTP("GET", resURL, nil)
	if httpRes.StatusCode == 404 {
		return nil, nil
	}

	if httpRes.ErrorMessage != "" {
		return nil, fmt.Errorf(httpRes.ErrorMessage)
	}

	return httpRes.Body, nil
}

func readIncludeFile(baseFile string, inc string) ([]byte, error) {
	file := ""
	if baseFile == "-" {
		return readJsonFile(inc)
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
			return readJsonFile(inc)
		}
		return readJsonFile(baseFile[:i+1] + inc)
	}

	return readJsonFile(file)
}

func CRUDFunc(cmd *cobra.Command, args []string) {
	getToken()
	files, err := cmd.Flags().GetStringArray("file")
	if err != nil {
		ErrStop(err.Error())
	}

	ignore, _ := cmd.Flags().GetBool("ignore")

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
		fmt.Fprintf(TabWriter, "TYPE\tNAME\tSTATUS\n")
	}

	for _, file := range files {
		data := []byte{}

		if file == "-" {
			data, err = readJsonFile("")
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
			data, err = JsonCDecode(data)
			if err != nil {
				ErrStop(err.Error())
			}
			Properties["file"] = file
		} else {
			Log(2, "Loading %q", file)
			data, err = readJsonFile(file)
			if err != nil {
				if os.IsNotExist(err) {
					ErrStop("Can't find resource file: %s\n", file)
				}
				ErrStop("Error reading file %s: %s", file, err)
			}
			data, _ = JsonCDecode(data)

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

		props := map[string]string{}
		for k, v := range Properties {
			props[k] = v
		}

		// Now iterate over the array and process each resource, stop on err
		for _, res := range resources {
			if string(res["type"]) == `"defaults"` {
				for k, v := range res {
					if k == "type" {
						continue
					}
					val := ""
					err := json.Unmarshal(v, &val)
					if err != nil {
						ErrStop("Defaults %q must be a string, not %q",
							k, v)
					}
					Properties[strings.ToUpper(k)] = newDoSubs(val, Properties)
				}

				continue
			}

			resName := string(res["name"])
			resType := string(res["type"])
			verb := ""

			if cmd.Use == "add" {
				verb = "adding"
				err = addResource(res)
			} else if cmd.Use == "delete" {
				verb = "deleting"
				err = deleteResource(res)
				if err != nil && ignore {
					fmt.Fprintf(os.Stderr, "Error %s %s/%s: %s "+
						"(ignoring)\n", verb, resType, resName, err)
					err = nil
				}
			} else if cmd.Use == "get" {
				verb = "getting"
				err = getResource(res)
			} else {
				ErrStop("Unknown cmd: %#v", cmd)
			}

			if err != nil {
				ErrStop(err.Error())
			}
		}

		Properties = props

	}

	if cmd.Use == "get" {
		TabWriter.Flush()
	}
}

func getAttribute(res map[string]json.RawMessage, attr string, props map[string]string) string {
	Log(4, ">In getAttribute(%v, %s, %v", res, attr, props)
	js, ok := res[attr]
	if !ok {
		Log(4, "-> ''")
		return ""
	}

	value := ""
	err := json.Unmarshal(js, &value)
	if err != nil {
		ErrStop("%q must be a string, not '%s'\n", attr, string(js))
	}

	delete(res, attr)

	value = newDoSubs(value, props)
	props[strings.ToUpper(attr)] = value
	Log(4, "<-> %s", value)
	return value
}

func addResource(res map[string]json.RawMessage) error {
	// Make local copy of Properties
	props := map[string]string{}
	for k, v := range Properties {
		props[k] = v
	}

	resType := getAttribute(res, "type", props)

	Log(3, "resType: %s", resType)

	resDef := getResourceDef(resType)
	if resDef == nil {
		return fmt.Errorf("Unknown resource type: %s", resType)
	}

	resURL := resDef.URL
	for k, v := range props {
		Log(3, "props[%s]=%q", k, v)
	}

	for k, v := range resDef.Defaults {
		props[k] = v
		Log(3, "%s default: %s=%q", resType, k, v)
	}

	resName := getAttribute(res, "name", props)
	getAttribute(res, "subscription", props)
	getAttribute(res, "resourcegroup", props)
	getAttribute(res, "apiversion", props)

	data, _ := json.MarshalIndent(res, "", "  ")
	data = []byte(newDoSubs(string(data), props))
	resURL = newDoSubs(resURL, props)

	Log(1, "Adding: %s/%s (%s)", resType, resName, props["file"])
	httpRes := doHTTP("PUT", resURL, data)
	if httpRes.ErrorMessage != "" {
		return fmt.Errorf("Error adding %s/%s: %s", resType, resName,
			httpRes.ErrorMessage)
	}

	if resDef.Defaults["WAIT"] == "true" {
		for {
			data, err := downloadResource(props["SUBSCRIPTION"],
				props["RESOURCEGROUP"],
				resType, resName,
				props["APIVERSION"])
			if err != nil {
				return fmt.Errorf("Error getting status of %s/%s: %s",
					resType, resName, err)
			}

			getData := struct {
				ID         string
				Name       string
				Type       string
				Properties map[string]interface{}
			}{}

			err = json.Unmarshal(data, &getData)
			if err != nil {
				return fmt.Errorf("Error parsing response adding %s/%s: %s\n%s",
					resType, resName, err, string(data))
			}
			if getData.Properties["provisioningState"].(string) == "Succeeded" {
				break
			}
			time.Sleep(time.Second)
		}
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
	resDef := getResourceDef(resType)
	if resDef == nil {
		return fmt.Errorf("Unknown resource type: %s", resType)
	}

	resURL := resDef.URL
	for k, v := range props {
		Log(3, "props[%s]=%q", k, v)
	}

	for k, v := range resDef.Defaults {
		props[k] = v
		Log(3, "%s default: %s=%q", resType, k, v)
	}

	resName := getAttribute(res, "name", props)
	getAttribute(res, "subscription", props)
	getAttribute(res, "resourcegroup", props)
	getAttribute(res, "apiversion", props)

	// data, _ := json.MarshalIndent(res, "", "  ")
	// data = []byte(newDoSubs(string(data), props))
	resURL = newDoSubs(resURL, props)

	Log(1, "Deleting: %s/%s (%s)", resType, resName, props["file"])
	httpRes := doHTTP("DELETE", resURL, nil) // data)
	if httpRes.ErrorMessage != "" {
		return fmt.Errorf("Error delting %s/%s: %s",
			resType, resName, httpRes.ErrorMessage)
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
	resDef := getResourceDef(resType)

	if resDef == nil {
		return fmt.Errorf("Unknown resource type: %s", resType)
	}

	resURL := resDef.URL
	for k, v := range props {
		Log(3, "props[%s]=%q", k, v)
	}

	for k, v := range resDef.Defaults {
		props[k] = v
		Log(3, "%s default: %s=%q", resType, k, v)
	}

	resName := getAttribute(res, "name", props)
	getAttribute(res, "subscription", props)
	getAttribute(res, "resourcegroup", props)
	getAttribute(res, "apiversion", props)

	// data, _ := json.MarshalIndent(res, "", "  ")
	// data = []byte(newDoSubs(string(data), props))
	resURL = newDoSubs(resURL, props)

	Log(2, "Getting: %s/%s (%s)", resType, resName, props["file"])
	httpRes := doHTTP("GET", resURL, nil)
	if httpRes.StatusCode == 404 {
		fmt.Fprintf(TabWriter, "%s\t%s\t%s\n", resType, resName,
			"<Not Found>")
	} else {
		if httpRes.ErrorMessage != "" {
			return fmt.Errorf("Error getting %s/%s: %s", resType, resName,
				httpRes.ErrorMessage)
		}

		getData := struct {
			ID         string
			Name       string
			Type       string
			Properties map[string]interface{}
		}{}

		err := json.Unmarshal(httpRes.Body, &getData)
		if err != nil {
			return fmt.Errorf("Error parsing response getting %s/%s:%s\n%s",
				resType, resName, err, string(httpRes.Body))
		}

		fmt.Fprintf(TabWriter, "%s\t%s\t%s\n",
			resType, resName,
			getData.Properties["provisioningState"].(string))
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

	Log(2, ">%s %s", verb, URL)
	defer Log(2, "<")
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
