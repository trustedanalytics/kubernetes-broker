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
package k8s

import (
	"fmt"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/apis/extensions"
	"k8s.io/kubernetes/pkg/labels"
)

type DeploymentManager interface {
	DeleteAll(selector labels.Selector) error
	UpdateReplicasNumber(name string, count int) error
	Create(replicationController *extensions.Deployment) (*extensions.Deployment, error)
	List(selector labels.Selector) (*extensions.DeploymentList, error)
}

type DeploymentConnector struct {
	client ExtensionsInterface
}

func NewDeploymentControllerManager(client ExtensionsInterface) *DeploymentConnector {
	return &DeploymentConnector{client: client}
}

func (r *DeploymentConnector) DeleteAll(selector labels.Selector) error {
	logger.Debug("Delete deployment selector:", selector)
	deployments, err := r.List(selector)
	if err != nil {
		logger.Error("List deployment failed:", err)
		return err
	}

	for _, deployment := range deployments.Items {
		name := deployment.ObjectMeta.Name

		if err := r.UpdateReplicasNumber(name, 0); err != nil {
			logger.Error("UpdateReplicasNumber for deployment failed:", err)
			return err
		}
		logger.Debug("Deleting deployment:", name)
		err = r.client.Deployments(api.NamespaceDefault).Delete(name, &api.DeleteOptions{})
		if err != nil {
			logger.Error("Delete deployment failed:", err)
			return err
		}
	}
	return nil
}

func (r *DeploymentConnector) UpdateReplicasNumber(name string, count int) error {
	logger.Info(fmt.Sprintf("Set replicas to %d. Deployment name: %s", count, name))
	deploymnet, err := r.client.Deployments(api.NamespaceDefault).Get(name)
	if err != nil {
		return err
	}
	deploymnet.Spec.Replicas = count
	if _, err = r.client.Deployments(api.NamespaceDefault).Update(deploymnet); err != nil {
		return err
	}

	return nil
}

func (r *DeploymentConnector) Create(deployment *extensions.Deployment) (*extensions.Deployment, error) {
	return r.client.Deployments(api.NamespaceDefault).Create(deployment)
}

func (r *DeploymentConnector) List(selector labels.Selector) (*extensions.DeploymentList, error) {
	logger.Debug("List DeploymentList selector:", selector)
	return r.client.Deployments(api.NamespaceDefault).List(api.ListOptions{
		LabelSelector: selector,
	})
}
