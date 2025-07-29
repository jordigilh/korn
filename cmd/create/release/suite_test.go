// NOTE: This file contains AI-generated test setup and patterns (Cursor)
// All test logic has been reviewed and validated for correctness

package release_test

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	releaseapiv1alpha1 "github.com/konflux-ci/release-service/api/v1alpha1"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	kruntime "k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	"k8s.io/kubectl/pkg/scheme"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var testEnv *envtest.Environment

func TestCreateRelease(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Create Release Command Suite")
}

var _ = BeforeSuite(func() {

	By("bootstrapping test environment")
	k8sassets, ok := os.LookupEnv("KUBEBUILDER_ASSETS")
	if !ok {
		logrus.Warnln("Missing environment variable KUBEBUILDER_ASSETS, using K8S version 1.32.0")
		k8sassets = filepath.Join("..", "..", "..", "bin", "k8s",
			fmt.Sprintf("1.32.0-%s-%s", runtime.GOOS, runtime.GOARCH))
	}
	testEnv = &envtest.Environment{
		ErrorIfCRDPathMissing: true,
		// The BinaryAssetsDirectory is only required if you want to run the tests directly
		// without call the makefile target test. If not informed it will look for the
		// default path defined in controller-runtime which is /usr/local/kubebuilder/.
		// Note that you must have the required binaries setup under the bin directory to perform
		// the tests directly. When we run make test it will be setup and used automatically.
		BinaryAssetsDirectory: k8sassets,
	}

	var err error
	// cfg is defined in this file globally.
	cfg, err = testEnv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())
})

var _ = AfterSuite(func() {
	By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func createFakeScheme() *kruntime.Scheme {
	s := scheme.Scheme
	builder := append(kruntime.SchemeBuilder{},
		corev1.AddToScheme,
		applicationapiv1alpha1.AddToScheme,
		releaseapiv1alpha1.AddToScheme,
	)
	Expect(builder.AddToScheme(s)).To(Succeed())
	return s
}
