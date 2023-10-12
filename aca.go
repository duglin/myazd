package main

import (
	"encoding/json"
	"fmt"
	"strconv"
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
	/*
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
	*/

	// ---

	cmd := &cobra.Command{
		Use:   "aca-app",
		Short: "Add an Azure Container App Application",
		Run:   AddAcaAppFunc,
	}
	cmd.Flags().StringP("name", "n", "", "Name of app")
	cmd.Flags().StringP("image", "i", "", "Name of container image")
	cmd.Flags().String("environment", "", "Name of ACA environment")
	cmd.Flags().StringP("subscription", "s", "", "Subscription ID")
	cmd.Flags().StringP("resource-group", "g", "", "Resource Group")
	cmd.Flags().StringP("location", "l", "", "Location")
	cmd.Flags().StringArrayP("env", "e", nil, "Name/value of env var")
	cmd.Flags().String("ingress", "", "'internal', or 'external'")
	cmd.Flags().Bool("external", false, "Enable public access")
	cmd.Flags().Bool("internal", true, "Disable public access")
	cmd.Flags().String("port", "", "listen port #")
	cmd.Flags().StringArray("bind", nil, "Services to connect to")
	cmd.Flags().StringArray("unbind", nil, "Bindings/services to disconnect")
	cmd.Flags().Bool("up", false, "Provision after update")
	cmd.MarkFlagsMutuallyExclusive("internal", "external", "ingress")
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
	cmd.Flags().StringP("subscription", "s", "", "Subscription ID")
	cmd.Flags().StringP("resource-group", "g", "", "Resource Group")
	cmd.Flags().StringP("location", "l", "", "Location")
	cmd.Flags().StringArrayP("env", "e", nil, "Name/value of env var")
	cmd.Flags().String("ingress", "", "'internal', or 'external'")
	cmd.Flags().Bool("external", false, "Enable public access")
	cmd.Flags().Bool("internal", true, "Disable public access")
	cmd.Flags().String("port", "", "listen port #")
	cmd.Flags().StringArray("bind", nil, "Services to connect to")
	cmd.Flags().StringArray("unbind", nil, "Bindings/services to disconnect")
	cmd.Flags().Bool("up", false, "Provision after update")
	cmd.MarkFlagsMutuallyExclusive("internal", "external", "ingress")
	cmd.MarkFlagRequired("name")
	UpdateCmd.AddCommand(cmd)

	cmd = &cobra.Command{
		Use:   "aca-app",
		Short: "Show details about an Azure Container App Application",
		Run:   ShowFunc,
	}
	cmd.Flags().StringP("name", "n", "", "Name of app")
	cmd.Flags().String("from", "iac", "Show data from: iac, rest, azure")
	cmd.Flags().StringP("output", "o", "pretty", "Format (pretty,json)")
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
	cmd.Flags().StringP("subscription", "s", "", "Subscription ID")
	cmd.Flags().StringP("resource-group", "g", "", "Resource Group")
	cmd.Flags().StringP("location", "l", "", "Location")
	cmd.Flags().Bool("up", false, "Provision after update")
	cmd.MarkFlagRequired("name")
	AddCmd.AddCommand(cmd)

	cmd = &cobra.Command{
		Use:   "aca-redis",
		Short: "Show details about an Azure Container App Redis Service",
		Run:   ShowFunc,
	}
	cmd.Flags().StringP("name", "n", "", "Name of service")
	cmd.Flags().String("from", "iac", "Show data from: iac, rest, azure")
	cmd.Flags().StringP("output", "o", "pretty", "Format (pretty,json)")
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
	Service *AcaAppService `json:"service,omitempty"`
}

type AcaAppService struct {
	Type *string `json:"type,omitempty"`
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
			tmpAa.Location = StringPtr(GetConfigProperty("defaults.Location"))
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
		if tmpAai.External != nil && tmpAai.External != nil &&
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
		Subscription:  GetConfigProperty("defaults.Subscription"),
		ResourceGroup: GetConfigProperty("defaults.ResourceGroup"),
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
		Subscription:  GetConfigProperty("defaults.Subscription"),
		ResourceGroup: GetConfigProperty("defaults.ResourceGroup"),
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

func (app *AcaApp) ToForm() *Form {
	res := &app.ResourceBase

	if res.NiceType == "aca-redis" {
		form := NewForm()
		form.AddProp("Service", NotNil(app.Properties.Configuration.Service.Type))
		return form
	}

	// Must be a normal app
	ingress := "internal"
	port := ""

	if app.Properties != nil && app.Properties.Configuration != nil &&
		app.Properties.Configuration.Ingress != nil {
		ing := app.Properties.Configuration.Ingress
		if ing.External != nil && *(ing.External) == true {
			ingress = "external"
		}
		if ing.TargetPort != nil {
			port = fmt.Sprintf("%d", *(ing.TargetPort))
		}
	}

	form := NewForm()
	form.AddProp("Name", app.Name)
	form.AddProp("Environment", NotNil(app.Properties.EnvironmentId))
	if app.Location != nil {
		form.AddProp("Location", NotNil(app.Location))
	}
	form.AddProp("Subscription", app.Subscription)
	form.AddProp("ResourceGroup", app.ResourceGroup)
	if app.Properties.WorkloadProfileName != nil { // to avoid name alignment
		wpf := form.AddSection("", "")
		wpf.Space = false
		wpf.AddProp("Workload Profile Name", *(app.Properties.WorkloadProfileName))
	}

	nf := form.AddSection("Ingress", ingress) // .Space = true
	if port != "" {
		nf.AddProp("Port", port)
	}

	template := app.Properties.Template
	if template != nil {
		// cont := template.Containers
		// if cont == nil || len(cont) == 0 {
		// form.AddArray("Containers", "none").Space = true
		// } else {
		nf := form.AddArray("Containers", "")
		// nf.Space = true
		for i, c := range template.Containers {
			cf := nf.AddSection(fmt.Sprintf("*#%d", i+1), "")
			cf.AddProp("Image", NotNil(c.Image))
			if len(c.Command) > 0 {
				cf.AddProp("Command", QuoteStrings(c.Command))
			}
			if len(c.Args) > 0 {
				cf.AddProp("Args", QuoteStrings(c.Args))
			}
			if c.Resources != nil {
				if c.Resources.CPU != nil {
					cf.AddProp("CPU", fmt.Sprintf("%v", *(c.Resources.CPU)))
				}
				if c.Resources.Memory != nil {
					cf.AddProp("Memory", fmt.Sprintf("%s", *(c.Resources.Memory)))
				}
			}

			if scale := template.Scale; scale != nil {
				if scale.MinReplicas != nil {
					cf.AddProp("Min Scale", fmt.Sprintf("%d", *(scale.MinReplicas)))
				}
				if scale.MaxReplicas != nil {
					cf.AddProp("Max Scale", fmt.Sprintf("%d", *(scale.MaxReplicas)))
				}
			}

			if len(c.Env) > 0 {
				ef := cf.AddArray("Environment variables", "")
				for _, env := range c.Env {
					// es := ef.AddSection("*"+NotNil(env.Name), "")
					// es.Space = false
					ef.AddProp(NotNil(env.Name), NotNil(env.Value))
				}
			}
		}
		// }

		/*
			if scale := template.Scale; scale != nil {
				if scale.MinReplicas != nil || scale.MaxReplicas != nil {
					nf := form.AddSection("Scaling", "")
					if scale.MinReplicas
					nf.AddProp("Min Scale",
				}
			}
		*/

		binds := template.ServiceBinds
		if len(binds) > 0 {
			nf := form.AddArray("Bindings", "")
			// nf.Space = true
			for _, bind := range binds {
				sec := nf.AddSection("*Service:"+NotNil(bind.ServiceId), "")
				sec.AddProp("Service", NotNil(bind.ServiceId))
				if bind.Name != nil {
					sec.AddProp("Name", NotNil(bind.Name))
				}
			}
		}
	}

	return form
}

func (app *AcaApp) MustProperties() *AcaAppProperties {
	if app.Properties == nil {
		app.Properties = &AcaAppProperties{}
	}
	return app.Properties
}

func (app *AcaApp) MustConfiguration() *AcaAppConfiguration {
	if props := app.MustProperties(); props.Configuration == nil {
		props.Configuration = &AcaAppConfiguration{}
	}
	return app.Properties.Configuration
}

func (app *AcaApp) MustIngress() *AcaAppIngress {
	if config := app.MustConfiguration(); config.Ingress == nil {
		config.Ingress = &AcaAppIngress{}
	}
	return app.Properties.Configuration.Ingress
}

func (app *AcaApp) MustTemplate() *AcaAppTemplate {
	if props := app.MustProperties(); props.Template == nil {
		props.Template = &AcaAppTemplate{}
	}
	return app.Properties.Template
}

func (app *AcaApp) MustScale() *AcaAppScale {
	if template := app.MustTemplate(); template.Scale == nil {
		template.Scale = &AcaAppScale{}
	}
	return app.Properties.Template.Scale
}

func (c *AcaAppContainer) MustResources() *AcaAppResources {
	if c.Resources == nil {
		c.Resources = &AcaAppResources{}
	}
	return c.Resources
}

func (app *AcaApp) FromForm(r *ResourceBase, f *Form) {
	var newApp *AcaApp

	if f.Type != "Section" {
		panic("Bad type: " + f.Type)
	}

	if r.NiceType == "aca-redis" {
		newApp = &AcaApp{
			ResourceBase: app.ResourceBase,
		}

		newApp.Properties = &AcaAppProperties{
			Configuration: &AcaAppConfiguration{
				Service: &AcaAppService{
					Type: StringPtr(f.GetProp("Service")),
				},
			},
		}
	} else {
		newApp = &AcaApp{
			ResourceBase: app.ResourceBase,
		}

		items := f.Items // allows for a growing list

		for len(items) > 0 {
			item := items[0]
			items = items[1:]

			if item.Type == "Section" && item.Title == "" {
				items = append(items, item.Items...)
				continue
			}

			switch item.Title {
			case "Name":
				// Skip
			case "Environment":
				newApp.MustProperties().EnvironmentId = StringPtr(item.Value)
			case "Location":
				newApp.Location = StringPtr(item.Value)
			case "Subscription":
				newApp.Subscription = item.Value
			case "ResourceGroup":
				newApp.ResourceGroup = item.Value

			case "Workload Profile Name":
				newApp.MustProperties().WorkloadProfileName =
					StringPtr(item.Value)

			case "Ingress":
				newApp.MustIngress().External = BoolPtr(item.Value == "external")
				if val := item.GetProp("Port"); val != "" {
					p, _ := strconv.Atoi(val)
					newApp.MustIngress().TargetPort = &p
				}

			case "Containers": // "Containers" Array
				for _, cSection := range item.Items { // cSection = Cont Section
					c := &AcaAppContainer{}
					newApp.MustTemplate().Containers =
						append(newApp.MustTemplate().Containers, c)

					for _, item := range cSection.Items {
						switch item.Title {
						case "Image":
							c.Image = StringPtr(item.Value)
						case "Command":
							c.Command = ParseQuotedString(item.Value)
						case "Args":
							c.Args = ParseQuotedString(item.Value)

						case "CPU":
							f, _ := strconv.ParseFloat(item.Value, 64)
							c.MustResources().CPU = &f
						case "Memory":
							c.MustResources().Memory = StringPtr(item.Value)

						case "Min Scale":
							s, _ := strconv.ParseInt(item.Value, 10, 64)
							i := int(s)
							newApp.MustScale().MinReplicas = &i

						case "Max Scale":
							s, _ := strconv.ParseInt(item.Value, 10, 64)
							i := int(s)
							newApp.MustScale().MaxReplicas = &i

						case "Environment variables":
							for _, env := range item.Items {
								c.Env = append(c.Env, &AcaAppEnv{
									Name:  StringPtr(env.Title),
									Value: StringPtr(env.Value),
								})
							}

						default:
							panic("Unknown c.item: " + item.Title)
						}
					}
				}

			case "Bindings":
				for _, bindSec := range item.Items { // bind=Section
					svc := bindSec.GetProp("Service")
					name := bindSec.GetProp("Name")

					newApp.Properties.Template.ServiceBinds =
						append(newApp.MustTemplate().ServiceBinds,
							&AcaAppServiceBind{
								ServiceId: NilStringPtr(svc),
								Name:      NilStringPtr(name),
							})
				}

			default:
				panic("Unknown item: " + item.Title)
			}
		}
	}

	data, _ := json.MarshalIndent(newApp, "", "  ")

	r.Object = newApp
	r.RawData = data
}

func (app *AcaApp) ToARMJson() string {
	WhyMarshal = "ARM"
	data, _ := json.MarshalIndent(app, "", "  ")
	WhyMarshal = ""

	return string(data)
}

func (app *AcaApp) ToJson() string {
	data, _ := json.MarshalIndent(app, "", "  ")
	return string(data)
}

func (app *AcaApp) HideServerFields() {
	if app.Properties != nil && app.Properties.Configuration != nil {
		c := app.Properties.Configuration
		if (*c == AcaAppConfiguration{}) {
			app.Properties.Configuration = nil
		}
	}
}

func AcaFromARMJson(data []byte) *ResourceBase {
	tmp := struct{ ID string }{}
	err := json.Unmarshal(data, &tmp)
	NoErr(err, "Error parsing resource: %s", err)

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
	app.Subscription = GetConfigProperty("defaults.Subscription")
	app.ResourceGroup = GetConfigProperty("defaults.ResourceGroup")
	app.Type = "Microsoft.App/containerApps"
	app.Name, _ = cmd.Flags().GetString("name")
	app.APIVersion = GetResourceDef(app.Type).Defaults["APIVERSION"]
	app.NiceType = cmd.CalledAs()

	app.Stage = GetConfigProperty("currentStage")
	app.Filename = fmt.Sprintf("%s-%s.json", app.NiceType, app.Name)

	if cmd.Flags().Changed("environment") && GetConfigProperty("aca-env") == "" {
		env, _ := cmd.Flags().GetString("environment")
		SetConfigProperty("defaults.aca-env", env, false)
	}

	set := SetStringProp(app, cmd.Flags(), "environment",
		`{"properties":{"environmentId":%s}}`)
	if !set && app.Properties == nil || app.Properties.EnvironmentId == nil {
		env := GetConfigProperty("defaults.aca-env")
		if env == "" {
			ErrStop("Missing the aca-env value. "+
				"Use either '--environment=???' "+
				"or '%s set defaults.aca-env=???'", APP)
		}
		SetJson(app, `{"properties":{"environmentId":%q}}`, env)
	}

	if cmd.Flags().Changed("subscription") {
		sub, _ := cmd.Flags().GetString("subscription")
		if sub == "" {
			sub = GetConfigProperty("defaults.Subscription")
		}
		app.Subscription = sub
		app.ID = app.AsID()
	}

	if cmd.Flags().Changed("resource-group") {
		rg, _ := cmd.Flags().GetString("resource-group")
		if rg == "" {
			rg = GetConfigProperty("defaults.ResourceGroup")
		}
		app.ResourceGroup = rg
		app.ID = app.AsID()
	}

	if cmd.Flags().Changed("location") {
		loc, _ := cmd.Flags().GetString("location")
		if loc == "" {
			loc = GetConfigProperty("defaults.Location")
		}
		app.Location = &loc
	}

	// Now set the app to be a dev mode service
	SetJson(app, `{"properties":{"configuration":{"service":{"type":%q}}}}`,
		service)

	app.Save()
	p, _ := cmd.Flags().GetBool("up")
	if p || GetConfigProperty("defaults.up") == "true" {
		app.Provision()
	}
}

func AddAcaAppFunc(cmd *cobra.Command, args []string) {
	log.VPrintf(2, ">Enter: AddAcaAppFunc (%q)", args)
	defer log.VPrintf(2, "<Exit: AddAcaAppFunc")

	app := &AcaApp{}
	app.Object = app

	// ResourceBase stuff
	app.Subscription = GetConfigProperty("defaults.Subscription")
	app.ResourceGroup = GetConfigProperty("defaults.ResourceGroup")
	app.Location = StringPtr(GetConfigProperty("defaults.Location"))
	app.Type = "Microsoft.App/containerApps"
	app.Name, _ = cmd.Flags().GetString("name")
	app.APIVersion = GetResourceDef(app.Type).Defaults["APIVERSION"]
	app.NiceType = "aca-app"

	app.Stage = GetConfigProperty("currentStage")
	app.Filename = fmt.Sprintf("%s-%s.json", app.NiceType, app.Name)

	// App specific stuff
	// app.Location = StringPtr(GetConfigProperty("defaults.Location"))

	app.ProcessFlags(cmd)
	app.Save()

	p, _ := cmd.Flags().GetBool("up")
	if p || GetConfigProperty("defaults.up") == "true" {
		app.Provision()
	}
}

func UpdateAcaAppFunc(cmd *cobra.Command, args []string) {
	log.VPrintf(2, ">Enter: UpdateAcaAppFunc (%q)", args)
	defer log.VPrintf(2, "<Exit: UpdateAcaAppFunc")

	stage := GetConfigProperty("currentStage")
	name, _ := cmd.Flags().GetString("name")
	name = fmt.Sprintf("%s-%s.json", "aca-app", name)
	res, err := ResourceFromFile(stage, name)
	NoErr(err, "Resource %s/%s not found", cmd.CalledAs(), name)

	app := res.Object.(*AcaApp)

	app.ProcessFlags(cmd)
	app.Save()

	p, _ := cmd.Flags().GetBool("up")
	if p || GetConfigProperty("defaults.up") == "true" {
		app.Provision()
	}
}

func (app *AcaApp) ProcessFlags(cmd *cobra.Command) {
	log.VPrintf(2, ">Enter: ProcessFlags")
	defer log.VPrintf(2, "<Exit: ProcessFlags")

	SetStringProp(app, cmd.Flags(), "image",
		`{"properties":{"template":{"containers":[{"image":%s}]}}}`)

	// TODO make sure 'environ' exists
	if cmd.Flags().Changed("environment") && GetConfigProperty("aca-env") == "" {
		env, _ := cmd.Flags().GetString("environment")
		SetConfigProperty("defaults.aca-env", env, false)
	}

	set := SetStringProp(app, cmd.Flags(), "environment",
		`{"properties":{"environmentId":%s}}`)
	if !set && app.Properties != nil && app.Properties.EnvironmentId == nil {
		env := GetConfigProperty("defaults.aca-env")
		if env == "" {
			ErrStop("Missing the aca-env value. "+
				"Use either '--environment=???' "+
				"or '%s set defaults.aca-env=???'", APP)
		}
		SetJson(app, `{"properties":{"environmentId":%q}}`, env)
	}

	if cmd.Flags().Changed("subscription") {
		sub, _ := cmd.Flags().GetString("subscription")
		if sub == "" {
			sub = GetConfigProperty("defaults.Subscription")
		}
		app.Subscription = sub
		app.ID = app.AsID()
	}

	if cmd.Flags().Changed("resource-group") {
		rg, _ := cmd.Flags().GetString("resource-group")
		if rg == "" {
			rg = GetConfigProperty("defaults.ResourceGroup")
		}
		app.ResourceGroup = rg
		app.ID = app.AsID()
	}

	if cmd.Flags().Changed("location") {
		loc, _ := cmd.Flags().GetString("location")
		if loc == "" {
			loc = GetConfigProperty("defaults.Location")
		}
		app.Location = &loc
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

	if cmd.Flags().Changed("internal") {
		SetJson(app,
			`{"properties":{"configuration":{"ingress":{"external":false}}}}`)
	}
	if cmd.Flags().Changed("external") {
		SetJson(app,
			`{"properties":{"configuration":{"ingress":{"external":true}}}}`)
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
