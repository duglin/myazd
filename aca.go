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
	cmd.Flags().StringP("output", "o", "pretty", "Format (pretty,raw,rest,azure)")
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
	cmd.Flags().StringP("output", "o", "pretty", "Format (pretty,raw,rest,azure)")
	cmd.MarkFlagRequired("name")
	ShowCmd.AddCommand(cmd)

}

func setupAcaResourceDefs() {
	AddResourceDef(&ResourceDef{
		Type: "Microsoft.App/managedEnvironments",
		URL:  "https://management.azure.com/subscriptions/${SUBSCRIPTION}/resourceGroups/${RESOURCEGROUP}/providers/Microsoft.App/managedEnvironments/${NAME}?api-version=${APIVERSION}",
		Defaults: map[string]string{
			"APIVERSION": "2022-10-01",
			"WAIT":       "true",
		},
	})

	AddResourceDef(&ResourceDef{
		Type: "Microsoft.App/containerApps",
		URL:  "https://management.azure.com/subscriptions/${SUBSCRIPTION}/resourceGroups/${RESOURCEGROUP}/providers/Microsoft.App/containerApps/${NAME}?api-version=${APIVERSION}",
		Defaults: map[string]string{
			"APIVERSION": "2023-05-02-preview",
			"WAIT":       "true",
		},
	})
}

type AcaAppIngress struct {
	External      *bool `json:"external,omitempty"`
	TargetPort    *int  `json:"targetPort,omitempty"`
	CustomDomains []*struct {
		Name          *string `json:"name,omitempty"`
		BindingType   *string `json:bindingType,omitempty"`
		CertificateId *string `json:certificateId,omitempty"`
	} `json:"customDomains,omitempty"`
	Traffic []*AcaAppTraffic `json:"traffic,omitempty"`
	// ipSecurityRestrictions
	// stickySessions
	// clientCertificateMode
	// corePolicy
}

type AcaAppTraffic struct {
}

type AcaAppConfiguration struct {
	Ingress *AcaAppIngress `json:"ingress,omitempty"`
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
	Image     *string          `json:"image,omitempty"`
	Name      *string          `json:"name,omitempty"`
	Env       []*AcaAppEnv     `json:"env,omitempty"`
	Resources *AcaAppResources `json:"resources,omitempty"`
	Command   []string         `json:"command,omitempty"`
	Args      []string         `json:"args,omitempty"`
	// probes
}

type AcaAppResources struct {
	CPU    *float64 `json:"cpu,omitempty"`
	Memory *string  `json:"memory,omitempty"`
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
		tmpAap.WorkloadProfileName = StringPtr("Consumption")

		// Temporary to get around an ACA NPE
		if tmpAap.Template == nil {
			tmpAap.Template = &AcaAppTemplate{
				Containers: []*AcaAppContainer{{
					Image: StringPtr("redis"),
					Name:  StringPtr("redis"),
				}},
			}
		}
		// END OF Temporary
	}
	return json.Marshal(tmpAap)
}

func (aac *AcaAppContainer) MarshalJSON() ([]byte, error) {
	tmpAac := *aac
	if WhyMarshal == "ARM" {
		if tmpAac.Name == nil {
			tmpAac.Name = StringPtr("main")
		}
		if tmpAac.Resources == nil {
			tmpAac.Resources = &AcaAppResources{}
		}
		if tmpAac.Resources.CPU == nil {
			f := 0.5
			tmpAac.Resources.CPU = &f
		}
		if tmpAac.Resources.Memory == nil {
			m := "1Gi"
			tmpAac.Resources.Memory = &m
		}
	}
	return json.Marshal(tmpAac)
}

func (aai *AcaAppIngress) MarshalJSON() ([]byte, error) {
	tmpAai := *aai
	if WhyMarshal == "ARM" {
		if tmpAai.External != nil && *(tmpAai.External) == true &&
			tmpAai.TargetPort == nil {
			port := 8080
			tmpAai.TargetPort = &port

			/*
				if tmpAai.Traffic == nil {
					tmpAai.Traffic = []*AcaAppTraffic{{}}
				}
			*/
		}
	}
	return json.Marshal(tmpAai)
}

func (asb *AcaAppServiceBind) MarshalJSON() ([]byte, error) {
	tmpAsb := *asb
	if WhyMarshal == "ARM" {
		// tmpAsb.ServiceId = StringPtr(*(tmpAsb.ServiceId))
		if tmpAsb.Name == nil {
			tmpAsb.Name = StringPtr(*(tmpAsb.ServiceId))
		}
	}
	return json.Marshal(tmpAsb)
}

func (asb *AcaAppServiceBind) ResolveServiceId() *ResourceReference {
	ref := asb.ServiceId
	if ref == nil || *ref == "" {
		ErrStop("AcaApp is missing a ServiceId in a binding")
	}

	// Set defaults
	resRef := &ResourceReference{
		Subscription:  Config["defaults.Subscription"],
		ResourceGroup: Config["defaults.ResourceGroup"],
		Type:          "Microsoft.App/containerapps", // Can't assume
		APIVersion:    GetResourceDef("Microsoft.App/containerapps").Defaults["APIVERSION"],
		Origin:        *(asb.ServiceId),
	}

	// Now, override with env values
	resRef.Populate(*ref)

	return resRef
}
func (aat *AcaAppTemplate) MarshalJSON() ([]byte, error) {
	tmpAat := *aat
	if WhyMarshal == "ARM" {
		if tmpAat.Scale == nil {
			tmpAat.Scale = &AcaAppScale{}
		}
		/*
			if tmpAat.Scale.MinReplicas == nil {
				m := 0
				tmpAat.Scale.MinReplicas = &m
			}
		*/
		if tmpAat.Scale.MaxReplicas == nil {
			m := 10
			tmpAat.Scale.MaxReplicas = &m
		}
	}
	return json.Marshal(tmpAat)
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
		APIVersion:    GetResourceDef("Microsoft.App/managedEnvironments").Defaults["APIVERSION"],
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

	if props := app.Properties; props != nil {
		resRef := props.ResolveEnvironmentId()
		refs = append(refs, resRef)

		if template := props.Template; template != nil {
			if sbs := template.ServiceBinds; sbs != nil {
				for _, sb := range sbs {
					if sb.ServiceId != nil {
						resRef := sb.ResolveServiceId()
						refs = append(refs, resRef)
					}
				}
			}
		}
	}

	return refs
}

func (app *AcaApp) ToARMJson() string {
	WhyMarshal = "ARM"
	data, _ := json.MarshalIndent(app, "", "  ")
	WhyMarshal = ""

	return string(data)
}

func (app *AcaApp) HideServerFields(diffA ARMResource) {
	// diffApp := diffA.(*AcaApp)

	if app.Properties != nil && app.Properties.Configuration != nil {
		c := app.Properties.Configuration
		if (*c == AcaAppConfiguration{}) {
			app.Properties.Configuration = nil
		}
	}
}

func AcaFromARMJson(data []byte) *ResourceBase {
	tmp := struct{ ID string }{}
	json.Unmarshal(data, &tmp)

	resRef := ParseResourceID(tmp.ID)

	if strings.EqualFold(resRef.Type, "Microsoft.App/containerApps") {
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

		app.ID = tmp.ID
		app.Object = app
		app.RawData = data

		return &app.ResourceBase
	} else if strings.EqualFold(resRef.Type, "Microsoft.App/managedEnvironments") {
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
	app.APIVersion = GetResourceDef(app.Type).Defaults["APIVERSION"]
	app.NiceType = cmd.CalledAs()

	app.Stage = Config["currentStage"]
	app.Filename = fmt.Sprintf("%s-%s.json", app.NiceType, app.Name)

	// App specific stuff
	// app.Location = StringPtr(Config["defaults.Location"])

	set := SetStringProp(app, cmd.Flags(), "environment",
		`{"properties":{"environmentId":%s}}`)
	if !set && app.Properties == nil || app.Properties.EnvironmentId == nil {
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

	output, _ := cmd.Flags().GetString("output")

	if output == "raw" {
		fmt.Printf("%s\n", string(res.RawData))
		return
	}

	if output == "rest" {
		fmt.Printf("%s\n", res.Object.ToARMJson())
		return
	}

	if output == "azure" {
		data, err := res.Download()
		if err != nil {
			ErrStop("Error downloading: %s", err)
		}
		if len(data) == 0 {
			ErrStop("Resource doesn't existin in Azure - "+
				"try '%s up' to create it", APP)
		}

		tmp := map[string]json.RawMessage{}
		json.Unmarshal(data, &tmp)
		str, _ := json.MarshalIndent(tmp, "", "  ")

		fmt.Printf("%s\n", string(str))
		return
	}

	if output != "pretty" {
		ErrStop("Unknown output format %q", output)
	}

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
	app.APIVersion = GetResourceDef(app.Type).Defaults["APIVERSION"]
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
	log.VPrintf(2, ">Enter: ShowAcaAppFunc (%q)", args)
	defer log.VPrintf(2, "<Exit: ShowAcaAppFunc")

	LoadConfig()

	stage := Config["currentStage"]
	name, _ := cmd.Flags().GetString("name")
	name = fmt.Sprintf("%s-%s.json", "aca-app", name)
	res, err := ResourceFromFile(stage, name)
	NoErr(err, "Resource %s/%s not found", cmd.CalledAs(), name)

	app := res.Object.(*AcaApp)

	output, _ := cmd.Flags().GetString("output")

	if output == "raw" {
		fmt.Printf("%s\n", string(res.RawData))
		return
	}

	if output == "rest" {
		fmt.Printf("%s\n", res.Object.ToARMJson())
		return
	}

	if output == "azure" {
		data, err := res.Download()
		if err != nil {
			ErrStop("Error downloading: %s", err)
		}
		if len(data) == 0 {
			ErrStop("Resource doesn't existin in Azure - "+
				"try '%s up' to create it", APP)
		}

		tmp := map[string]json.RawMessage{}
		json.Unmarshal(data, &tmp)
		str, _ := json.MarshalIndent(tmp, "", "  ")

		fmt.Printf("%s\n", string(str))
		return
	}

	if output != "pretty" {
		ErrStop("Unknown output format %q", output)
	}

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
	fmt.Printf("\n")
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
	if !set && app.Properties != nil && app.Properties.EnvironmentId == nil {
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
