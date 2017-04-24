// Copyright (C) 2015 NTT Innovation Institute, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package framework

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/cloudwan/gohan/util"
	. "github.com/onsi/gomega"
)

func TestGetTestFiles(t *testing.T) {
	RegisterTestingT(t)

	rootDir, err := ioutil.TempDir(os.TempDir(), fmt.Sprint(time.Now().UnixNano()))
	Expect(err).ToNot(HaveOccurred())
	defer os.RemoveAll(rootDir)

	leftDir, err := ioutil.TempDir(rootDir, fmt.Sprint(time.Now().UnixNano()))
	Expect(err).ToNot(HaveOccurred())

	rightDir, err := ioutil.TempDir(rootDir, fmt.Sprint(time.Now().UnixNano()))
	Expect(err).ToNot(HaveOccurred())

	deepDir, err := ioutil.TempDir(leftDir, fmt.Sprint(time.Now().UnixNano()))
	Expect(err).ToNot(HaveOccurred())

	testFile1, err := util.TempFile(rootDir, "test_", ".js")
	Expect(err).ToNot(HaveOccurred())
	defer testFile1.Close()

	testFile2, err := util.TempFile(rightDir, "test_", ".js")
	Expect(err).ToNot(HaveOccurred())
	defer testFile2.Close()

	testFile3, err := util.TempFile(deepDir, "test_", ".js")
	Expect(err).ToNot(HaveOccurred())
	defer testFile3.Close()

	irrelevantFile, err := util.TempFile(leftDir, "irrelevant", ".js")
	Expect(err).ToNot(HaveOccurred())
	defer irrelevantFile.Close()

	args := []string{rootDir}
	tests := getTestFiles(args, "js")

	Expect(tests).To(ConsistOf(testFile1.Name(), testFile2.Name(), testFile3.Name()))
}
