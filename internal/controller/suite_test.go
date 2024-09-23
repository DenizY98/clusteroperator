/*
Copyright 2024.

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
<<<<<<< HEAD
	"path/filepath"
=======
	"context"
	"fmt"
	"path/filepath"
	"runtime"
>>>>>>> 59ff505 (fix(iter1): added ctrl, crd and sample)
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

<<<<<<< HEAD
	clusteropv1alpha1 "github.com/DenizY98/clusteroperator.git/api/v1alpha1"
	//+kubebuilder:scaffold:imports
=======
	clustergroupv1 "github.com/DenizY98/clusteroperator/api/v1"
	// +kubebuilder:scaffold:imports
>>>>>>> 59ff505 (fix(iter1): added ctrl, crd and sample)
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment
<<<<<<< HEAD
=======
var ctx context.Context
var cancel context.CancelFunc
>>>>>>> 59ff505 (fix(iter1): added ctrl, crd and sample)

func TestControllers(t *testing.T) {
	RegisterFailHandler(Fail)

	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

<<<<<<< HEAD
=======
	ctx, cancel = context.WithCancel(context.TODO())

>>>>>>> 59ff505 (fix(iter1): added ctrl, crd and sample)
	By("bootstrapping test environment")
	testEnv = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
<<<<<<< HEAD
=======

		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.31.0-%s-%s", runtime.GOOS, runtime.GOARCH)),
>>>>>>> 59ff505 (fix(iter1): added ctrl, crd and sample)
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

<<<<<<< HEAD
	err = clusteropv1alpha1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	//+kubebuilder:scaffold:scheme
=======
	err = clustergroupv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme
>>>>>>> 59ff505 (fix(iter1): added ctrl, crd and sample)

	k8sClient, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())

})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
<<<<<<< HEAD
=======
	cancel()
>>>>>>> 59ff505 (fix(iter1): added ctrl, crd and sample)
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
