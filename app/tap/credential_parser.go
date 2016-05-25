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

package main

import (
	"errors"
	"regexp"
	"strconv"
	"strings"

	"k8s.io/kubernetes/pkg/api"

	"github.com/trustedanalytics/kubernetes-broker/catalog"
	"github.com/trustedanalytics/kubernetes-broker/k8s"
)

func ParseCredentialMappingAdvanced(serviceMetaName string, svcCreds []ServiceCredential, pods []k8s.PodEnvs,
	blueprint catalog.KubernetesBlueprint) (string, error) {
	var err error = nil
	var parsedMapping string

	if blueprint.ReplicaTemplate != "" {
		parsedMapping, err = parseSvcCredsForClusteredPlan(blueprint, svcCreds)
		if err != nil {
			return "", err
		}
	} else {
		parsedMapping, err = parseSvcCredsForSimplePlan(blueprint.CredentialsMapping, svcCreds)
		if err != nil {
			return "", err
		}
	}

	parsedMapping = strings.Replace(parsedMapping, "$name", serviceMetaName, -1)

	//TODO why we put this?
	parsedMapping = strings.Replace(parsedMapping, "$uri", "NOTSUPPORTED://yet", -1)
	parsedMapping = parseEnvs(parsedMapping, pods)

	return parsedMapping, nil
}

func parseSvcCredsForClusteredPlan(blueprint catalog.KubernetesBlueprint, svcCreds []ServiceCredential) (string, error) {
	var err error = nil
	parsedReplicas := []string{}

	for _, svc := range svcCreds {
		templateToParse := blueprint.ReplicaTemplate

		templateToParse = strings.Replace(templateToParse, "$hostname", svc.Host, -1)
		templateToParse = strings.Replace(templateToParse, "$nodeName", svc.Name, -1)
		templateToParse, err = parsePorts(templateToParse, svc.Ports)
		if err != nil {
			return "", err
		}
		parsedReplicas = append(parsedReplicas, templateToParse)
	}

	parsedMapping := blueprint.CredentialsMapping
	parsedMapping = strings.Replace(parsedMapping, "$nodes", strings.Join(parsedReplicas, ","), -1)
	return parsedMapping, nil
}

func parseSvcCredsForSimplePlan(templateToParse string, svcCreds []ServiceCredential) (string, error) {
	var err error = nil

	//TODO we should refactor all our credentialmapping.json in simple plans because they don't support more then 1 service
	for _, svc := range svcCreds {
		templateToParse = strings.Replace(templateToParse, "$hostname", svc.Host, -1)
		templateToParse, err = parsePorts(templateToParse, svc.Ports)
		if err != nil {
			return "", err
		}

		//TODO since we don't support more then one service there is no point to continue this
		break
	}
	return templateToParse, nil
}

func parseEnvs(templateToParse string, pods []k8s.PodEnvs) string {
	var re_env = regexp.MustCompile(`\$env_[A-Za-z\-_]+`)
	parsed_env_list := re_env.FindAllString(templateToParse, -1)
	logger.Debug("$env_ variables: ", parsed_env_list)

	allEnv := map[string]string{}
	for _, pod := range pods {
		for _, container := range pod.Containers {
			allEnv = appendMaps(allEnv, container.Envs)
		}
	}

	for _, parsed_env := range parsed_env_list {
		parsed_env = strings.Replace(parsed_env, "$env_", "", -1)
		templateToParse = strings.Replace(templateToParse, "$env_"+parsed_env, allEnv[parsed_env], -1)
	}
	return templateToParse
}

func parsePorts(templateToParse string, ports []api.ServicePort) (string, error) {
	var re = regexp.MustCompile(`\$port_[0-9]+`)
	parsed_ports_list := re.FindAllString(templateToParse, -1)
	logger.Debug("$port_ variables: ", parsed_ports_list)

	for _, parsed_port := range parsed_ports_list {
		expected_port_num_strs := strings.Split(parsed_port, "_")
		if len(expected_port_num_strs) != 2 {
			return "", errors.New("Template error: Port mapping incorrect on " + parsed_port)
		}
		expected_port_num_int, err := strconv.Atoi(expected_port_num_strs[1])
		if err != nil {
			return "", errors.New("Parsing error: Port value has incorrect fromat " + expected_port_num_strs[1])
		}
		for _, p := range ports {
			target_port_int, err := strconv.Atoi(p.TargetPort.String())
			if err != nil {
				return "", errors.New("Parsing error: TargetPort value has incorrect fromat " + p.TargetPort.String())
			}
			if target_port_int == expected_port_num_int {
				templateToParse = strings.Replace(templateToParse, parsed_port, strconv.Itoa(p.NodePort), -1)
				continue
			}
		}
	}
	return templateToParse, nil
}

func appendMaps(source, newMap map[string]string) map[string]string {
	for k, v := range newMap {
		source[k] = v
	}
	return source
}
