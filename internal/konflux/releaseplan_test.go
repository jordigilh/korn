package konflux_test

import (
	"github.com/jordigilh/korn/internal/konflux"
	"github.com/jordigilh/korn/testutils"
	releaseapiv1alpha1 "github.com/konflux-ci/release-service/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Test constants for release plans
const (
	testReleasePlanName       = "test-releaseplan"
	stagingReleasePlanName    = "staging-releaseplan"
	productionReleasePlanName = "production-releaseplan"
	otherAppReleasePlanName   = "other-app-releaseplan"
)

var _ = Describe("ReleasePlan functionality", func() {
	var (
		fakeClientBuilder *fake.ClientBuilder
		scheme            *runtime.Scheme
		ns                *corev1.Namespace
		kornInstance      *konflux.Korn
	)

	BeforeEach(func() {
		scheme = createFakeScheme()
		ns = newNamespace(testutils.TestNamespace)
		fakeClientBuilder = fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ns)

		kornInstance = &konflux.Korn{
			Namespace:       testutils.TestNamespace,
			ApplicationName: testutils.TestAppName,
			ReleasePlanName: testReleasePlanName,
		}
	})

	Context("ListReleasePlans functionality", func() {
		DescribeTable("should handle various release plan scenarios",
			func(applicationName string, releasePlans []runtime.Object, expectedCount int, expectError bool, description string) {
				kornInstance.ApplicationName = applicationName
				if len(releasePlans) > 0 {
					fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(releasePlans...)
				}
				kornInstance.KubeClient = fakeClientBuilder.Build()

				result, err := kornInstance.ListReleasePlans()

				if expectError {
					Expect(err).To(HaveOccurred(), description)
					Expect(result).To(BeNil(), description)
				} else {
					Expect(err).ToNot(HaveOccurred(), description)
					Expect(result).To(HaveLen(expectedCount), description)
				}
			},

			Entry("should return empty list when no release plans exist",
				testutils.TestAppName, []runtime.Object{}, 0, false,
				"Should handle empty release plan list"),

			Entry("should return all release plans when ApplicationName is empty",
				"", getSimpleReleasePlans(), 2, false,
				"Should return all release plans when no app filter"),

			Entry("should filter release plans by application name",
				testutils.TestAppName, getMixedAppReleasePlans(), 2, false,
				"Should filter by application name"),

			Entry("should return empty list when no release plans match application name",
				testutils.TestAppName, getOtherAppReleasePlans(), 0, false,
				"Should return empty list for non-matching application"),

			Entry("should handle multiple environments for same application",
				testutils.TestAppName, getMultiEnvironmentReleasePlans(), 2, false,
				"Should return multiple environments for same app"),
		)
	})

	Context("GetReleasePlan functionality", func() {
		DescribeTable("should handle release plan retrieval scenarios",
			func(releasePlans []runtime.Object, releasePlanName string, expectError bool, expectedErrorSubstring string, description string) {
				kornInstance.ReleasePlanName = releasePlanName
				if len(releasePlans) > 0 {
					fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(releasePlans...)
				}
				kornInstance.KubeClient = fakeClientBuilder.Build()

				result, err := kornInstance.GetReleasePlan()

				if expectError {
					Expect(err).To(HaveOccurred(), description)
					Expect(result).To(BeNil(), description)
					if expectedErrorSubstring != "" {
						Expect(err.Error()).To(ContainSubstring(expectedErrorSubstring), description)
					}
				} else {
					Expect(err).ToNot(HaveOccurred(), description)
					Expect(result).ToNot(BeNil(), description)
					Expect(result.Name).To(Equal(releasePlanName), description)
				}
			},

			Entry("should return release plan when it exists",
				getSingleTestReleasePlan(), testReleasePlanName, false, "",
				"Should get existing release plan"),

			Entry("should return custom error when release plan not found",
				[]runtime.Object{}, testReleasePlanName, true, "ReleasePlan test-releaseplan not found in namespace test-namespace",
				"Should fail for non-existing release plan"),

			Entry("should find staging release plan",
				getMultiEnvironmentReleasePlans(), stagingReleasePlanName, false, "",
				"Should get staging release plan"),

			Entry("should find production release plan",
				getMultiEnvironmentReleasePlans(), productionReleasePlanName, false, "",
				"Should get production release plan"),
		)
	})

})

// Helper functions to create release plan objects
func newReleasePlan(name, namespace, application string, labels map[string]string) *releaseapiv1alpha1.ReleasePlan {
	return &releaseapiv1alpha1.ReleasePlan{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: releaseapiv1alpha1.ReleasePlanSpec{Application: application},
	}
}

func newStagingReleasePlan(name, namespace, application string) *releaseapiv1alpha1.ReleasePlan {
	return newReleasePlan(name, namespace, application, map[string]string{
		konflux.EnvironmentLabel: "staging",
	})
}

func newProductionReleasePlan(name, namespace, application string) *releaseapiv1alpha1.ReleasePlan {
	return newReleasePlan(name, namespace, application, map[string]string{
		konflux.EnvironmentLabel: "production",
	})
}

// Helper functions to get sets of release plans for different test scenarios
func getSimpleReleasePlans() []runtime.Object {
	return []runtime.Object{
		newReleasePlan("releaseplan1", testutils.TestNamespace, "app1", nil),
		newReleasePlan("releaseplan2", testutils.TestNamespace, "app2", nil),
	}
}

func getMixedAppReleasePlans() []runtime.Object {
	return []runtime.Object{
		newStagingReleasePlan(stagingReleasePlanName, testutils.TestNamespace, testutils.TestAppName),
		newProductionReleasePlan(productionReleasePlanName, testutils.TestNamespace, testutils.TestAppName),
		newStagingReleasePlan(otherAppReleasePlanName, testutils.TestNamespace, testutils.OtherAppName),
	}
}

func getOtherAppReleasePlans() []runtime.Object {
	return []runtime.Object{
		newStagingReleasePlan(otherAppReleasePlanName, testutils.TestNamespace, testutils.OtherAppName),
	}
}

func getSingleTestReleasePlan() []runtime.Object {
	return []runtime.Object{
		newStagingReleasePlan(testReleasePlanName, testutils.TestNamespace, testutils.TestAppName),
	}
}

func getMultiEnvironmentReleasePlans() []runtime.Object {
	return []runtime.Object{
		newStagingReleasePlan(stagingReleasePlanName, testutils.TestNamespace, testutils.TestAppName),
		newProductionReleasePlan(productionReleasePlanName, testutils.TestNamespace, testutils.TestAppName),
	}
}
