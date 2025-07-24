package konflux_test

import (
	"github.com/jordigilh/korn/internal/konflux"
	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = When("retrieving applications", func() {

	var (
		fakeClientBuilder *fake.ClientBuilder
		scheme            = createFakeScheme()
		ns                = newNamespace("default")
		korn              = konflux.Korn{}
	)

	BeforeEach(func() {
		fakeClientBuilder = fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ns)
	})
	DescribeTable("it correctly identifies all applications as expected", func(hasError bool, appName string, objs []runtime.Object) {

		if len(objs) > 0 {
			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(objs...)
			korn.ApplicationName = appName
		}
		korn.KubeClient = fakeClientBuilder.Build()
		korn.Namespace = "default"
		Expect(k8sClient).To(BeNil())
		apps, err := korn.GetApplication()
		if hasError {
			Expect(err).To(HaveOccurred())
		} else {
			Expect(err).NotTo(HaveOccurred())
			Expect(apps).NotTo(BeNil())
			Expect(apps.Name).To(Equal(appName))
		}

	},
		Entry("retrieves no application when none are available in the given namespace", true, "foo", nil),
		Entry("retrieves one application in the given namespace", false, "foo", []runtime.Object{
			newApp("foo", "default"),
		}),
		Entry("retrieves one application in the given namespace", true, "foo", []runtime.Object{
			newApp("bar", "default"),
		}),
		Entry("retrieves one application in the given namespace", true, "foo", []runtime.Object{
			newApp("foo", "new-namespace"),
		}),
	)
})

func newNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
}

func newApp(name, namespace string) *applicationapiv1alpha1.Application {
	return &applicationapiv1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}
