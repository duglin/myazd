package main

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func initRedis() {
	return
	setupRedisCmds()
	setupRedisResourceDefs()
	RegisteredParsers = append(RegisteredParsers, RedisFromARMJson)
}

func setupRedisCmds() {
	cmd := &cobra.Command{
		Use:   "redis",
		Short: "Add a Redis instance",
		Run:   ResourceAddFunc,
	}
	cmd.Flags().StringArrayP("name", "n", nil, "Name of Redis instance")
	cmd.Flags().StringArrayP("type", "t", nil, "'dev' or 'managed'")
	cmd.MarkFlagRequired("name")
	AddCmd.AddCommand(cmd)
}

func setupRedisResourceDefs() {
	AddResourceDef(&ResourceDef{
		Type: "Microsoft.DocumentDB/databaseAccounts",
		URL:  "https://management.azure.com/subscriptions/${SUBSCRIPTION}/resourceGroups/${RESOURCEGROUP}/providers/Microsoft.DocumentDB/databaseAccounts/${NAME}?api-version=${APIVERSION}",
		Defaults: map[string]string{
			"APIVERSION": "2021-04-01-preview",
		},
	})

	AddResourceDef(&ResourceDef{
		Type: "Microsoft.Cache/redis",
		URL:  "https://management.azure.com/subscriptions/${SUBSCRIPTION}/resourceGroups/${RESOURCEGROUP}/providers/Microsoft.Cache/redis/${NAME}?api-version=${APIVERSION}",
		Defaults: map[string]string{
			"APIVERSION": "2023-04-01",
		},
	})

}

type RedisAppConfiguration struct {
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
	// service
}

type RedisProperties struct {
}

func (rp *RedisProperties) MarshalJSON() ([]byte, error) {
	tmpRp := *rp
	if WhyMarshal == "ARM" {
	}
	return json.Marshal(tmpRp)
}

type Redis struct {
	ResourceBase

	Location   *string          `json:"location,omitempty"`
	Properties *RedisProperties `json:"properties,omitempty"`
}

func (r *Redis) DependsOn() []*ResourceReference {
	refs := []*ResourceReference{}

	if r.Properties != nil {
		// resRef := r.Properties.ResolveXXX()
		// refs = append(refs, resRef)
	}

	return refs
}

func (r *Redis) ToARMJson() string {
	saveID := r.ID
	r.ID = ""

	WhyMarshal = "ARM"
	data, _ := json.MarshalIndent(r, "", "  ")
	WhyMarshal = ""

	r.ID = saveID
	return string(data)
}

func RedisFromARMJson(data []byte) *ResourceBase {
	return nil
}

func AddRedisFunc(cmd *cobra.Command, args []string) {
	redis := &Redis{}
	redis.Object = redis

	// ResourceBase stuff
	redis.Subscription = GetConfigProperty("defaults.Subscription")
	redis.ResourceGroup = GetConfigProperty("defaults.ResourceGroup")
	redis.Type = "Microsoft.Cache/redis"
	redis.Name, _ = cmd.Flags().GetString("name")
	redis.APIVersion = GetResourceDef(redis.Type).Defaults["APIVERSION"]
	redis.NiceType = "redis"

	redis.Stage = GetConfigProperty("currentStage")
	redis.Filename = fmt.Sprintf("%s-%s.json", redis.NiceType, redis.Name)

	// Redis specific stuff
	redis.Location = StringPtr(GetConfigProperty("defaults.Location"))

	redis.ProcessFlags(cmd)
	redis.Save()
	redis.Provision()
}

func (redis *Redis) ProcessFlags(cmd *cobra.Command) {
}

/*
func (r  *Redis) Save() {
	log.VPrintf(0, "In redis save")
	r.ResourceBase.Save()
}
*/

func (r *Redis) HideServerFields(diffR ARMResource) {
}
