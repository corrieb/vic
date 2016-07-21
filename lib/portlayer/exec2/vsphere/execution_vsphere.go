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

package vsphere

import (
	"fmt"
	"net/url"

	"github.com/vmware/vic/lib/portlayer/exec2"
	"github.com/vmware/vic/pkg/vsphere/vm"
)

type ContainerVM struct {
	exec2.Container
	vm vm.VirtualMachine
}

type PendingCommitVM struct {
	exec2.PendingCommit
}

type CvmHandleFactory struct {
}

func (h *CvmHandleFactory) CreateHandle(cid exec2.ID) exec2.Handle {
	newPc := &PendingCommitVM{}
	newPc.ContainerID = cid
	return newPc
}

// Basic handle resolver just passes back the handle passed in
func (h *CvmHandleFactory) RefreshHandle(oldHandle exec2.Handle) exec2.Handle {
	return oldHandle
}

// PortLayerVsphere is a WIP implementation of the execution.go interfaces
type PortLayerVsphere struct {
	vmomiGateway VmomiGateway
	handles      exec2.HandleFactory
	containers   map[exec2.ID]*ContainerVM
}

func (p *PortLayerVsphere) getContainer(handle exec2.Handle) *ContainerVM {
	return p.containers[handle.(*PendingCommitVM).ContainerID]
}

func (p *PortLayerVsphere) newHandle(cid exec2.ID) *PendingCommitVM {
	return p.handles.CreateHandle(cid).(*PendingCommitVM)
}

func (p *PortLayerVsphere) Init(gateway VmomiGateway, factory exec2.HandleFactory) error {
	p.handles = factory
	p.vmomiGateway = gateway
	p.containers = make(map[exec2.ID]*ContainerVM)
	return nil
}

func (p *PortLayerVsphere) CreateContainer(name string) (exec2.Handle, error) {
	cid := exec2.GenerateID()
	handle := p.newHandle(cid)
	handle.Config.Name = name
	handle.RunState = exec2.Created
	return handle, nil
}

func (p *PortLayerVsphere) GetHandle(cid exec2.ID) (exec2.Handle, error) {
	c := p.containers[cid]
	if c == nil {
		return nil, fmt.Errorf("Invalid container ID")
	}
	return p.handles.CreateHandle(c.ContainerID), nil
}

func (p *PortLayerVsphere) SetEntryPoint(handle exec2.Handle, workDir string, execPath string, execArgs []string, env []string) (exec2.Handle, error) {
	resolvedHandle := handle.(*PendingCommitVM)
	resolvedHandle.MainProcess = exec2.NewProcessConfig(workDir, execPath, execArgs, env)
	return p.handles.RefreshHandle(handle), nil
}

func (p *PortLayerVsphere) SetInteraction(handle exec2.Handle, attach bool, tty bool) (exec2.Handle, error) {
	resolvedHandle := handle.(*PendingCommitVM)
	resolvedHandle.Config.Attach = attach
	resolvedHandle.Config.Tty = tty
	return p.handles.RefreshHandle(handle), nil
}

func (p *PortLayerVsphere) Commit(handle exec2.Handle) (exec2.ID, error) {
	var err error
	c := p.getContainer(handle)
	if c == nil {
		c, err = p.createContainer(handle)
	} else {
		//		if c.vm == nil {
		//			return "", fmt.Errorf("Cannot modify container with no VM")
		//		}
		err = p.modifyContainer(c.RunState, handle)
	}
	// Handle will be garbage collected
	return c.ContainerID, err
}

func (p *PortLayerVsphere) CopyTo(handle exec2.Handle, targetDir string, fname string, perms int16, data []byte) (exec2.Handle, error) {
	var result exec2.Handle
	resolvedHandle := handle.(*PendingCommitVM)
	u, err := url.Parse("file://" + targetDir + "/" + fname)
	if err == nil {
		fileToCopy := exec2.FileToCopy{Target: *u, Perms: perms, Data: data}
		resolvedHandle.FilesToCopy = append(resolvedHandle.FilesToCopy, fileToCopy)
		result = p.handles.RefreshHandle(handle)
	}
	return result, err
}

func (p *PortLayerVsphere) SetLimits(handle exec2.Handle, memoryMb int, cpuMhz int) (exec2.Handle, error) {
	resolvedHandle := handle.(*PendingCommitVM)
	resolvedHandle.Config.Limits = exec2.ResourceLimits{MemoryMb: memoryMb, CPUMhz: cpuMhz}
	return p.handles.RefreshHandle(handle), nil
}

func (p *PortLayerVsphere) SetRunState(handle exec2.Handle, runState exec2.RunState) (exec2.Handle, error) {
	resolvedHandle := handle.(*PendingCommitVM)
	resolvedHandle.RunState = runState
	return p.handles.RefreshHandle(handle), nil
}

func (p *PortLayerVsphere) DestroyContainer(cid exec2.ID) error {
	c := p.containers[cid]
	if c == nil {
		return fmt.Errorf("Invalid container ID")
	}
	delete(p.containers, cid)
	return nil
}

func (p *PortLayerVsphere) createContainer(handle exec2.Handle) (*ContainerVM, error) {
	resolvedHandle := handle.(*PendingCommitVM)
	c := ContainerVM{}
	p.containers[resolvedHandle.ContainerID] = &c
	c.ContainerID = resolvedHandle.ContainerID
	c.RunState = resolvedHandle.RunState
	// followed by other transfer of state from pending to container
	//	fmt.Printf("Creating container for %v\n", pending)
	return &c, nil
}

func (p *PortLayerVsphere) modifyContainer(runState exec2.RunState, handle exec2.Handle) error {
	// fmt.Printf("Modifying container for %v\n", pending)
	return nil
}
