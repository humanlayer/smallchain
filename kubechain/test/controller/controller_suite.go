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

package controller

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"
)

// SharedTestEnv provides a shared test environment for all controller tests
var SharedTestEnv *TestEnv

// SetupEnvTest sets up the shared test environment
func SetupEnvTest() (*TestEnv, error) {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	// Use absolute path for binaries
	testEnv := &envtest.Environment{
		CRDDirectoryPaths:     []string{"/Users/nyx/dev/humanlayer/smallchain/kubechain/config/crd/bases"},
		ErrorIfCRDPathMissing: true,
		// Use absolute paths for binary assets
		BinaryAssetsDirectory: "/Users/nyx/dev/humanlayer/smallchain/kubechain/bin/k8s/1.32.0-darwin-arm64",
	}

	cfg, err := testEnv.Start()
	if err != nil {
		return nil, err
	}

	err = kubechainv1alpha1.AddToScheme(scheme.Scheme)
	if err != nil {
		return nil, err
	}

	k8sClient, err := client.New(cfg, client.Options{Scheme: scheme.Scheme})
	if err != nil {
		return nil, err
	}

	return &TestEnv{
		Client:    k8sClient,
		Env:       testEnv,
		Config:    cfg,
		Ctx:       nil, // Will be set in BeforeSuite
		Namespace: "default",
	}, nil
}

// SetupSuite sets up a common test suite configuration for controller tests
func SetupSuite(t *testing.T, description string) {
	RegisterFailHandler(Fail)
	RunSpecs(t, description)
}