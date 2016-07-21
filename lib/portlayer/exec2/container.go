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
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
)

type ID uuid.UUID

var NilID = ID(uuid.Nil)

func GenerateID() ID {
	return ID(uuid.New())
}

func ParseID(idStr string) (ID, error) {
	result, err := uuid.Parse(idStr)
	return ID(result), err
}

func (id ID) String() string {
	return uuid.UUID(id).String()
}

// Struct that represents the internal port-layer representation of a container
// All data in this struct must be data that is either immutable
// or can be relied upon without having to query either the container guest
// or the underlying infrastructure. Some of this state will be updated by events
// Immutable state is public and can be accessed directly. Mutable state is private
type Container struct {
	ConstantConfig
	config Config

	runState     RunState
	processes    map[ID]*ProcessConfig
	processState map[ID]*ProcessState

	filesToCopy []FileToCopy // cache if copy while not running

	opaque map[string]interface{}

	lock sync.Mutex // lock rw updates to the mutable state
}

// config that will be applied to a container on commit
// All state is public as it will only ever be used/modified by a single thread
type PendingCommit struct {
	ConstantConfig

	ChangeID    int
	RunState    RunState
	Config      Config
	MainProcess ProcessConfig
	FilesToCopy []FileToCopy

	Opaque map[string]interface{}
}

// config state that cannot change for the lifetime of the container
type ConstantConfig struct {
	ContainerID ID
	Created     time.Time
}

// variable container configuration state
type Config struct {
	Name   *string
	Limits *ResourceLimits
	Attach *bool
	Tty    *bool
}

// configuration state of a container process
type ProcessConfig struct {
	ProcessID ID
	WorkDir   string
	ExecPath  string
	ExecArgs  []string
	Env       []string
}

func NewProcessConfig(workDir string, execPath string, execArgs []string, env []string) ProcessConfig {
	return ProcessConfig{ProcessID: GenerateID(), WorkDir: workDir, ExecPath: execPath, ExecArgs: execArgs, Env: env}
}

type ProcessStatus int

const (
	_ ProcessStatus = iota
	Started
	Exited
)

type ProcessState struct {
	ProcessID ID
	GuestPid  int
	StartedAt time.Time
	ProcessRunState
}

// runtime status of a container process
type ProcessRunState struct {
	Status     *ProcessStatus
	ExitCode   *int
	ExitMsg    *string
	FinishedAt *time.Time
}

type FileToCopy struct {
	Target url.URL
	Perms  int16
	Data   []byte
}

type ResourceLimits struct {
	MemoryMb *int
	CPUMhz   *int
}
