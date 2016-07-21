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

package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/vmware/vic/lib/portlayer/exec2"
	"github.com/vmware/vic/lib/portlayer/exec2/remote"
)

var portLayer exec2.ContainerLifecycle
var plquery exec2.ContainerQuery

type ErrReturnCode int

const (
	_ ErrReturnCode = iota
	NoOp
	IDNoExist
	InvalidState
	Timeout
	HostUnreachable
	NoResource
	Failed
)

func main() {
	pl := &remote.PortLayerRPCClient{}
	err := pl.Connect()
	if err != nil {
		os.Exit(int(HostUnreachable))
	}
	portLayer = pl
	if len(os.Args) < 2 {
		usage()
		return
	}
	command := os.Args[1]
	bio := bufio.NewReader(os.Stdin)

	var exitCode ErrReturnCode
	switch command {
	case "container":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "tce container usage:\n   command needs a path to an executable\n")
		} else {
			exitCode = createContainer(os.Args[2])
		}
	case "start":
		cidStr, _, _ := bio.ReadLine()
		var args string
		if len(os.Args) > 2 {
			args = os.Args[2]
		}
		cid, err := exec2.ParseID(string(cidStr))
		if err != nil {
			fmt.Println(err)
		}
		exitCode = startContainer(cid, args)
	default:
		usage()
	}
	if exitCode > 0 {
		os.Exit(int(exitCode))
	}
}

func createContainer(execPath string) ErrReturnCode {
	exec, err := ioutil.ReadFile(execPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return NoOp
	}
	//	fmt.Printf("Creating container for %q...\n", execPath)
	handle, err := portLayer.CreateContainer("")
	if err != nil {
		fmt.Printf("%v\n", err)
		return Failed // TODO: or NoResource
	}
	fName := filepath.Base(execPath)
	handle, _ = portLayer.CopyTo(handle, "/", fName, 0777, exec)
	handle, _ = portLayer.SetEntryPoint(handle, "/", fName, "")
	handle, _ = portLayer.SetLimits(handle, 1024, -1)
	cid, _ := portLayer.Commit(handle)
	fmt.Fprintf(os.Stdout, "%s\n", cid)
	return 0
}

func startContainer(cid exec2.ID, args string) ErrReturnCode {
	handle, _ := portLayer.GetHandle(cid)
	if handle == nil {
		return IDNoExist
	}
	handle, _ = portLayer.SetRunState(handle, exec2.Running)

	// To set just the args, we need to get the workdDir and path from the create step
	//	cState, _ := plquery.GetState(cid)
	//	pConfig := cState.process
	//	handle, _ = portLayer.SetEntryPoint(handle, cState.WorkDir, pConfig.path, args)
	cid, _ = portLayer.Commit(handle)
	fmt.Fprintf(os.Stdout, "%s\n", cid)
	return 0
}

func usage() {
	fmt.Fprintf(os.Stderr, "\nTrivial Container Engine Usage:\n")
	fmt.Fprintf(os.Stderr, "   tce <command> <options>\n")
	fmt.Fprintf(os.Stderr, "       container <executable>: create a container\n")
	fmt.Fprintf(os.Stderr, "       start <executable args>: start a container with optional args\n")
}
