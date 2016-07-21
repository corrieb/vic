// Copyright 2016 VMware, Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package exec2

import (
	"sync"
	"time"
)

type StateManager struct {
	containers     map[ID]*Container
	containersLock sync.Mutex
}

func (p *StateManager) getContainerForID(cid ID) *Container {
	p.containersLock.Lock()
	defer p.containersLock.Unlock()
	return p.containers[cid]
}

func (p *StateManager) getContainerForHandle(handle Handle) *Container {
	return p.getContainerForID(handle.(*PendingCommit).ContainerID)
}

func (p *StateManager) createContainer(pc *PendingCommit) *Container {
	c := Container{}
	c.ContainerID = pc.ContainerID
	c.Created = time.Now()
	c.config = pc.Config
	c.runState = Created // presume Created
	c.processes = make(map[ID]*ProcessConfig)
	c.processes[c.ContainerID] = &pc.MainProcess
	c.processState = make(map[ID]*ProcessState)
	c.filesToCopy = pc.FilesToCopy
	c.opaque = make(map[string]interface{})
	if pc.Opaque != nil {
		for k, v := range pc.Opaque {
			c.opaque[k] = v
		}
	}
	p.containersLock.Lock()
	defer p.containersLock.Unlock()
	p.containers[c.ContainerID] = &c
	return &c
}

// Note: ChangeID prevents the caller from making concurrent updates to the same container
func (p *StateManager) updateContainer(pc *PendingCommit) *Container {
	c := p.getContainerForHandle(Handle(pc))
	c.setRunState(pc.RunState)
	c.reconfigure(pc.Config)
	c.pushFiles(pc.FilesToCopy)
	if pc.Opaque != nil {
		for k, v := range pc.Opaque {
			c.opaque[k] = v
		}
	}
	return c
}

func (p *StateManager) filterByRunState(r RunState) []*Container {
	return nil
}

// Not public as this should be part of commit
func (c *Container) setRunState(r RunState) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.runState = r
}

func (c *Container) RunState() RunState {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.runState
}

// Note: this is only for execd processes
func (c *Container) AddProcess(p ProcessConfig) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.processes[p.ProcessID] = &p
}

func (c *Container) Process(id ID) *ProcessConfig {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.processes[id] != nil {
		copy := *c.processes[id]
		return &copy
	}
	return nil
}

func (c *Container) AddProcessState(id ID, state ProcessState) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.processState[id] = &state
}

func (c *Container) UpdateProcessState(id ID, state ProcessRunState) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.processState[id] == nil {
		return // nothign to update
	}
	currentState := c.processState[id]
	if state.Status != nil {
		*currentState.Status = *state.Status
	}
	if state.ExitCode != nil {
		*currentState.ExitCode = *state.ExitCode
	}
	if state.ExitMsg != nil {
		*currentState.ExitMsg = *state.ExitMsg
	}
	if state.FinishedAt != nil {
		*currentState.FinishedAt = *state.FinishedAt
	}
}

// Presumes that the validator has already validated this
// Not public as this should be part of commit
func (c *Container) reconfigure(config Config) {
	c.lock.Lock()
	defer c.lock.Unlock()
	if config.Name != nil {
		*c.config.Name = *config.Name
	}
	if config.Limits != nil {
		if config.Limits.CPUMhz != nil {
			*c.config.Limits.CPUMhz = *config.Limits.CPUMhz
		}
		if config.Limits.MemoryMb != nil {
			*c.config.Limits.MemoryMb = *config.Limits.MemoryMb
		}
	}
	if config.Attach != nil {
		*c.config.Attach = *config.Attach
	}
	if config.Tty != nil {
		*c.config.Tty = *config.Tty
	}
}

func (c *Container) ProcessState(id ID) *ProcessState {
	c.lock.Lock()
	defer c.lock.Unlock()
	if c.processState[id] != nil {
		copy := *c.processState[id]
		return &copy
	}
	return nil
}

// Not public as this should be part of commit
func (c *Container) pushFiles(f []FileToCopy) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.filesToCopy = append(c.filesToCopy, f)
}

// Not public as this should be part of commit
func (c *Container) popFiles() []FileToCopy {
	c.lock.Lock()
	defer c.lock.Unlock()
	copy := c.filesToCopy
	c.filesToCopy = nil
	return copy
}

func (c *Container) Opaque(key string) interface{} {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.opaque[key]
}

func (c *Container) SetOpaque(key string, value interface{}) {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.opaque[key] = value
}
