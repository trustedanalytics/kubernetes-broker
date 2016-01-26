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
package k8s

import (
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/labels"
)

type ReplicationControllerManager interface {
	DeleteAll(selector labels.Selector) error
	UpdateReplicasNumber(name string, count int) error
	Create(replicationController *api.ReplicationController) (*api.ReplicationController, error)
	List(selector labels.Selector) (*api.ReplicationControllerList, error)
}

type RcConnector struct {
	client KubernetesClient
}

func NewReplicationControllerManager(client KubernetesClient) *RcConnector {
	return &RcConnector{client: client}
}

func (r *RcConnector) DeleteAll(selector labels.Selector) error {
	logger.Debug("[DeleteAllReplicationControllers] selector:", selector)
	rcs, err := r.List(selector)

	if err != nil {
		logger.Error("[DeleteAllReplicationControllers] List replication controlles failed:", err)
		return err
	}
	for _, i := range rcs.Items {
		name := i.ObjectMeta.Name

		if err := r.UpdateReplicasNumber(name, 0); err != nil {
			logger.Error("[DeleteAllReplicationControllers] Get replication controlles failed:", err)
			return err
		}
		logger.Debug("[DeleteAllReplicationControllers] Deleting replication controller:", name)
		err = r.client.ReplicationControllers(api.NamespaceDefault).Delete(name)
		if err != nil {
			logger.Error("[DeleteAllReplicationControllers] Delete replication controller failed:", err)
			return err
		}
	}

	return nil
}

func (r *RcConnector) UpdateReplicasNumber(name string, count int) error {
	logger.Info("[DeleteAllReplicationControllers] Set replicas to 0:", name)
	rc, err := r.client.ReplicationControllers(api.NamespaceDefault).Get(name)
	if err != nil {
		return err
	}
	rc.Spec.Replicas = count
	if _, err = r.client.ReplicationControllers(api.NamespaceDefault).Update(rc); err != nil {
		return err
	}

	return nil
}

func (r *RcConnector) Create(replicationController *api.ReplicationController) (*api.ReplicationController, error) {
	return r.client.ReplicationControllers(api.NamespaceDefault).Create(replicationController)
}

func (r *RcConnector) List(selector labels.Selector) (*api.ReplicationControllerList, error) {
	logger.Debug("[ListReplicationControllers] selector:", selector)
	return r.client.ReplicationControllers(api.NamespaceDefault).List(api.ListOptions{
		LabelSelector: selector,
	})
}
