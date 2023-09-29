package main

import (
	"encoding/json"
	"fmt"
	"strings"

	log "github.com/duglin/dlog"
	"github.com/spf13/cobra"
)

func initAca() {
	log.VPrintf(3, "Init initAca")
	setupAcaCmds()
	setupAcaResourceDefs()
	RegisteredParsers = append(RegisteredParsers, AcaFromARMJson)
}

func setupAcaCmds() {
	cmd := &cobra.Command{
		Use:   "aca-env",
		Short: "Add an Azure Container App Environment",
		Run:   ResourceAddFunc,
	}
	cmd.Flags().StringP("name", "n", "", "Name of environment")
	cmd.MarkFlagRequired("name")
	AddCmd.AddCommand(cmd)

	cmd = &cobra.Command{
		Use:   "aca-env",
		Short: "Update an Azure Container App Environment",
		// Run:   ResourceUpdateFunc,
	}
	cmd.Flags().StringP("name", "n", "", "Name of environment")
	cmd.MarkFlagRequired("name")
	UpdateCmd.AddCommand(cmd)

	// ---

	cmd = &cobra.Command{
		Use:   "aca-app",
		Short: "Add an Azure Container App Application",
		Run:   AddAcaAppFunc,
	}
	cmd.Flags().StringP("name", "n", "", "Name of app")
	cmd.Flags().StringP("image", "i", "", "Name of container image")
	cmd.Flags().String("environment", "", "Name of ACA environment")
	cmd.Flags().StringArrayP("env", "e", nil, "Name/value of env var")
	cmd.Flags().String("ingress", "", "'internal', or 'external'")
	cmd.Flags().String("port", "", "listen port #")
	cmd.Flags().StringArray("bind", nil, "Services to connect to")
	cmd.Flags().StringArray("unbind", nil, "Bindings/services to disconnect")
	cmd.Flags().Bool("provision", false, "Provision after update")
	cmd.MarkFlagRequired("name")
	AddCmd.AddCommand(cmd)

	cmd = &cobra.Command{
		Use:   "aca-app",
		Short: "Update an Azure Container App Application",
		Run:   UpdateAcaAppFunc,
	}
	cmd.Flags().StringP("name", "n", "", "Name of app")
	cmd.Flags().StringP("image", "i", "", "Name of container image")
	cmd.Flags().String("environment", "", "Name of ACA environment")
	cmd.Flags().StringArrayP("env", "e", nil, "Name/value of env var")
	cmd.Flags().String("ingress", "", "'internal', or 'external'")
	cmd.Flags().String("port", "", "listen port #")
	cmd.Flags().StringArray("bind", nil, "Services to connect to")
	cmd.Flags().StringArray("unbind", nil, "Bindings/services to disconnect")
	cmd.Flags().Bool("provision", false, "Provision after update")
	cmd.MarkFlagRequired("name")
	UpdateCmd.AddCommand(cmd)

	cmd = &cobra.Command{
		Use:   "aca-app",
		Short: "Show details about an Azure Container App Application",
		Run:   ShowAcaAppFunc,
	}
	cmd.Flags().StringP("name", "n", "", "Name of app")
	cmd.MarkFlagRequired("name")
	ShowCmd.AddCommand(cmd)

	// ---

	cmd = &cobra.Command{
		Use:   "aca-redis",
		Short: "Add an Azure Container App Redis Service",
		Run:   AddAcaServiceFunc,
	}
	cmd.Flags().StringP("name", "n", "", "Name of service")
	cmd.Flags().String("environment", "", "Name of ACA environment")
	cmd.Flags().Bool("provision", false, "Provision after update")
	cmd.MarkFlagRequired("name")
	AddCmd.AddCommand(cmd)

	cmd = &cobra.Command{
		Use:   "aca-redis",
		Short: "Show details about an Azure Container App Redis Service",
		Run:   ShowAcaServiceFunc,
	}
	cmd.Flags().StringP("name", "n", "", "Name of service")
	cmd.MarkFlagRequired("name")
	ShowCmd.AddCommand(cmd)

}

func setupAcaResourceDefs() {
	ResourceDefs["Microsoft.App/managedEnvironments"] = &ResourceDef{
		Type: "Microsoft.App/managedEnvironments",
		URL:  "https://management.azure.com/subscriptions/${SUBSCRIPTION}/resourceGroups/${RESOURCEGROUP}/providers/Microsoft.App/managedEnvironments/${NAME}?api-version=${APIVERSION}",
		Defaults: map[string]string{
			"APIVERSION": "2022-10-01",
			"WAIT":       "true",
		},
	}

	ResourceDefs["Microsoft.App/containerApps"] = &ResourceDef{
		Type: "Microsoft.App/containerApps",
		URL:  "https://management.azure.com/subscriptions/${SUBSCRIPTION}/resourceGroups/${RESOURCEGROUP}/providers/Microsoft.App/containerApps/${NAME}?api-version=${APIVERSION}",
		Defaults: map[string]string{
			"APIVERSION": "2022-11-01-preview",
			"WAIT":       "true",
		},
	}
}

type AcaAppConfiguration struct {
	Ingress *struct {
		External      *bool `json:"external,omitempty"`
		TargetPort    *int  `json:"targetPort,omitempty"`
		CustomDomains []*struct {
			Name          *string `json:"name,omitempty"`
			BindingType   *string `json:bindingType,omitempty"`
			CertificateId *string `json:certificateId,omitempty"`
		} `json:"customDomains,omitempty"`
		Traffic []*struct {
		} `json:"traffic,omitempty"`
		// ipSecurityRestrictions
		// stickySessions
		// clientCertificateMode
		// corePolicy
	} `json:"ingress,omitempty"`
	// dapr
	// maxInactiveRevisions
	Service *struct {
		Type *string `json:"type,omitempty"`
	} `json:"service,omitempty"`
	// service
}

type AcaAppEnv struct {
	Name  *string `json:"name,omitempty"`
	Value *string `json:"value,omitempty"`
}

type AcaAppContainer struct {
	Image     *string      `json:"image,omitempty"`
	Name      *string      `json:"name,omitempty"`
	Env       []*AcaAppEnv `json:"env,omitempty"`
	Resources *struct {
		CPU    *float64 `json:"cpu,omitempty"`
		Memory *string  `json:"memory,omitempty"`
	} `json:"resources,omitempty"`
	Command []string `json:"command,omitempty"`
	Args    []string `json:"args,omitempty"`
	// probes
}

type AcaAppScale struct {
	MinReplicas *int `json:"minReplicas,omitempty"`
	MaxReplicas *int `json:"maxReplicas,omitempty"`
	// rules
}

type AcaAppServiceBind struct {
	ServiceId *string `json:"serviceId,omitempty"`
	Name      *string `json:"name,omitempty"`
}

type AcaAppTemplate struct {
	Containers []*AcaAppContainer `json:"containers,omitempty"`
	// initContainers
	Scale        *AcaAppScale         `json:"scale,omitempty"`
	ServiceBinds []*AcaAppServiceBind `json:"serviceBinds,omitempty"`
}

type AcaAppProperties struct {
	EnvironmentId       *string              `json:"environmentId,omitempty"`
	WorkloadProfileName *string              `json:"workloadProfileName,omitempty"`
	Configuration       *AcaAppConfiguration `json:"configuration,omitempty"`
	Template            *AcaAppTemplate      `json:"template,omitempty"`
}

func (aa *AcaApp) MarshalJSON() ([]byte, error) {
	tmpAa := *aa
	if WhyMarshal == "ARM" {
		if tmpAa.Location == nil {
			tmpAa.Location = StringPtr(Config["defaults.Location"])
		}
		if tmpAa.Location == nil || *(tmpAa.Location) == "" {
			ErrStop(`Missing "location" for "%s/%s"`, aa.NiceType, aa.Name)
		}
	}
	return json.Marshal(tmpAa)
}

func (aap *AcaAppProperties) MarshalJSON() ([]byte, error) {
	tmpAap := *aap
	if WhyMarshal == "ARM" {
		envRef := aap.ResolveEnvironmentId()
		tmpAap.EnvironmentId = StringPtr(envRef.AsID())
	}
	return json.Marshal(tmpAap)
}

func (aac *AcaAppContainer) MarshalJSON() ([]byte, error) {
	tmpAac := *aac
	if WhyMarshal == "ARM" {
		if tmpAac.Name == nil {
			tmpAac.Name = StringPtr("main")
		}
	}
	return json.Marshal(tmpAac)
}

func (asb *AcaAppServiceBind) xMarshalJSON() ([]byte, error) {
	tmpAsb := *asb
	if WhyMarshal == "ARM" {
		// tmpAsb.ServiceId = StringPtr(*(tmpAsb.ServiceId))
		if tmpAsb.Name == nil {
			tmpAsb.Name = StringPtr(*(tmpAsb.ServiceId))
		}
	}
	return json.Marshal(tmpAsb)
}

func (aap *AcaAppProperties) ResolveEnvironmentId() *ResourceReference {
	ref := aap.EnvironmentId
	if ref == nil || *ref == "" {
		ErrStop("AcaApp is missing an \"environmentId\" value")
	}

	// Set defaults
	resRef := &ResourceReference{
		Subscription:  Config["defaults.Subscription"],
		ResourceGroup: Config["defaults.ResourceGroup"],
		Type:          "Microsoft.App/managedEnvironments",
		APIVersion:    ResourceDefs["Microsoft.App/managedEnvironments"].Defaults["APIVERSION"],
		Origin:        *(aap.EnvironmentId),
	}

	// Now, override with env values
	resRef.Populate(*ref)

	return resRef
}

type AcaApp struct {
	ResourceBase

	Location   *string           `json:"location,omitempty"`
	Properties *AcaAppProperties `json:"properties,omitempty"`
}

func (app *AcaApp) DependsOn() []*ResourceReference {
	refs := []*ResourceReference{}

	if app.Properties != nil {
		resRef := app.Properties.ResolveEnvironmentId()
		refs = append(refs, resRef)
	}

	return refs
}

func (app *AcaApp) ToARMJson() string {
	saveURL := app.URL
	app.URL = ""

	if app.Properties != nil && app.Properties.Configuration != nil &&
		app.Properties.Configuration.Ingress != nil {
		ingress := app.Properties.Configuration.Ingress
		if ingress.External != nil && *ingress.External == true &&
			ingress.TargetPort == nil {
			port := 8080
			ingress.TargetPort = &port
		}
	}

	WhyMarshal = "ARM"
	data, _ := json.MarshalIndent(app, "", "  ")
	WhyMarshal = ""

	app.URL = saveURL
	return string(data)
}

func AcaFromARMJson(data []byte) *ResourceBase {
	tmp := struct{ URL string }{}
	json.Unmarshal(data, &tmp)

	resRef := ParseResourceURL(tmp.URL)

	if resRef.Type == "Microsoft.App/containerApps" {
		app := &AcaApp{}
		json.Unmarshal(data, &app)

		// ResourceBase stuff
		app.Subscription = resRef.Subscription
		app.ResourceGroup = resRef.ResourceGroup
		app.Type = resRef.Type
		app.Name = resRef.Name
		app.APIVersion = resRef.APIVersion

		if app.Properties != nil &&
			app.Properties.Configuration != nil &&
			app.Properties.Configuration.Service != nil &&
			app.Properties.Configuration.Service.Type != nil {
			app.NiceType = "aca-" + *app.Properties.Configuration.Service.Type
		} else {
			app.NiceType = "aca-app"
		}

		app.URL = tmp.URL
		app.Object = app
		app.RawData = data

		return &app.ResourceBase
	} else if resRef.Type == "Microsoft.App/managedEnvironments" {
	} else {
		return nil
	}

	return nil
}

func AddAcaServiceFunc(cmd *cobra.Command, args []string) {
	LoadConfig()

	_, service, _ := strings.Cut(cmd.CalledAs(), "-")
	if service == "" {
		ErrStop("Unknown resource type: %s", cmd.CalledAs())
	}
	if service != "redis" {
		ErrStop("Unsupported service type: %s", service)
	}

	app := &AcaApp{}
	app.Object = app

	// ResourceBase stuff
	app.Subscription = Config["defaults.Subscription"]
	app.ResourceGroup = Config["defaults.ResourceGroup"]
	app.Type = "Microsoft.App/containerApps"
	app.Name, _ = cmd.Flags().GetString("name")
	app.APIVersion = ResourceDefs[app.Type].Defaults["APIVERSION"]
	app.NiceType = cmd.CalledAs()

	app.Stage = Config["currentStage"]
	app.Filename = fmt.Sprintf("%s-%s.json", app.NiceType, app.Name)

	// App specific stuff
	// app.Location = StringPtr(Config["defaults.Location"])

	set := SetStringProp(app, cmd.Flags(), "environment",
		`{"properties":{"environmentId":%s}}`)
	if !set && app.Properties.EnvironmentId == nil {
		env := Config["defaults.aca-env"]
		if env == "" {
			ErrStop("Missing the aca-env value. "+
				"Use either '--environment=???' "+
				"or '%s set defaults.aca-env=???'", APP)
		}
		SetJson(app, `{"properties":{"environmentId":%q}}`, env)
	}

	// Now set the app to be a dev mode service
	SetJson(app, `{"properties":{"configuration":{"service":{"type":%q}}}}`,
		service)

	app.Save()
	p, _ := cmd.Flags().GetBool("provision")
	if p || Config["defaults.provision"] == "true" {
		app.Provision()
	}
}

func ShowAcaServiceFunc(cmd *cobra.Command, args []string) {
	log.VPrintf(2, ">Enter: ShowAcaServiceFunc (%q)", args)
	defer log.VPrintf(2, "<Exit: ShowAcaServiceFunc")

	LoadConfig()

	stage := Config["currentStage"]
	name, _ := cmd.Flags().GetString("name")
	name = fmt.Sprintf("%s-%s.json", cmd.CalledAs(), name)
	res, err := ResourceFromFile(stage, name)
	NoErr(err, "Resource %s/%s not found", cmd.CalledAs(), name)

	app := res.Object.(*AcaApp)

	fmt.Printf("Name: %s\n", app.Name)
	fmt.Printf("Service: %s\n", *(app.Properties.Configuration.Service.Type))
}

func AddAcaAppFunc(cmd *cobra.Command, args []string) {
	log.VPrintf(2, ">Enter: AddAcaAppFunc (%q)", args)
	defer log.VPrintf(2, "<Exit: AddAcaAppFunc")

	LoadConfig()

	app := &AcaApp{}
	app.Object = app

	// ResourceBase stuff
	app.Subscription = Config["defaults.Subscription"]
	app.ResourceGroup = Config["defaults.ResourceGroup"]
	app.Type = "Microsoft.App/containerApps"
	app.Name, _ = cmd.Flags().GetString("name")
	app.APIVersion = ResourceDefs[app.Type].Defaults["APIVERSION"]
	app.NiceType = "aca-app"

	app.Stage = Config["currentStage"]
	app.Filename = fmt.Sprintf("%s-%s.json", app.NiceType, app.Name)

	// App specific stuff
	// app.Location = StringPtr(Config["defaults.Location"])

	app.ProcessFlags(cmd)
	app.Save()

	p, _ := cmd.Flags().GetBool("provision")
	if p || Config["defaults.provision"] == "true" {
		app.Provision()
	}
}

func UpdateAcaAppFunc(cmd *cobra.Command, args []string) {
	log.VPrintf(2, ">Enter: UpdateAcaAppFunc (%q)", args)
	defer log.VPrintf(2, "<Exit: UpdateAcaAppFunc")

	LoadConfig()

	stage := Config["currentStage"]
	name, _ := cmd.Flags().GetString("name")
	name = fmt.Sprintf("%s-%s.json", "aca-app", name)
	res, err := ResourceFromFile(stage, name)
	NoErr(err, "Resource %s/%s not found", cmd.CalledAs(), name)

	app := res.Object.(*AcaApp)

	app.ProcessFlags(cmd)
	app.Save()

	p, _ := cmd.Flags().GetBool("provision")
	if p || Config["defaults.provision"] == "true" {
		app.Provision()
	}
}

func ShowAcaAppFunc(cmd *cobra.Command, args []string) {
	log.VPrintf(2, ">Enter: UpdateAcaAppFunc (%q)", args)
	defer log.VPrintf(2, "<Exit: UpdateAcaAppFunc")

	LoadConfig()

	stage := Config["currentStage"]
	name, _ := cmd.Flags().GetString("name")
	name = fmt.Sprintf("%s-%s.json", "aca-app", name)
	res, err := ResourceFromFile(stage, name)
	NoErr(err, "Resource %s/%s not found", cmd.CalledAs(), name)

	app := res.Object.(*AcaApp)

	fmt.Printf("Name         : %s\n", app.Name)
	fmt.Printf("Environment  : %s\n", NoNil(app.Properties.EnvironmentId))
	if app.Location != nil {
		fmt.Printf("Location     : %s\n", *(app.Location))
	}
	// fmt.Printf("\n")
	fmt.Printf("Subscription : %s\n", app.Subscription)
	fmt.Printf("ResourceGroup: %s\n", app.ResourceGroup)

	ingress := "internal"
	port := ""

	if app.Properties != nil && app.Properties.Configuration != nil &&
		app.Properties.Configuration.Ingress != nil {
		ing := app.Properties.Configuration.Ingress
		if ing.External != nil && *(ing.External) == true {
			ingress = "external"
		}
		if ing.TargetPort != nil {
			port = fmt.Sprintf("%s", ing.TargetPort)
		}
	}

	fmt.Printf("\n")
	fmt.Printf("Ingress: %s\n", ingress)
	if port != "" {
		fmt.Printf("Port   : %s\n", port)
	}

	template := app.Properties.Template
	if template != nil {
		cont := template.Containers
		if cont == nil || len(cont) == 0 {
			fmt.Printf("\n")
			fmt.Printf("Container: none\n")
		} else {
			for i, c := range cont {
				fmt.Printf("\n")
				if len(cont) > 1 {
					fmt.Printf("Container(%d):\n", i+1)
				} else {
					fmt.Printf("Container:\n")
				}
				fmt.Printf("  Image: %s\n", NoNil(c.Image))
			}
		}

		// scale := template.Scale

		binds := template.ServiceBinds
		if len(binds) > 0 {
			fmt.Printf("\n")
			fmt.Printf("Bindings:\n")
			for _, bind := range binds {
				fmt.Printf("  - Service: %s\n", NoNil(bind.ServiceId))
			}
		}
	}
}

func NoNil(str *string) string {
	if str == nil {
		return ""
	}
	return *str
}

func (app *AcaApp) ProcessFlags(cmd *cobra.Command) {
	log.VPrintf(2, ">Enter: ProcessFlags")
	defer log.VPrintf(2, "<Exit: ProcessFlags")

	SetStringProp(app, cmd.Flags(), "image",
		`{"properties":{"template":{"containers":[{"image":%s}]}}}`)

	// TODO make sure 'environ' exists
	set := SetStringProp(app, cmd.Flags(), "environment",
		`{"properties":{"environmentId":%s}}`)
	if !set && app.Properties.EnvironmentId == nil {
		env := Config["defaults.aca-env"]
		if env == "" {
			ErrStop("Missing the aca-env value. "+
				"Use either '--environment=???' "+
				"or '%s set defaults.aca-env=???'", APP)
		}
		SetJson(app, `{"properties":{"environmentId":%q}}`, env)
	}

	envs, _ := cmd.Flags().GetStringArray("env")
	for i, env := range envs {
		if i == 0 {
			// Just make sure the container is there first
			SetJson(app,
				`{"properties":{"template":{"containers":[{"name":"main"}]}}}`)
			app.Properties.Template.Containers[0].Name = nil
		}
		name, val, found := strings.Cut(env, "=")
		c := app.Properties.Template.Containers[0]

		pos := -1
		for j, tmpE := range c.Env {
			if tmpE.Name != nil && *(tmpE.Name) == name {
				pos = j
				break
			}
		}

		if !found {
			// remove env var
			if pos >= 0 {
				c.Env = append(c.Env[:pos], c.Env[pos+1:]...)
			}
		} else {
			if pos >= 0 {
				// Update
				c.Env[pos].Value = StringPtr(val)
			} else {
				// Add
				c.Env = append(c.Env, &AcaAppEnv{
					Name:  StringPtr(name),
					Value: StringPtr(val)})
			}
		}

	}

	if cmd.Flags().Changed("ingress") {
		tmp, _ := cmd.Flags().GetString("ingress")
		val := (tmp == "external")
		SetJson(app,
			`{"properties":{"configuration":{"ingress":{"external":%v}}}}`, val)
	}

	if cmd.Flags().Changed("port") {
		port, _ := cmd.Flags().GetInt("port")
		SetJson(app,
			`{"properties":{"configuration":{"ingress":{"targetPort":%d}}}}`, port)
	}

	if cmd.Flags().Changed("bind") {
		bindServices, _ := cmd.Flags().GetStringArray("bind")

		templ := app.Properties.Template

		for _, bindName := range bindServices {
			found := false
			for i, sb := range templ.ServiceBinds {
				if sb.ServiceId != nil && *sb.ServiceId == bindName {
					templ.ServiceBinds = append(templ.ServiceBinds[:i],
						templ.ServiceBinds[i+1:]...)
					found = true
					break
				}
			}
			if found {
				ErrStop("Binding %q already exists", bindName)
			}

			newBind := &AcaAppServiceBind{
				ServiceId: StringPtr(bindName),
				// Name:      bindName,
			}
			templ.ServiceBinds = append(templ.ServiceBinds, newBind)
		}
	}

	if cmd.Flags().Changed("unbind") {
		bindServices, _ := cmd.Flags().GetStringArray("unbind")

		templ := app.Properties.Template

		for _, bindName := range bindServices {
			found := false
			for i, sb := range templ.ServiceBinds {
				// TODO check for the same service connected more than
				// once but w/o them giving us a bindingName
				if sb.ServiceId != nil && *sb.ServiceId == bindName {
					templ.ServiceBinds = append(templ.ServiceBinds[:i],
						templ.ServiceBinds[i+1:]...)
					found = true
					break
				}
			}
			if !found {
				ErrStop("Binding %q was not found", bindName)
			}
		}
	}
}

/*
func (app *AcaApp) Save() {
	log.VPrintf(0, "In aca-app save")
	app.ResourceBase.Save()
}
*/
