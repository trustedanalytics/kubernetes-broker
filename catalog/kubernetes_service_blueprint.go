/**
 * Copyright (c) 2016 Intel Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package catalog

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"os"
	"regexp"
	"strconv"
	"strings"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
)

type KubernetesBlueprint struct {
	Id                    int
	SecretsJson           []string
	DeploymentJson        []string
	ServiceJson           []string
	ServiceAcccountJson   []string
	PersistentVolumeClaim []string
	CredentialsMapping    string
	ReplicaTemplate       string
	UriTemplate           string
}

type KubernetesComponent struct {
	PersistentVolumeClaims []*api.PersistentVolumeClaim `json:"persistentVolumeClaims"`
	Deployments            []*extensions.Deployment     `json:"deployments"`
	Services               []*api.Service               `json:"services"`
	ServiceAccounts        []*api.ServiceAccount        `json:"serviceAccounts"`
	Secrets                []*api.Secret                `json:"secrets"`
}

var TEMP_DYNAMIC_BLUEPRINTS = map[string]KubernetesBlueprint{}
var possible_rand_chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

func GetParsedKubernetesComponentByTemplate(catalogPath, instanceId, org, space string, temp *TemplateMetadata) (*KubernetesComponent, error) {
	blueprint, err := GetKubernetesBlueprint(catalogPath, temp.TemplateDirName, temp.TemplatePlanDirName, temp.Id)
	if err != nil {
		return nil, err
	}

	return ParseKubernetesComponent(blueprint, instanceId, temp.Id, temp.Id, org, space)
}

func GetParsedKubernetesComponentByServiceAndPlan(catalogPath, instanceId, org, space string, svcMeta ServiceMetadata, planMeta PlanMetadata) (*KubernetesComponent, error) {
	blueprint, err := GetKubernetesBlueprint(catalogPath, svcMeta.InternalId, planMeta.InternalId, svcMeta.Id)
	if err != nil {
		return nil, err
	}

	return ParseKubernetesComponent(blueprint, instanceId, svcMeta.Id, planMeta.Id, org, space)
}

func ParseKubernetesComponent(blueprint KubernetesBlueprint, instanceId, svcMetaId, planMetaId, org, space string) (*KubernetesComponent, error) {
	parsedPVC := []string{}
	for i, pvc := range blueprint.PersistentVolumeClaim {
		parsedPVC = append(parsedPVC, adjust_params(pvc, org, space, instanceId, svcMetaId, planMetaId, i))
	}
	blueprint.PersistentVolumeClaim = parsedPVC

	parsedSecrets := []string{}
	for i, secret := range blueprint.SecretsJson {
		parsedSecrets = append(parsedSecrets, adjust_params(secret, org, space, instanceId, svcMetaId, planMetaId, i))
	}
	blueprint.SecretsJson = parsedSecrets

	parsedDeployments := []string{}
	for i, deployment := range blueprint.DeploymentJson {
		parsedDeployments = append(parsedDeployments, adjust_params(deployment, org, space, instanceId, svcMetaId, planMetaId, i))
	}
	blueprint.DeploymentJson = parsedDeployments

	parsedSvcs := []string{}
	for i, svc := range blueprint.ServiceJson {
		parsedSvcs = append(parsedSvcs, adjust_params(svc, org, space, instanceId, svcMetaId, planMetaId, i))
	}
	blueprint.ServiceJson = parsedSvcs

	parsedAccountSvcs := []string{}
	for i, svc := range blueprint.ServiceAcccountJson {
		parsedAccountSvcs = append(parsedAccountSvcs, adjust_params(svc, org, space, instanceId, svcMetaId, planMetaId, i))
	}
	blueprint.ServiceAcccountJson = parsedAccountSvcs

	return CreateKubernetesComponentFromBlueprint(blueprint, false)
}

func CreateKubernetesComponentFromBlueprint(blueprint KubernetesBlueprint, encodeSecrets bool) (*KubernetesComponent, error) {
	result := &KubernetesComponent{}

	for _, pvc := range blueprint.PersistentVolumeClaim {
		parsedPVC := &api.PersistentVolumeClaim{}
		err := json.Unmarshal([]byte(pvc), parsedPVC)
		if err != nil {
			logger.Error("Unmarshalling PersistentVolumeClaim error:", err)
			return result, err
		}
		result.PersistentVolumeClaims = append(result.PersistentVolumeClaims, parsedPVC)
	}

	for _, secret := range blueprint.SecretsJson {
		parsedSecret := &api.Secret{}
		if encodeSecrets {
			secret = encodeByte64ToString(secret)
		}
		err := json.Unmarshal([]byte(secret), parsedSecret)
		if err != nil {
			logger.Error("Unmarshalling secret error:", err)
			return result, err
		}
		result.Secrets = append(result.Secrets, parsedSecret)
	}

	for _, deployment := range blueprint.DeploymentJson {
		parsedDeployemnt := &extensions.Deployment{}
		err := json.Unmarshal([]byte(deployment), parsedDeployemnt)
		if err != nil {
			logger.Error("Unmarshalling deployment error:", err)
			return result, err
		}
		result.Deployments = append(result.Deployments, parsedDeployemnt)
	}

	for _, svc := range blueprint.ServiceJson {
		parsedSvc := &api.Service{}
		err := json.Unmarshal([]byte(svc), parsedSvc)
		if err != nil {
			logger.Error("Unmarshalling service error:", err)
			return result, err
		}
		result.Services = append(result.Services, parsedSvc)
	}

	for _, Accsvc := range blueprint.ServiceAcccountJson {
		parsedAccSvc := &api.ServiceAccount{}
		err := json.Unmarshal([]byte(Accsvc), parsedAccSvc)
		if err != nil {
			logger.Error("Unmarshalling service account error:", err)
			return result, err
		}
		result.ServiceAccounts = append(result.ServiceAccounts, parsedAccSvc)
	}
	return result, nil
}

func GetCatalogFilesPath(catalogPath, templateDirName, planDirName string) (plan_path, secrets_path, k8s_plan_path string) {
	svc_path := catalogPath + templateDirName + "/"
	plan_path = svc_path + planDirName + "/"
	secrets_path = svc_path + "secretTemplates/"
	k8s_plan_path = plan_path + "k8s/"
	return
}

func GetKubernetesBlueprint(catalogPath, templateDirName, planDirName, templateId string) (KubernetesBlueprint, error) {
	result := KubernetesBlueprint{}
	var err error
	var secretTemplatesExists bool

	//todo replace it by psotgres!
	// first check in registred dynamic templates:
	if blueprint, ok := TEMP_DYNAMIC_BLUEPRINTS[templateId]; ok {
		return blueprint, nil
	}

	plan_path, secrets_path, k8s_plan_path := GetCatalogFilesPath(catalogPath, templateDirName, planDirName)

	result.PersistentVolumeClaim, err = read_k8s_json_files_with_prefix_from_dir(k8s_plan_path, "persistentvolumeclaim")
	if err != nil {
		logger.Error("Error reading PersistentVolumeClaim file", err)
		return result, err
	}

	result.SecretsJson, err = read_k8s_json_files_with_prefix_from_dir(k8s_plan_path, "secret")
	if err != nil {
		logger.Error("Error reading secret files", err)
		return result, err
	}

	//if secret.jsons are not present k8s_plan_path, check if secretTemplates dir exists and read secret.jsons
	//from there
	if len(result.SecretsJson) == 0 {
		secretTemplatesExists, err = check_if_file_or_dir_exists(secrets_path)
		if err != nil {
			logger.Error("Error checking if secretTemplates exists!", err)
			return result, err
		}

		if secretTemplatesExists {
			result.SecretsJson, err = read_k8s_files_with_prefix_from_dir(secrets_path, "secret")
			if err != nil {
				logger.Error("Error reading secret files from secretTemplates path", err)
				return result, err
			}
		}
	}
	result.DeploymentJson, err = read_k8s_json_files_with_prefix_from_dir(k8s_plan_path, "deployment")
	if err != nil {
		logger.Error("Error reading deployment file", err)
		return result, err
	}

	result.ServiceJson, err = read_k8s_json_files_with_prefix_from_dir(k8s_plan_path, "service")
	if err != nil {
		logger.Error("Error reading service file", err)
		return result, err
	}

	result.ServiceAcccountJson, err = read_k8s_json_files_with_prefix_from_dir(k8s_plan_path, "account")
	if err != nil {
		logger.Error("Error reading account file", err)
		return result, err
	}

	credentialMappings, err := read_k8s_json_files_with_prefix_from_dir(plan_path, "credentials-mappings")
	if err != nil {
		logger.Error("Error reading credential mappings file", err)
		return result, err
	}

	replicas, err := read_k8s_json_files_with_prefix_from_dir(plan_path, "node_template")
	if err != nil {
		logger.Error("Error reading replica template files", plan_path)
		return result, err
	}

	uriTemplate, err := read_k8s_files_with_prefix_from_dir(plan_path, "uri_cluster_template")
	if err != nil {
		logger.Error("Error reading uri template files", plan_path)
		return result, err
	}

	if len(credentialMappings) > 1 || len(replicas) > 1 {
		logger.Error("WARNING: Multiple env mappings or replica templates files found... looks like a problem with catalog structure. Will use only the first one.")
	}
	if len(credentialMappings) > 0 {
		result.CredentialsMapping = string(credentialMappings[0])
	}
	if len(replicas) > 0 {
		result.ReplicaTemplate = string(replicas[0])
	}
	if len(uriTemplate) > 0 {
		result.UriTemplate = string(uriTemplate[0])
	}
	return result, nil
}

func adjust_params(content, org, space, cf_service_id string, svc_meta_id, plan_meta_id string, idx int) string {
	f := content
	f = strings.Replace(f, "$org", org, -1)
	f = strings.Replace(f, "$space", space, -1)
	f = strings.Replace(f, "$catalog_service_id", svc_meta_id, -1)
	f = strings.Replace(f, "$catalog_plan_id", plan_meta_id, -1)
	f = strings.Replace(f, "$service_id", cf_service_id, -1)

	proper_dns_name := cf_id_to_domain_valid_name(cf_service_id + "x" + strconv.Itoa(idx))
	f = strings.Replace(f, "$idx_and_short_serviceid", proper_dns_name, -1)

	proper_short_dns_name := cf_id_to_domain_valid_name(cf_service_id)
	f = strings.Replace(f, "$short_serviceid", proper_short_dns_name, -1)

	for i := 0; i < 9; i++ {
		f = strings.Replace(f, "$random"+strconv.Itoa(i), get_random_string(10), -1)
	}
	f = encodeByte64ToString(f)
	return f
}

func encodeByte64ToString(content string) string {
	rp := regexp.MustCompile(`\$base64\-(.*)\"`)
	fs := rp.FindAllString(content, -1)
	for _, sub := range fs {
		sub = strings.Replace(sub, "$base64-", "", -1)
		sub = strings.Replace(sub, "\"", "", -1)
		content = strings.Replace(content, "$base64-"+sub, base64.StdEncoding.EncodeToString([]byte(sub)), -1)
	}

	return content
}

/*
 * x, as "Service \"181864c5711445\" is invalid: metadata.name: invalid value '181864c5711445',
 Details: must be a DNS 952 label (at most 24 characters, matching regex [a-z]([-a-z0-9]*[a-z0-9])?): e.g. \"my-name\"",
*/
func cf_id_to_domain_valid_name(cf_id string) string {
	return "x" + strings.Replace(cf_id[0:15], "-", "", -1)
}

func get_random_string(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = possible_rand_chars[rand.Intn(len(possible_rand_chars))]
	}
	return string(b)
}

func read_k8s_json_files_with_prefix_from_dir(path, prefix string) ([]string, error) {
	return read_k8s_files_with_prefix_suffix_from_dir(path, prefix, ".json")
}

func read_k8s_files_with_prefix_from_dir(path, prefix string) ([]string, error) {
	return read_k8s_files_with_prefix_suffix_from_dir(path, prefix, "")
}

func read_k8s_files_with_prefix_suffix_from_dir(path, prefix string, suffix string) ([]string, error) {
	logger.Debug("read_k8s_files_with_prefix_from_dir", path, prefix)
	results := []string{}
	file_in_path, err := ioutil.ReadDir(path)
	if err != nil {
		logger.Error("[read_k8s_files_with_prefix_from_dir] Read Dir failed!:", err)
		return results, err
	}
	for _, f := range file_in_path {
		if strings.HasPrefix(f.Name(), prefix) && strings.HasSuffix(f.Name(), suffix) {
			fcontent, err := ioutil.ReadFile(path + "/" + f.Name())
			if err != nil {
				logger.Error("[read_k8s_files_with_prefix_from_dir] Error reading file:", fcontent, err)
				return results, err
			}
			results = append(results, string(fcontent))
		}
	}
	return results, nil
}

func save_k8s_file_in_dir(path, fileName string, file interface{}) error {
	logger.Debug("[save_k8s_file_in_dir]", path)

	bBody, err := json.Marshal(file)
	if err != nil {
		logger.Error("[save_k8s_file_in_dir] Crate Dir failed!:", err)
		return err
	}

	err = os.MkdirAll(path, 0777)
	if err != nil {
		logger.Error("[save_k8s_file_in_dir] Crate Dir failed!:", err)
		return err
	}
	err = ioutil.WriteFile(path+"/"+fileName, bBody, 0666)
	if err != nil {
		logger.Error("[save_k8s_file_in_dir] Save file failed:", err)
		return err
	}
	return nil
}

func check_if_file_or_dir_exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		} else {
			return false, err
		}
	}
	return true, nil
}
