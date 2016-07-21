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
	"fmt"
	"sync"
	"time"
)

type Validator interface {
	checkPrerequisites(LifecycleStage, Handle, *Container) error
}

type defaultValidator struct {
}

func (*defaultValidator) checkPrerequisites(stage LifecycleStage, handle Handle, container *Container) error {
	return nil
}

type PostProcessor interface {
	postProcessHandle(LifecycleStage, Handle, error) (Handle, error)
}

type defaultPostProcessor struct {
}

func (*defaultPostProcessor) postProcessHandle(stage LifecycleStage, handle Handle, err error) (Handle, error) {
	return handle, err
}

type Platform interface {
	create(pendingCommit *PendingCommit) (*PendingCommit, error)
	modify(container *Container, pendingCommit *PendingCommit) (*PendingCommit, error)
	getChangeID(container *Container) int
	checkChangeID(container *Container, oldID int) bool
}

type ContainerEvent struct {
	eventTime   time.Time
	newRunState RunState
}

type ProcessEvent struct {
}

type LifecycleStage int

const (
	_ LifecycleStage = iota
	CreateContainer
	GetHandle
	Commit
	SetRunState
)

type BaseContainerLifecycle struct {
	stateManager  StateManager
	handles       HandleFactory
	validator     Validator
	postProcessor PostProcessor
	platform      Platform
	commitLock    sync.Mutex
}

func (p *BaseContainerLifecycle) newHandle(cid ID) *PendingCommit {
	return p.handles.CreateHandle(cid).(*PendingCommit)
}

func (p *BaseContainerLifecycle) Init(factory HandleFactory, platform Platform, stateManager StateManager) error {
	p.handles = factory
	p.platform = platform
	p.stateManager = stateManager
	p.validator = &defaultValidator{}         // avoid nil checks
	p.postProcessor = &defaultPostProcessor{} // avoid nil checks
	return nil
}

func (p *BaseContainerLifecycle) SetValidator(v Validator) {
	p.validator = v
}

func (p *BaseContainerLifecycle) SetPostProcessor(pp PostProcessor) {
	p.postProcessor = pp
}

func (p *BaseContainerLifecycle) CreateContainer(name string) (Handle, error) {
	if err := p.validator.checkPrerequisites(CreateContainer, nil, nil); err != nil {
		return nil, err
	}
	cid := GenerateID()
	handle := p.newHandle(cid)
	*handle.Config.Name = name
	handle.RunState = Created
	return p.postProcessor.postProcessHandle(CreateContainer, handle, nil)
}

func (p *BaseContainerLifecycle) GetHandle(cid ID) (Handle, error) {
	c := p.stateManager.getContainerForID(cid)
	if c == nil {
		return nil, fmt.Errorf("Invalid container ID")
	}
	if err := p.validator.checkPrerequisites(GetHandle, nil, c); err != nil {
		return nil, err
	}
	newHandle := p.newHandle(c.ContainerID)
	newHandle.ChangeID = p.platform.getChangeID(c)
	return p.postProcessor.postProcessHandle(GetHandle, newHandle, nil)
}

func (p *BaseContainerLifecycle) SetRunState(handle Handle, runState RunState) (Handle, error) {
	c := p.stateManager.getContainerForHandle(handle)
	if err := p.validator.checkPrerequisites(SetRunState, handle, c); err != nil {
		return nil, err
	}
	resolvedHandle := handle.(*PendingCommit)
	resolvedHandle.RunState = runState
	newHandle := p.handles.RefreshHandle(handle)
	return p.postProcessor.postProcessHandle(SetRunState, newHandle, nil)
}

func (p *BaseContainerLifecycle) Commit(handle Handle) (ID, error) {
	var err error
	c := p.stateManager.getContainerForHandle(handle)
	if err = p.validator.checkPrerequisites(Commit, handle, c); err != nil {
		return NilID, err
	}
	pc := handle.(*PendingCommit)
	if c == nil {
		pc, err = p.platform.create(pc)
		if err == nil {
			c = p.stateManager.createContainer(pc)
		}
	} else {
		p.commitLock.Lock() // Ensure only one commit per changeID
		defer p.commitLock.Unlock()
		if p.platform.checkChangeID(c, pc.ChangeID) {
			pc, err = p.platform.modify(c, pc)
			if err == nil {
				p.stateManager.updateContainer(pc)
			}
		} else {
			/* TODO: How can the caller detect this particular error case? */
			err = fmt.Errorf("Concurrent change error for container %v", c.ContainerID)
		}
	}
	if err != nil {
		return NilID, err
	}
	// Handle will be garbage collected
	return c.ContainerID, nil
}
