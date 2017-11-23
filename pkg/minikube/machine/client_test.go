/*
Copyright 2016 The Kubernetes Authors All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package machine

import (
	"bufio"
	"io/ioutil"
	"log"
	"net"
	"os"
	"testing"

	"github.com/docker/machine/libmachine/drivers/plugin/localbinary"
	"github.com/minishift/minishift/pkg/minikube/constants"
	"github.com/stretchr/testify/assert"
)

func makeTempDir() string {
	tempDir, err := ioutil.TempDir("", "minipath")
	if err != nil {
		log.Fatal(err)
	}
	constants.Minipath = tempDir
	return tempDir
}

func TestRunNotDriver(t *testing.T) {
	tempDir := makeTempDir()
	defer os.RemoveAll(tempDir)
	StartDriver()
	assert.True(t, localbinary.CurrentBinaryIsDockerMachine, "CurrentBinaryIsDockerMachine Not set This will prevent driver initialization")
}

func TestRunDriver(t *testing.T) {
	// This test is a bit complicated. It verifies that when the root command is
	// called with the proper environment variables, we setup the libmachine driver.

	tempDir := makeTempDir()
	defer os.RemoveAll(tempDir)
	os.Setenv(localbinary.PluginEnvKey, localbinary.PluginEnvVal)
	os.Setenv(localbinary.PluginEnvDriverName, "virtualbox")

	// Capture stdout and reset it later.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	// Run the command asynchronously. It should listen on a port for connections.
	go StartDriver()

	// The command will write out what port it's listening on over stdout.
	reader := bufio.NewReader(r)
	addr, _, err := reader.ReadLine()
	assert.NoError(t, err, "Cannot read address in the standard output")
	os.Stdout = old

	// Now that we got the port, make sure we can connect.
	_, err = net.Dial("tcp", string(addr))
	assert.NoError(t, err, "Driver is not listening")
}
