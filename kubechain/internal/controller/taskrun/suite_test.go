/*
Copyright 2025.

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

package taskrun

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	testutil "github.com/humanlayer/smallchain/kubechain/test/controller"
)

var (
	testEnv *testutil.TestEnv
)

func TestControllers(t *testing.T) {
	testutil.SetupSuite(t, "TaskRun Controller Suite")
}

var _ = BeforeSuite(func() {
	var err error
	testEnv, err = testutil.SetupEnvTest()
	Expect(err).NotTo(HaveOccurred())
	Expect(testEnv).NotTo(BeNil())

	ctx, cancel := context.WithCancel(context.TODO())
	testEnv.Ctx = ctx
	testEnv.Cancel = cancel
	testEnv.Recorder = testEnv.NewFakeRecorder(10)
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	if testEnv != nil {
		testEnv.Stop()
	}
})