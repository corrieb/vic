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
	"github.com/google/uuid"
)

/* A Handle should be completely opaque */
type Handle interface{}

// A Handle can be anything, so this takes advantage of that by creating a sparse handle
// to send to the client and using that sparse handle as a key in a hashtable which points to
// rich handles created by the HandleFactory
type SparseHandle Handle

type HandleFactory interface {
	CreateHandle(cID ID) Handle
	RefreshHandle(oldHandle Handle) Handle
}

type SparseHandleFactory interface {
	CreateSparseHandle(handle Handle) SparseHandle
	ResolveSparseHandle(handle SparseHandle) Handle
	RefreshSparseHandle(oldHandle SparseHandle, newHandle Handle) SparseHandle
}

type BasicHandleFactory struct {
}

func (h *BasicHandleFactory) CreateHandle(cid ID) Handle {
	newPc := &PendingCommit{}
	newPc.ContainerID = cid
	return newPc
}

// Basic handle resolver just passes back the handle passed in
func (h *BasicHandleFactory) RefreshHandle(oldHandle Handle) Handle {
	return oldHandle
}

// A sparse handle is simply a unique string
type StringSparseHandleFactory struct {
	handles map[SparseHandle]Handle
}

func NewSparseHandleFactory() SparseHandleFactory {
	shf := &StringSparseHandleFactory{}
	shf.handles = make(map[SparseHandle]Handle)
	return shf
}

func (h *StringSparseHandleFactory) newSparseHandle() SparseHandle {
	return SparseHandle(uuid.New())
}

func (h *StringSparseHandleFactory) CreateSparseHandle(handle Handle) SparseHandle {
	key := h.newSparseHandle()
	h.handles[key] = handle
	return key
}

func (h *StringSparseHandleFactory) ResolveSparseHandle(handle SparseHandle) Handle {
	return h.handles[handle]
}

func (h *StringSparseHandleFactory) RefreshSparseHandle(oldHandle SparseHandle, newHandle Handle) SparseHandle {
	result := h.CreateSparseHandle(newHandle)
	delete(h.handles, oldHandle)
	return result
}
