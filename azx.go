package main

/*
- Should be "noun verb", not "verb noun"
  - E.g. not "azx add aca-app ...", should be "azx aca-app create ..."
  - Allows for custom verbs per noun

*/

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	// "reflect"
	"regexp"
	"strings"
	"text/tabwriter"
	"time"

	log "github.com/duglin/dlog"
	"github.com/itchyny/gojq"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var APP = "azx"
var Properties map[string]string = map[string]string{}
var Token string = ""
var TabWriter = tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
var WhyMarshal = ""

var RootCmd *cobra.Command
var ShowCmd *cobra.Command
var AddCmd *cobra.Command
var UpdateCmd *cobra.Command

var Config = map[string]string{}
var Resources = map[string]ARMResource{}

type ARMParser func([]byte) *ResourceBase // FromARMJson
var RegisteredParsers = []ARMParser{}     // FromARMJson

type ARMResource interface {
	DependsOn() []*ResourceReference
	ToARMJson() string // json
	HideServerFields(ARMResource)
}

/*
type Resource struct {
	Stage    string
	Filename string

	Subscription  string
	ResourceGroup string
	Type          string
	Name          string
	APIVersion    string

	Object ARMResource

	FromFile   string // filename, URL or stdin(-)
	FromServer bool

	RawData  []byte
	FullJson map[string]json.RawMessage // Json + extra attrs (sub,rg,type,name)
	Json     map[string]json.RawMessage

	DependsOn []string // [[sub:]rg:]type/name[@api]
}
*/

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

func getToken() {
	if Token != "" {
		return
	}

	// az account get-access-token -s $SUB -o tsv | sed 's/\t.*//'
	cmd := exec.Command("az", "account", "get-access-token", "-s",
		Properties["SUBSCRIPTION"], "-o", "tsv")
	out, err := cmd.CombinedOutput()
	NoErr(err, "Error getting token: %s\n", err)

	Token, _, _ = strings.Cut(string(out), "\t")
	if Token == "" {
		ErrStop("Token is empty something went wrong")
	}
	log.VPrintf(3, "Token: %s", Token[:5])
}

func setupRootCmds() *cobra.Command {
	RootCmd = &cobra.Command{
		Use:   APP,
		Short: "Demo " + APP + " command",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			v, _ := cmd.Flags().GetInt("verbose")
			log.SetVerbose(v)
		},
	}
	RootCmd.PersistentFlags().IntP("verbose", "v", 0, "Verbose value")
	RootCmd.CompletionOptions.HiddenDefaultCmd = true

	httpCmd := &cobra.Command{
		Use:    "http",
		Short:  "Do an HTTP GET",
		Run:    httpFunc,
		Hidden: true,
	}
	RootCmd.AddCommand(httpCmd)

	setCmd := &cobra.Command{
		Use:   "set",
		Short: "Set configuration/default values",
		Run:   SetFunc,
	}
	RootCmd.AddCommand(setCmd)

	initCmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new project",
		Run:   InitFunc,
	}
	RootCmd.AddCommand(initCmd)

	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Provision all resources",
		Run:   ProvisionFunc,
	}
	RootCmd.AddCommand(upCmd)

	downCmd := &cobra.Command{
		Use:   "down",
		Short: "Deprovision all resources",
		Run:   DeprovisionFunc,
	}
	downCmd.Flags().BoolP("wait", "w", false, "Wait for resources to vanish")
	RootCmd.AddCommand(downCmd)

	diffCmd := &cobra.Command{
		Use:   "diff",
		Short: "Diff resource with azure's version",
		Run:   DiffFunc,
	}
	RootCmd.AddCommand(diffCmd)

	stageCmd := &cobra.Command{
		Use:   "stage",
		Short: "Manage stages",
	}
	RootCmd.AddCommand(stageCmd)

	stageCmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "Show all stages",
		Run:   StageListFunc,
	})

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "Show list of resources in project",
		Run:   ListFunc,
	}
	listCmd.Flags().StringP("output", "o", "", "Format (table*,json)")
	RootCmd.AddCommand(listCmd)

	ShowCmd = &cobra.Command{
		Use:   "show",
		Short: "Show details about a resource",
		// Run:   ShowFunc,
	}
	// ShowCmd.Flags().StringP("output", "o", "pretty", "Format (pretty,json,arm)")
	RootCmd.AddCommand(ShowCmd)

	AddCmd = &cobra.Command{
		Use:   "add",
		Short: "Add a resource",
		// Run:   ResourceAddFunc,
	}
	RootCmd.AddCommand(AddCmd)

	UpdateCmd = &cobra.Command{
		Use:   "update",
		Short: "Update a resource",
	}
	RootCmd.AddCommand(UpdateCmd)

	return RootCmd
}

var ResourceAliases = map[string]string{
	"App":         "Microsoft.App/containerApps",
	"Env":         "Microsoft.App/managedEnvironments",
	"Environment": "Microsoft.App/managedEnvironments",
	"Redis":       "Microsoft.Cache/redis",
	"DBAccount":   "Microsoft.DocumentDB/databaseAccounts",
}

type ResourceDef struct {
	Type     string
	URL      string
	Defaults map[string]string
}

var ResourceDefs = map[string]*ResourceDef{
	"ResourceGroup": &ResourceDef{
		Type: "ResourceGroup",
		URL:  "https://management.azure.com/subscriptions/${SUBSCRIPTION}/resourcegroups/${NAME}?api-version=${APIVERSION}",
		Defaults: map[string]string{
			"APIVERSION": "2021-04-01",
		},
	},
}

func AddResourceDef(def *ResourceDef) {
	ResourceDefs[strings.ToLower(def.Type)] = def
}

func GetResourceDef(resType string) *ResourceDef {
	tmp, ok := ResourceAliases[resType]
	if ok {
		resType = tmp
	}
	resDef := ResourceDefs[strings.ToLower(resType)]
	if resDef == nil {
		ErrStop("Unknown resource type: %s", resType)
	}
	return resDef
}

type ResourceReference struct {
	// [[[[sub:]rg:]type[@apiVer]/]]name[.prop]
	Subscription  string
	ResourceGroup string
	Type          string // provider/type
	APIVersion    string
	Name          string
	Property      string

	Origin string
}

func (rr *ResourceReference) AsID() string {
	return fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/%s",
		rr.Subscription, rr.ResourceGroup, rr.Type, rr.Name)
}

func (rr *ResourceReference) AsURL() string {
	return fmt.Sprintf("https://management.azure.com/subscriptions/%s/resourceGroups/%s/providers/%s/%s?api-version=%s",
		rr.Subscription, rr.ResourceGroup, rr.Type, rr.Name, rr.APIVersion)
}

func (rr *ResourceReference) Populate(ref string) {
	if strings.HasPrefix(ref, "/subscriptions/") {
		// subscriptions/xx/resourceGroups/xx/providers/xx/type/name
		//  0   1       2         3      4     5  6     7
		parts := strings.Split(ref, "/")

		if len(parts) != 8 || parts[0] != "subscriptions" ||
			parts[2] != "resourceGroups" || parts[4] != "providers" {

			ErrStop("Reference %q isn't well formed, should be of "+
				"the form: /subscriptions/??/resourceGroups/??/providers/??/"+
				"??/NAME", ref)
		}
		rr.Subscription = parts[1]
		rr.ResourceGroup = parts[3]
		rr.Type = parts[5] + "/" + parts[6]
		rr.Name = parts[7]
		rr.Origin = ref
		return
	}

	// [[[[sub:]rg:]type[@apiVer]/]]name[.prop]
	prr := ParseResourceReference(ref)
	if prr.Subscription != "" {
		rr.Subscription = prr.Subscription
	}
	if prr.ResourceGroup != "" {
		rr.ResourceGroup = prr.ResourceGroup
	}
	if prr.Type != "" {
		rr.Type = prr.Type
	}
	if prr.APIVersion != "" {
		rr.APIVersion = prr.APIVersion
	}
	if prr.Name != "" {
		rr.Name = prr.Name
	}
	if prr.Property != "" {
		rr.Property = prr.Property
	}
}

func ParseResourceReference(ref string) *ResourceReference {
	// [[[sub:]rg:]type[@apiVer]/]]name[.prop]
	re := regexp.MustCompile(`^(?:(?:(?:(.*):)?(.*):)?([^@}]+)?(?:@([^/}]*))?/)?([^\.}]+)(?:\.([^}]+))?$`)
	strs := re.FindStringSubmatch(ref)

	return &ResourceReference{
		Subscription:  strs[1],
		ResourceGroup: strs[2],
		Type:          strs[3],
		APIVersion:    strs[4],
		Name:          strs[5],
		Property:      strs[6],
		Origin:        ref,
	}
}

func ParseResourceID(ref string) *ResourceReference {
	// /subscriptions/xx/resourceGroups/xx/providers/xx/type/name
	//         0      1       2         3      4     5  6     7
	ref = strings.TrimLeft(ref, "/")
	parts := strings.Split(ref, "/")

	if len(parts) != 8 || parts[0] != "subscriptions" ||
		parts[2] != "resourceGroups" || parts[4] != "providers" {

		ErrStop("Reference %q isn't well formed, should be of "+
			"the form: /subscriptions/??/resourceGroups/??/providers/??/"+
			"??/NAME", ref)
	}
	rr := &ResourceReference{}

	rr.Subscription = parts[1]
	rr.ResourceGroup = parts[3]
	rr.Type = parts[5] + "/" + parts[6]
	rr.Name = parts[7]
	rr.APIVersion = GetResourceDef(rr.Type).Defaults["APIVERSION"]
	rr.Origin = ref

	return rr
}

var ResRefTest = [][]string{
	// test -> Sub, RG, Type, APIVer, Name, Prop
	{"sub:rg:rp/t@api/name.prop", "sub", "rg", "rp/t", "api", "name", "prop"},
	{"rg:rp/t@api/name.prop", "", "rg", "rp/t", "api", "name", "prop"},
	{"rp/t@api/name.prop", "", "", "rp/t", "api", "name", "prop"},
	{"t@api/name.prop", "", "", "t", "api", "name", "prop"},
	{"@api/name.prop", "", "", "", "api", "name", "prop"},
	{"rp/t/name.prop", "", "", "rp/t", "", "name", "prop"},
	{"t/name.prop", "", "", "t", "", "name", "prop"},
	{"name.prop", "", "", "", "", "name", "prop"},
	{"name", "", "", "", "", "name", ""},
}

func init() {
	for _, test := range ResRefTest {
		rr := ParseResourceReference(test[0])
		if rr.Subscription != test[1] || rr.ResourceGroup != test[2] ||
			rr.Type != test[3] || rr.APIVersion != test[4] ||
			rr.Name != test[5] || rr.Property != test[6] {
			ErrStop("RR Test failed: %s -> %+v  should have been: %+v",
				test[0], rr, test[1:])
		}
	}
}

// Sub recursive history
var history = map[string]bool{}

func newDoSubs(str string, props map[string]string) string {
	// ${[[[[sub:]rg:]type[@apiVer]/]]name[.prop]}
	re := regexp.MustCompile(`\${(?:(?:(?:(.*):)?(.*):)?([^@}]+)?(?:@([^/}]*))?/)?([^\.}]+)(?:\.([^}]+))?}`)
	indexes := re.FindAllStringSubmatchIndex(str, -1)
	nextIndex := 0
	pos := 0
	result := strings.Builder{}

	log.VPrintf(4, ">SUB OLD: %s\n", str)

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

		log.VPrintf(4, "%s -> sub(%s) rg(%s) type(%s) api(%s) name(%s) prop(%s)\n",
			str[index[0]:index[1]], sub, rg, resType, apiVer, resName, prop)

		if resType == "" {
			// Simple ${NAME}
			varName := strings.ToUpper(resName)
			if history[varName] == true {
				ErrStop("Recursive variable substitution: %s", varName)
			}
			value := props[varName]
			history[varName] = true
			log.VPrintf(4, "Var: %s -> %s", varName, value)
			value = newDoSubs(value, props)
			delete(history, varName)

			result.WriteString(value)
		} else {
			res := GetResourceDef(resType)
			if apiVer == "" {
				apiVer = res.Defaults["APIVERSION"]
				if apiVer == "" {
					ErrStop("Can't determine apiVersion for %q", resType)
				}
			}

			data, err := downloadResource(props["SUBSCRIPTION"],
				props["RESOURCEGROUP"], res.Type, resName, apiVer)
			NoErr(err, "Error downloading resource(%s/%s): %s", res.Type,
				resName, err)

			if data == nil {
				ErrStop("Resource '%s/%s'  was not found", res.Type, resName)
			}

			log.VPrintf(4, "Res json: %s", string(data))

			log.VPrintf(4, "Prop: .%s", prop)
			query, err := gojq.Parse("." + prop)
			NoErr(err, "Error in prop(%s): %s", prop, err)

			daJson := map[string]any{}
			err = json.Unmarshal(data, &daJson)
			NoErr(err, "Error in parsing resource: %s", err)

			iter := query.Run(daJson)
			value, ok := iter.Next()
			if !ok {
				ErrStop("Can't find value for %q", prop)
			}
			log.VPrintf(4, "Value: %s", value)

			// result.WriteString(fmt.Sprintf("%s/%s.%s", res.Type, resName, prop))
			result.WriteString(fmt.Sprintf("%v", value))
		}

		nextIndex++
	}

	log.VPrintf(4, "<SUB NEW: %s", result.String())

	return result.String()
}

func extract(str string, start int, end int) string {
	if start == -1 || end == -1 {
		return ""
	}
	return str[start:end]
}

func downloadResource(sub, rg, resType, resName, api string) ([]byte, error) {
	log.VPrintf(2, ">Enter: downloadResource(%s/%s/%s?%s)", sub, rg, resType, api)
	defer log.VPrintf(2, "<Exit: downloadResource")

	log.VPrintf(2, "Download: %s/%s/%s/%s@%s", sub, rg, resType, resName, api)
	res := GetResourceDef(resType)
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

/*
func (r *Resource) Save() {
	rID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/%s/containerApps/%s", r.Subscription, r.ResourceGroup, r.Type, r.Name)

	baseField := reflect.ValueOf(r.Object).Elem().FieldByName("ResourceBase")
	base := baseField.Addr().Interface().(*ResourceBase)
	base.ID = rID

	data, _ := json.MarshalIndent(r.Object, "", "  ")
	NoErr(WriteStageFile(r.Stage, r.Filename, data))
}

func (r *Resource) Load() {
}
*/

func ResourceFromFile(stage string, name string) (*ResourceBase, error) {
	data, err := ReadStageFile(stage, name)
	if err != nil {
		return nil, err
	}
	return ResourceFromBytes(stage, name, data)
}

func ResourceFromBytes(stage string, name string, data []byte) (*ResourceBase, error) {
	for _, parser := range RegisteredParsers {
		res := parser(data)
		if res != nil {
			res.Stage = stage
			res.Filename = name
			return res, nil
		}
	}

	tmp := struct{ ID string }{}
	json.Unmarshal(data, &tmp)

	ErrStop("Bad type in stage file: %s/%s (%s)", stage, name, tmp.ID)
	return nil, nil
}

func ReadStageFile(stage string, name string) ([]byte, error) {
	fi := GetConfigDir()
	file := path.Join(fi.Name(), "stage_"+stage, name)
	return os.ReadFile(file)
}

func WriteStageFile(stage string, name string, data []byte) error {
	fi := GetConfigDir()
	file := path.Join(fi.Name(), "stage_"+stage, name)
	return os.WriteFile(file, data, 0644)
}

func GenerateConfigFileName(stage string, name string) string {
	fi := GetConfigDir()
	return path.Join(fi.Name(), "stage_"+stage, name)
}

func GetStageResources(stage string) []*ResourceBase {
	if stage == "" {
		stage = Config["currentStage"]
		if stage == "" {
			ErrStop("No current stage defined")
		}
	}

	fi := GetConfigDir()
	dir := path.Join(fi.Name(), "stage_"+stage)

	entries, err := os.ReadDir(dir)
	NoErr(err, "Error listing stage %q: %s", stage, err)

	result := []*ResourceBase{}
	for _, entry := range entries {
		res, err := ResourceFromFile(stage, entry.Name())
		NoErr(err, "Error reading \"%s/%s\": %s", stage, entry.Name(), err)
		result = append(result, res)
	}

	return result
}

func GetStages() []string {
	fi := GetConfigDir()
	entries, err := os.ReadDir(fi.Name())
	NoErr(err)

	result := []string{}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if !strings.HasPrefix(entry.Name(), "stage_") {
			continue
		}
		name := path.Join(fi.Name(), entry.Name())
		result = append(result, name)
	}
	return result
}

func GetConfigDir() fs.FileInfo {
	configDir := "." + APP
	fi, err := os.Stat(configDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	NoErr(err)

	if !fi.IsDir() {
		ErrStop("%q must be a directory", configDir)
	}

	return fi
}

func CreateConfigDir() fs.FileInfo {
	configDir := "." + APP
	err := os.Mkdir(configDir, 0755)
	NoErr(err)
	fi, err := os.Stat(configDir)
	NoErr(err)
	err = os.Mkdir(path.Join(fi.Name(), "stage_default"), 0755)
	NoErr(err)

	Config["currentStage"] = "default"
	Config["defaults.Subscription"] = "fe108f6a-2bd6-409c-8bfb-8f21dbb7ba0a"
	Config["defaults.ResourceGroup"] = "default"
	Config["defaults.Location"] = "East US"

	SaveConfig()
	return fi
}

func LoadConfig() {
	log.VPrintf(2, ">Enter: LoadConfig")
	defer log.VPrintf(2, "<Exit: LoadConfig")

	fi := GetConfigDir()
	if fi == nil {
		ErrStop("Directory isn't initialized, try: %s init", APP)
	}

	fileName := path.Join(fi.Name(), APP+".config")
	data, err := os.ReadFile(fileName)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		NoErr(err)
	}

	if len(data) != 0 {
		err = json.Unmarshal(data, &Config)
		NoErr(err, "Error loading config file: %s", err)
	}
}

func SaveConfig() {
	log.VPrintf(2, ">Enter: SaveConfig")
	defer log.VPrintf(2, "<Exit: SaveConfig")

	fi := GetConfigDir()
	fileName := path.Join(fi.Name(), APP+".config")
	data, err := json.MarshalIndent(Config, "", "  ")
	NoErr(err)
	err = os.WriteFile(fileName, data, 0644)
	NoErr(err)
}

func httpFunc(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		ErrStop("Must have just one arg - the URL (or PATH)")
	}

	URL := args[0]
	if !strings.HasPrefix(URL, "http:") {
		URL = "https://management.azure.com/" + URL
	}

	httpRes := doHTTP("GET", URL, nil)
	if httpRes.ErrorMessage != "" {
		ErrStop("Error: %s", httpRes.ErrorMessage)
	}

	if httpRes.StatusCode != 200 {
		fmt.Printf("%d %s\n", httpRes.StatusCode, httpRes.Status)
	}
	fmt.Printf("\n%s\n", string(httpRes.Body))
}

func SetFunc(cmd *cobra.Command, args []string) {
	LoadConfig()
	changed := false

	for _, arg := range args {
		before, after, found := strings.Cut(arg, "=")
		changed = true
		if !found {
			delete(Config, before)
		} else {
			Config[before] = after
		}
	}
	if changed {
		SaveConfig()
	}
}

func InitFunc(cmd *cobra.Command, args []string) {
	fi := GetConfigDir()

	if fi != nil {
		ErrStop("Already initialized\n")
	}

	CreateConfigDir()
}

func ProvisionFunc(cmd *cobra.Command, args []string) {
	log.VPrintf(2, ">Enter: ProvisionFunc: %q", args)
	defer log.VPrintf(2, "<Exit: ProvisionFunc")

	LoadConfig()

	if len(args) > 0 {
		stage := Config["currentStage"]
		if stage == "" {
			ErrStop("No current stage defined")
		}

		for _, arg := range args {
			argTmp := strings.ReplaceAll(arg, "/", "-")
			res, err := ResourceFromFile(stage, argTmp+".json")
			NoErr(err, "Error reading %q: %s", arg, err)
			res.Provision()
		}
	} else {
		resources := GetStageResources("")
		for _, res := range resources {
			res.Provision()
		}
	}
}

func DeprovisionFunc(cmd *cobra.Command, args []string) {
	log.VPrintf(2, ">Enter: DeprovisionFunc: %q", args)
	defer log.VPrintf(2, "<Exit: DeprovisionFunc")

	LoadConfig()

	resources := []*ResourceBase{}

	if len(args) > 0 {
		stage := Config["currentStage"]
		if stage == "" {
			ErrStop("No current stage defined")
		}

		for _, arg := range args {
			argTmp := strings.ReplaceAll(arg, "/", "-")
			res, err := ResourceFromFile(stage, argTmp+".json")
			NoErr(err, "Error reading %q: %s", arg, err)
			resources = append(resources, res)
		}
	} else {
		resources = GetStageResources("")
	}

	for _, res := range resources {
		res.Deprovision()
	}

	wait, _ := cmd.Flags().GetBool("wait")
	if wait {
		fmt.Printf("Waiting for them to disappear...\n")
		for len(resources) > 0 {
			time.Sleep(1 * time.Second)
			for i, res := range resources {
				if !res.Exists() {
					resources = append(resources[:i], resources[i+1:]...)
					break
				}
			}
		}
	}
}

func DiffFunc(cmd *cobra.Command, args []string) {
	log.VPrintf(2, ">Enter: DeprovisionFunc: %q", args)
	defer log.VPrintf(2, "<Exit: DeprovisionFunc")

	LoadConfig()

	stage := Config["currentStage"]
	if stage == "" {
		ErrStop("No current stage defined")
	}

	if len(args) == 1 {
		arg := args[0]
		argTmp := strings.ReplaceAll(arg, "/", "-")
		res, err := ResourceFromFile(stage, argTmp+".json")
		NoErr(err, "Error reading %q: %s", arg, err)

		diff, err := res.Diff()
		NoErr(err)
		fmt.Printf("%s\n", diff)
	} else {
		resources := GetStageResources("")
		for _, res := range resources {
			diff, err := res.Diff()
			NoErr(err)
			if len(diff) == 0 {
				continue
			}
			fmt.Printf("%s:\n%s\n", res.NiceType+"/"+res.Name, diff)
		}
	}
}

func StageListFunc(cmd *cobra.Command, args []string) {
	LoadConfig()
	for _, stage := range GetStages() {
		_, file := path.Split(stage)
		if !strings.HasPrefix(file, "stage_") {
			ErrStop("Bad stage name: %s", file)
		}
		stage = file[len("stage_"):]

		isCurrent := ""
		if stage == Config["currentStage"] {
			isCurrent = "*"
		}

		fmt.Printf("%s%s\n", stage, isCurrent)
	}
}

func ListFunc(cmd *cobra.Command, args []string) {
	LoadConfig()

	resources := GetStageResources("")

	output, _ := cmd.Flags().GetString("output")
	if output == "json" {
		res := []interface{}{}
		for _, resource := range resources {
			next := interface{}(nil)
			json.Unmarshal(resource.RawData, &next)
			res = append(res, next)
		}

		str, _ := json.MarshalIndent(res, "", "  ")
		fmt.Printf("%s\n", string(str))
		return
	}

	fmt.Fprintf(TabWriter, "TYPE\tNAME\n")
	for _, resource := range resources {
		fmt.Fprintf(TabWriter, "%s\t%s\n", resource.NiceType, resource.Name)
	}
	TabWriter.Flush()
}

func ResourceAddFunc(cmd *cobra.Command, args []string) {
	LoadConfig()
}

type ResourceBase struct {
	Stage    string `json:"-"`
	Filename string `json:"-"`

	Subscription  string `json:"-"`
	ResourceGroup string `json:"-"`
	Type          string `json:"-"` // Provider/ResourceType
	Name          string `json:"-"`
	APIVersion    string `json:"-"`
	NiceType      string `json:"-"`

	ID      string      `json:"id,omitempty"`
	Object  ARMResource `json:"-"` // Basically "self". Owning ARM Object
	RawData []byte      `json:"-"`
}

func (r *ResourceBase) AsID() string {
	rr := ResourceReference{
		Subscription:  r.Subscription,
		ResourceGroup: r.ResourceGroup,
		Type:          r.Type,
		Name:          r.Name,
		APIVersion:    r.APIVersion,
	}
	return rr.AsID()
}

func (r *ResourceBase) AsURL() string {
	rr := ResourceReference{
		Subscription:  r.Subscription,
		ResourceGroup: r.ResourceGroup,
		Type:          r.Type,
		Name:          r.Name,
		APIVersion:    r.APIVersion,
	}
	return rr.AsURL()
}

func (r *ResourceBase) Save() {
	log.VPrintf(2, ">Enter: Save")
	defer log.VPrintf(2, "<Enter: Save")

	r.ID = r.AsID()
	Resources[r.ID] = r.Object

	depends := r.Object.DependsOn()
	for _, dep := range depends {
		id := dep.AsID()
		res := Resources[id]
		if res == nil {
			log.VPrintf(2, "%q isn't local", dep.Name)

			data, err := downloadResource(dep.Subscription,
				dep.ResourceGroup, dep.Type, dep.Name, dep.APIVersion)
			if err != nil {
				ErrStop("Error downloading %q: %s", id, err)
			}
			if len(data) == 0 {
				ErrStop("Can't find dependency for %s/%s: %s",
					r.NiceType, r.Name, dep.Origin)
			}
		}
	}

	data, _ := json.MarshalIndent(r.Object, "", "  ")
	data = append(data, byte('\n'))
	NoErr(WriteStageFile(r.Stage, r.Filename, data))
	if log.GetVerbose() > 0 {
		fmt.Printf("Saved: %s/%s\n", r.Stage, r.Filename)
	}
}

func (r *ResourceBase) Provision() {
	log.VPrintf(2, ">Enter: RB:Provision (%s)", r.NiceType+"/"+r.Name)
	defer log.VPrintf(2, "<Exit: RB:Provision")

	data := r.Object.ToARMJson()
	resURL := r.AsURL()
	resDef := GetResourceDef(r.Type)

	fmt.Printf("Provision: %s/%s\n", r.NiceType, r.Name)
	log.VPrintf(2, "URL: %s", resURL)
	httpRes := doHTTP("PUT", resURL, []byte(data))
	if httpRes.ErrorMessage != "" {
		ErrStop("Error adding %s/%s: %s\n\n%s", r.NiceType, r.Name,
			httpRes.ErrorMessage, data)
	}

	if resDef.Defaults["WAIT"] == "true" {
		// fmt.Printf("Waiting\n")
		state := ""
		for {
			data, err := downloadResource(r.Subscription, r.ResourceGroup,
				r.Type, r.Name, r.APIVersion)
			NoErr(err, "Error getting status of %s/%s: %s",
				r.NiceType, r.Name, err)

			getData := struct {
				Properties map[string]interface{}
			}{}

			err = json.Unmarshal(data, &getData)
			NoErr(err, "Error parsing response adding %s/%s: %s\n%s",
				r.NiceType, r.Name, err, string(data))

			log.VPrintf(2, "State: %s", getData.Properties["provisioningState"])
			state = getData.Properties["provisioningState"].(string)
			if state != "InProgress" {
				break
			}
			time.Sleep(time.Second)
		}
		if state != "Succeeded" {
			ErrStop("Error provisioning %s/%s", r.NiceType, r.Name)
		}
	}
}

func (r *ResourceBase) Deprovision() {
	log.VPrintf(2, ">Enter: RB:Deprovision (%s)", r.NiceType+"/"+r.Name)
	defer log.VPrintf(2, "<Exit: RB:Deprovision")

	resURL := r.AsURL()

	fmt.Printf("Deprovision: %s/%s\n", r.NiceType, r.Name)
	log.VPrintf(2, "URL: %s", resURL)
	httpRes := doHTTP("DELETE", resURL, nil)
	if httpRes.ErrorMessage != "" {
		ErrStop("Error deleting %s/%s: %s", r.NiceType, r.Name,
			httpRes.ErrorMessage)
	}
}

func (r *ResourceBase) Exists() bool {
	log.VPrintf(2, ">Enter: RB:Exists (%s)", r.NiceType+"/"+r.Name)
	defer log.VPrintf(2, "<Exit: RB:Exists")

	resURL := r.AsURL()

	log.VPrintf(2, "URL: %s", resURL)
	httpRes := doHTTP("GET", resURL, nil)
	if httpRes.ErrorMessage != "" || httpRes.StatusCode != 200 {
		return false
	}
	return true
}

func (r *ResourceBase) Download() ([]byte, error) {
	log.VPrintf(2, ">Enter: RB:Download (%s)", r.NiceType+"/"+r.Name)
	defer log.VPrintf(2, "<Exit: RB:Download")

	resURL := r.AsURL()

	log.VPrintf(2, "URL: %s", resURL)

	data, err := downloadResource(r.Subscription, r.ResourceGroup,
		r.Type, r.Name, r.APIVersion)

	return data, err
}

func (r *ResourceBase) Diff() (string, error) {
	log.VPrintf(2, ">Enter: RB:Diff (%s)", r.NiceType+"/"+r.Name)
	defer log.VPrintf(2, "<Exit: RB:Diff")

	// Save it as ARM Json and then covert it back into a ResourceBase
	tmp := map[string]json.RawMessage{}
	json.Unmarshal([]byte(r.Object.ToARMJson()), &tmp)
	buf, _ := json.Marshal(tmp)
	res, err := ResourceFromBytes(r.Stage, r.NiceType+"/"+r.Name, buf)
	if err != nil {
		return "", err
	}

	// Now get the Azure version
	azureData, err := res.Download()
	NoErr(err, "Error downloading %q: %s", r.NiceType+"/"+r.Name, err)
	if len(azureData) == 0 {
		return "", fmt.Errorf("Not in Azure")
	}
	azure, err := ResourceFromBytes(res.Stage, res.Name, azureData)
	if err != nil {
		return "", err
	}

	if strings.EqualFold(res.ID, azure.ID) {
		azure.ID = res.ID
	}
	azure.Object.HideServerFields(res.Object)

	srcJson, _ := json.MarshalIndent(res.Object, "", "  ")
	tgtJson, _ := json.MarshalIndent(azure.Object, "", "  ")

	srcJson = ShrinkJson(srcJson)
	tgtJson = ShrinkJson(tgtJson)

	return string(Diff("local", srcJson, "azure", tgtJson)), nil
}

func SetJson(obj interface{}, format string, args ...interface{}) {
	str := fmt.Sprintf(format, args...)
	NoErr(json.Unmarshal([]byte(str), obj))
}

func SetStringProp(obj any, fs *pflag.FlagSet, flag string, jsonPath string) bool {
	if !fs.Changed(flag) {
		return false
	}

	tmp, _ := fs.GetString(flag)
	if tmp == "" {
		SetJson(obj, jsonPath, "null")
	} else {
		SetJson(obj, jsonPath, `"`+tmp+`"`)
	}
	return true
}

func getAttribute(res map[string]json.RawMessage, attr string, props map[string]string) string {
	log.VPrintf(4, ">Enter: getAttribute(%v, %s, %v", res, attr, props)
	defer log.VPrintf(4, ">Exit: getAttribute")

	js, ok := res[attr]
	if !ok {
		log.VPrintf(4, "-> ''")
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
	log.VPrintf(4, "<-> %s", value)
	return value
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
	getToken()
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

	log.VPrintf(2, ">%s %s", verb, URL)
	defer log.VPrintf(2, "<")
	if len(data) > 0 {
		log.VPrintf(2, "Data:\n%s", string(data))
	} else {
		log.VPrintf(2, "Data: <empty>")
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
	log.VPrintf(2, "Res: %s", res.Status)
	for k, v := range res.Header {
		if len(v) == 1 {
			log.VPrintf(3, "%s: %v", k, v[0])
		} else {
			log.VPrintf(3, "%s: %v", k, v)
		}
	}

	tmp := map[string]json.RawMessage{}
	json.Unmarshal(body, &tmp)
	str, _ := json.MarshalIndent(tmp, "", "  ")
	log.VPrintf(3, "\n%s", string(str))

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
			fmt.Println("%s\n%s", err, string(body))
			// Can't pretty print, so just dump it
			msg = fmt.Sprintf("Error: %s\n%s", res.Status, string(str))
		}

		httpResponse.ErrorMessage = msg
	}

	return httpResponse
}

func main() {
	RootCmd = setupRootCmds()
	initAca()
	initRedis()

	if err := RootCmd.Execute(); err != nil {
		ErrStop(err.Error())
	}
}
