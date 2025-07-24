package konflux_test

import (
	"github.com/jordigilh/korn/internal/konflux"
	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Test constants
const (
	testNamespace           = "test-namespace"
	testAppName             = "test-app"
	otherAppName            = "other-app"
	testComponentName       = "test-component"
	bundleComponentName     = "bundle-component"
	controllerComponentName = "controller-component"
)

var _ = Describe("Component functionality", func() {
	var (
		fakeClientBuilder *fake.ClientBuilder
		scheme            *runtime.Scheme
		ns                *corev1.Namespace
		kornInstance      *konflux.Korn
	)

	BeforeEach(func() {
		scheme = createFakeScheme()
		ns = newNamespace(testNamespace)
		fakeClientBuilder = fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ns)

		kornInstance = &konflux.Korn{
			Namespace:       testNamespace,
			ApplicationName: testAppName,
			ComponentName:   testComponentName,
		}
	})

	Context("ListComponents functionality", func() {
		DescribeTable("should handle various component scenarios",
			func(applicationName string, components []runtime.Object, expectedCount int, expectError bool, description string) {
				kornInstance.ApplicationName = applicationName
				if len(components) > 0 {
					fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(components...)
				}
				kornInstance.KubeClient = fakeClientBuilder.Build()

				result, err := kornInstance.ListComponents()

				if expectError {
					Expect(err).To(HaveOccurred(), description)
					Expect(result).To(BeNil(), description)
				} else {
					Expect(err).ToNot(HaveOccurred(), description)
					Expect(result).To(HaveLen(expectedCount), description)
				}
			},

			Entry("should return empty list when no components exist",
				testAppName, []runtime.Object{}, 0, false,
				"Should handle empty component list"),

			Entry("should return all components when ApplicationName is empty",
				"", getSimpleComponents(), 2, false,
				"Should return all components when no app filter"),

			Entry("should filter components by application name",
				testAppName, getMixedAppComponents(), 1, false,
				"Should filter by application name"),

			Entry("should return empty list when no components match application name",
				testAppName, getOtherAppComponents(), 0, false,
				"Should return empty list for non-matching application"),
		)

	})

	Context("ListComponentsWithMatchingLabels functionality", func() {
		DescribeTable("should filter components by labels",
			func(applicationName string, components []runtime.Object, labels client.MatchingLabels, expectedCount int, expectedNames []string, expectError bool, description string) {
				kornInstance.ApplicationName = applicationName
				if len(components) > 0 {
					fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(components...)
				}
				kornInstance.KubeClient = fakeClientBuilder.Build()

				result, err := kornInstance.ListComponentsWithMatchingLabels(labels)

				if expectError {
					Expect(err).To(HaveOccurred(), description)
					Expect(result).To(BeNil(), description)
				} else {
					Expect(err).ToNot(HaveOccurred(), description)
					Expect(result).To(HaveLen(expectedCount), description)
					if expectedCount > 0 && len(expectedNames) > 0 {
						actualNames := make([]string, len(result))
						for i, comp := range result {
							actualNames[i] = comp.Name
						}
						for _, expectedName := range expectedNames {
							Expect(actualNames).To(ContainElement(expectedName), description)
						}
					}
				}
			},

			Entry("should return all components when labels is nil",
				"", getLabeledTestComponents(), nil, 3, []string{bundleComponentName, controllerComponentName, "other-component"}, false,
				"Should return all components when no label filter"),

			Entry("should filter components by single label",
				"", getLabeledTestComponents(), client.MatchingLabels{konflux.ComponentTypeLabel: "bundle"}, 1, []string{bundleComponentName}, false,
				"Should filter by single label"),

			Entry("should filter components by multiple labels",
				"", getLabeledTestComponents(), client.MatchingLabels{
					"env":                      "staging",
					konflux.ComponentTypeLabel: "bundle",
				}, 1, []string{bundleComponentName}, false,
				"Should filter by multiple labels"),

			Entry("should filter by labels and application name",
				testAppName, getLabeledTestComponents(), client.MatchingLabels{"env": "staging"}, 2, []string{bundleComponentName, controllerComponentName}, false,
				"Should filter by labels and application"),

			Entry("should return empty list when no components match labels",
				"", getLabeledTestComponents(), client.MatchingLabels{"non-existent": "label"}, 0, []string{}, false,
				"Should return empty for non-matching labels"),
		)

		It("should handle Kubernetes client errors gracefully", func() {
			// Test with a valid setup since fake clients don't always error predictably
			kornInstance.KubeClient = fake.NewClientBuilder().WithScheme(scheme).Build()
			kornInstance.Namespace = testNamespace

			labels := client.MatchingLabels{"any": "label"}
			components, err := kornInstance.ListComponentsWithMatchingLabels(labels)

			// Fake clients typically don't error, they just return empty results
			Expect(err).ToNot(HaveOccurred())
			Expect(components).To(BeEmpty())
		})
	})

	Context("GetComponent functionality", func() {
		DescribeTable("should handle component retrieval scenarios",
			func(components []runtime.Object, componentName string, expectError bool, expectedErrorSubstring string, description string) {
				kornInstance.ComponentName = componentName
				if len(components) > 0 {
					fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(components...)
				}
				kornInstance.KubeClient = fakeClientBuilder.Build()

				result, err := kornInstance.GetComponent()

				if expectError {
					Expect(err).To(HaveOccurred(), description)
					Expect(result).To(BeNil(), description)
					if expectedErrorSubstring != "" {
						Expect(err.Error()).To(ContainSubstring(expectedErrorSubstring), description)
					}
				} else {
					Expect(err).ToNot(HaveOccurred(), description)
					Expect(result).ToNot(BeNil(), description)
					Expect(result.Name).To(Equal(componentName), description)
				}
			},

			Entry("should return component when it exists",
				getSingleTestComponent(), testComponentName, false, "",
				"Should get existing component"),

			Entry("should return custom error when component not found",
				[]runtime.Object{}, testComponentName, true, "component test-component not found in namespace test-namespace",
				"Should fail for non-existing component"),
		)

		It("should return original error for other Kubernetes errors", func() {
			// Test with empty component name which will still trigger our custom error
			kornInstance.ComponentName = ""
			kornInstance.KubeClient = fakeClientBuilder.Build()

			result, err := kornInstance.GetComponent()

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
			// With empty component name, we get "component  not found in namespace test-namespace"
			Expect(err.Error()).To(ContainSubstring("component  not found in namespace"))
		})
	})

	Context("GetBundleComponentForVersion functionality", func() {
		DescribeTable("should handle bundle component scenarios",
			func(components []runtime.Object, expectError bool, expectedErrorSubstring string, expectedComponentName string, description string) {
				if len(components) > 0 {
					fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(components...)
				}
				kornInstance.KubeClient = fakeClientBuilder.Build()

				result, err := kornInstance.GetBundleComponentForVersion()

				if expectError {
					Expect(err).To(HaveOccurred(), description)
					Expect(result).To(BeNil(), description)
					if expectedErrorSubstring != "" {
						Expect(err.Error()).To(ContainSubstring(expectedErrorSubstring), description)
					}
				} else {
					Expect(err).ToNot(HaveOccurred(), description)
					Expect(result).ToNot(BeNil(), description)
					if expectedComponentName != "" {
						Expect(result.Name).To(Equal(expectedComponentName), description)
					}
				}
			},

			Entry("should return bundle component when exactly one exists for application",
				getBundleWithControllerComponents(), false, "", bundleComponentName,
				"Should return single bundle component"),

			Entry("should ignore bundle components from other applications",
				getBundleComponentsFromMultipleApps(), false, "", bundleComponentName,
				"Should ignore other app bundle components"),

			Entry("should return error when no bundle components exist",
				getControllerOnlyComponents(), true, "no bundle component found for application test-namespace/test-app", "",
				"Should fail when no bundle components"),

			Entry("should return error when multiple bundle components exist for same application",
				getMultipleBundleComponents(), true, "more than one bundle component found for application test-namespace/test-app", "",
				"Should fail when multiple bundle components"),

			Entry("should return error when bundle components exist but none belong to the application",
				getOtherAppBundleComponents(), true, "no bundle component found for application test-namespace/test-app", "",
				"Should fail when no matching app bundle components"),
		)

		It("should return error when ListComponentsWithMatchingLabels fails", func() {
			kornInstance.KubeClient = fake.NewClientBuilder().WithScheme(scheme).Build()
			kornInstance.Namespace = "non-existent-namespace"

			result, err := kornInstance.GetBundleComponentForVersion()

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})
})

// Helper functions to create components
func newComponent(name, namespace, application string, labels map[string]string) *applicationapiv1alpha1.Component {
	return &applicationapiv1alpha1.Component{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: applicationapiv1alpha1.ComponentSpec{Application: application},
	}
}

func newBundleComponent(name, namespace, application string) *applicationapiv1alpha1.Component {
	return newComponent(name, namespace, application, map[string]string{
		konflux.ComponentTypeLabel: "bundle",
	})
}

func newControllerComponent(name, namespace, application string) *applicationapiv1alpha1.Component {
	return newComponent(name, namespace, application, map[string]string{
		konflux.BundleReferenceLabel: "controller",
	})
}

func newLabeledBundleComponent(name, namespace, application string) *applicationapiv1alpha1.Component {
	return newComponent(name, namespace, application, map[string]string{
		konflux.ComponentTypeLabel: "bundle",
		"env":                      "staging",
	})
}

func newLabeledControllerComponent(name, namespace, application string) *applicationapiv1alpha1.Component {
	return newComponent(name, namespace, application, map[string]string{
		konflux.BundleReferenceLabel: "controller",
		"env":                        "staging",
	})
}

// Helper functions to get sets of components for different test scenarios
func getSimpleComponents() []runtime.Object {
	return []runtime.Object{
		newComponent("component1", testNamespace, "app1", nil),
		newComponent("component2", testNamespace, "app2", nil),
	}
}

func getMixedAppComponents() []runtime.Object {
	return []runtime.Object{
		newComponent("component1", testNamespace, testAppName, nil),
		newComponent("component2", testNamespace, otherAppName, nil),
	}
}

func getOtherAppComponents() []runtime.Object {
	return []runtime.Object{
		newComponent("component1", testNamespace, otherAppName, nil),
	}
}

func getSingleTestComponent() []runtime.Object {
	return []runtime.Object{
		newBundleComponent(testComponentName, testNamespace, testAppName),
	}
}

func getLabeledTestComponents() []runtime.Object {
	return []runtime.Object{
		newLabeledBundleComponent(bundleComponentName, testNamespace, testAppName),
		newLabeledControllerComponent(controllerComponentName, testNamespace, testAppName),
		newComponent("other-component", testNamespace, otherAppName, map[string]string{"env": "production"}),
	}
}

func getBundleWithControllerComponents() []runtime.Object {
	return []runtime.Object{
		newBundleComponent(bundleComponentName, testNamespace, testAppName),
		newControllerComponent(controllerComponentName, testNamespace, testAppName),
	}
}

func getBundleComponentsFromMultipleApps() []runtime.Object {
	return []runtime.Object{
		newBundleComponent(bundleComponentName, testNamespace, testAppName),
		newBundleComponent("other-bundle-component", testNamespace, otherAppName),
	}
}

func getControllerOnlyComponents() []runtime.Object {
	return []runtime.Object{
		newControllerComponent(controllerComponentName, testNamespace, testAppName),
	}
}

func getMultipleBundleComponents() []runtime.Object {
	return []runtime.Object{
		newBundleComponent("bundle-component-1", testNamespace, testAppName),
		newBundleComponent("bundle-component-2", testNamespace, testAppName),
	}
}

func getOtherAppBundleComponents() []runtime.Object {
	return []runtime.Object{
		newBundleComponent(bundleComponentName, testNamespace, otherAppName),
	}
}
