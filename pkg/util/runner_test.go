/*
Copyright (C) 2016 Red Hat, Inc.

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

package util

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func Test_Runner_Run(t *testing.T) {
	var runner Runner = &RealRunner{}
	var stdOut, stdErr io.Writer

	currentDir, _ := os.Getwd()
	dummybinary := filepath.Join(currentDir, "..", "..", "test", "testdata", "dummybinary")

	if runtime.GOOS == "windows" {
		dummybinary = filepath.Join(currentDir, "..", "..", "test", "testdata", "dummybinary_windows.exe")
	}
	if runtime.GOOS == "darwin" {
		dummybinary = filepath.Join(currentDir, "..", "..", "test", "testdata", "dummybinary_darwin")
	}

	expected := 0
	actual := runner.Run(stdOut, stdErr, dummybinary)

	if expected != actual {
		t.Fatalf("Expected %d Got %d", expected, actual)
	}
}
