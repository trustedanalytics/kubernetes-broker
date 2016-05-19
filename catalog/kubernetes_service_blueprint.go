/**
 * Copyright (c) 2015 Intel Corporation
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
	"regexp"
	"strconv"
	"strings"

	"k8s.io/kubernetes/pkg/api"
)

type KubernetesBlueprint struct {
	Id                        int
	SecretsJson               []string
	ReplicationControllerJson []string
	ServiceJson               []string
	ServiceAcccountJson       []string
	PersistentVolumeClaim     []string
	CredentialsMapping        string
	ReplicaTemplate           string
}

type KubernetesComponent struct {
	PersistentVolumeClaim  []*api.PersistentVolumeClaim
	ReplicationControllers []*api.ReplicationController
	Services               []*api.Service
	ServiceAccounts        []*api.ServiceAccount
	Secrets                []*api.Secret
}

var TEMP_DYNAMIC_BLUEPRINTS = map[string]KubernetesBlueprint{}
var possible_rand_chars = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890")

func GetParsedKubernetesComponent(catalogPath, instanceId, org, space string, svcMeta ServiceMetadata, planMeta PlanMetadata) (*KubernetesComponent, error) {
	blueprint, err := GetKubernetesBlueprintByServiceAndPlan(catalogPath, svcMeta, planMeta)
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

	parsedRcs := []string{}
	for i, rc := range blueprint.ReplicationControllerJson {
		parsedRcs = append(parsedRcs, adjust_params(rc, org, space, instanceId, svcMetaId, planMetaId, i))
	}
	blueprint.ReplicationControllerJson = parsedRcs

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

	return CreateKubernetesComponentFromBlueprint(blueprint)
}

func CreateKubernetesComponentFromBlueprint(blueprint KubernetesBlueprint) (*KubernetesComponent, error) {
	result := &KubernetesComponent{}

	for _, pvc := range blueprint.PersistentVolumeClaim {
		parsedPVC := &api.PersistentVolumeClaim{}
		err := json.Unmarshal([]byte(pvc), parsedPVC)
		if err != nil {
			logger.Error("[ParseKubernetesComponenets] Unmarshalling PersistentVolumeClaim error:", err)
			return result, err
		}
		result.PersistentVolumeClaim = append(result.PersistentVolumeClaim, parsedPVC)
	}

	for _, secret := range blueprint.SecretsJson {
		parsedSecret := &api.Secret{}
		err := json.Unmarshal([]byte(secret), parsedSecret)
		if err != nil {
			logger.Error("[ParseKubernetesComponenets] Unmarshalling secret error:", err)
			return result, err
		}
		result.Secrets = append(result.Secrets, parsedSecret)
	}

	for _, rc := range blueprint.ReplicationControllerJson {
		parsedRc := &api.ReplicationController{}
		err := json.Unmarshal([]byte(rc), parsedRc)
		if err != nil {
			logger.Error("[ParseKubernetesComponenets] Unmarshalling replication controller error:", err)
			return result, err
		}
		result.ReplicationControllers = append(result.ReplicationControllers, parsedRc)
	}

	for _, svc := range blueprint.ServiceJson {
		parsedSvc := &api.Service{}
		err := json.Unmarshal([]byte(svc), parsedSvc)
		if err != nil {
			logger.Error("[ParseKubernetesComponenets] Unmarshalling service error:", err)
			return result, err
		}
		result.Services = append(result.Services, parsedSvc)
	}

	for _, Accsvc := range blueprint.ServiceAcccountJson {
		parsedAccSvc := &api.ServiceAccount{}
		err := json.Unmarshal([]byte(Accsvc), parsedAccSvc)
		if err != nil {
			logger.Error("[ParseKubernetesComponenets] Unmarshalling service account error:", err)
			return result, err
		}
		result.ServiceAccounts = append(result.ServiceAccounts, parsedAccSvc)
	}
	return result, nil
}

func GetKubernetesBlueprintByServiceAndPlan(catalogPath string, svcMeta ServiceMetadata, planMeta PlanMetadata) (KubernetesBlueprint, error) {
	result := KubernetesBlueprint{}
	var err error

	//todo replace it by psotgres!
	// first check in registred dynamic templates:
	if blueprint, ok := TEMP_DYNAMIC_BLUEPRINTS[svcMeta.Id]; ok {
		return blueprint, nil
	}

	svc_path := catalogPath + svcMeta.InternalId + "/"
	plan_path := svc_path + planMeta.InternalId + "/k8s/"

	result.PersistentVolumeClaim, err = read_k8s_files_with_prefix_from_dir(plan_path, "persistentvolumeclaim")
	if err != nil {
		logger.Error("[GetKubernetesBlueprintForServiceAndPlan] Error reading Replication Controller file", err)
		return result, err
	}

	result.SecretsJson, err = read_k8s_files_with_prefix_from_dir(plan_path, "secret")
	if err != nil {
		logger.Error("[GetKubernetesBlueprintForServiceAndPlan] Error reading secret files", err)
		return result, err
	}

	result.ReplicationControllerJson, err = read_k8s_files_with_prefix_from_dir(plan_path, "replicationcontroller")
	if err != nil {
		logger.Error("[GetKubernetesBlueprintForServiceAndPlan] Error reading Replication Controller file", err)
		return result, err
	}

	result.ServiceJson, err = read_k8s_files_with_prefix_from_dir(plan_path, "service")
	if err != nil {
		logger.Error("[GetKubernetesBlueprintForServiceAndPlan] Error reading service file", err)
		return result, err
	}

	result.ServiceAcccountJson, err = read_k8s_files_with_prefix_from_dir(plan_path, "account")
	if err != nil {
		logger.Error("[GetKubernetesBlueprintForServiceAndPlan] Error reading account file", err)
		return result, err
	}

	credentialMappings, err := read_k8s_files_with_prefix_from_dir(svc_path, "credentials-mappings")
	if err != nil {
		logger.Error("[GetKubernetesBlueprintForServiceAndPlan] Error reading credential mappings file", err)
		return result, err
	}

	replicas, err := read_k8s_files_with_prefix_from_dir(svc_path, "node_template")
	if err != nil {
		logger.Error("[GetKubernetesBlueprintForServiceAndPlan] Error reading replica template files", svc_path)
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

	rp := regexp.MustCompile(`\$base64\-(.*)\"`)
	fs := rp.FindAllString(f, -1)
	for _, sub := range fs {
		sub = strings.Replace(sub, "$base64-", "", -1)
		sub = strings.Replace(sub, "\"", "", -1)
		f = strings.Replace(f, "$base64-"+sub, base64.StdEncoding.EncodeToString([]byte(sub)), -1)
	}

	return f
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

func read_k8s_files_with_prefix_from_dir(path, prefix string) ([]string, error) {
	logger.Debug("read_k8s_files_with_prefix_from_dir", path, prefix)
	results := []string{}
	file_in_path, err := ioutil.ReadDir(path)
	if err != nil {
		logger.Error("[read_k8s_files_with_prefix_from_dir] Read Dir failed!:", err)
		return results, err
	}
	for _, f := range file_in_path {
		if strings.HasPrefix(f.Name(), prefix) && strings.HasSuffix(f.Name(), ".json") {
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
