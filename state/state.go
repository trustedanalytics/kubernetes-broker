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

package state

import (
	"sync"
	"time"

	"github.com/trustedanalytics/kubernetes-broker/logger"
)

var logger = logger_wrapper.InitLogger("state")

type StateEvent struct {
	ts    time.Time
	state string
	err   error
}

type StateService interface {
	ReportProgress(guid string, state string, err error)
	HasProgressRecords(guid string) bool
	ReadProgress(guid string) (time.Time, string, error)
}

type StateMemoryService struct{}

/*
	TODO TODO TODO TODO TODO

	After deployment, when etcd|redis|consol|postgres is available (and credentials are filled),
	use the external storage to keep state.
*/

var state_map map[string]StateEvent = make(map[string]StateEvent)
var state_mutex sync.RWMutex

func (s *StateMemoryService) ReportProgress(guid string, state string, err error) {
	logger.Info("[StateMemoryService] service:", guid, ", state:", state, err)
	state_mutex.Lock()
	state_map[guid] = StateEvent{time.Now(), state, err}
	state_mutex.Unlock()
}

func (s *StateMemoryService) HasProgressRecords(guid string) bool {
	state_mutex.RLock()
	ret := false
	if _, ok := state_map[guid]; ok {
		ret = true
	}
	state_mutex.RUnlock()
	return ret
}

func (s *StateMemoryService) ReadProgress(guid string) (time.Time, string, error) {
	state_mutex.RLock()
	se := state_map[guid]
	state_mutex.RUnlock()
	return se.ts, se.state, se.err
}
