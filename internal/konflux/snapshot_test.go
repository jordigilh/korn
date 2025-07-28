package konflux_test

import (
	"context"
	"fmt"
	"strings"

	"github.com/containers/podman/v5/pkg/domain/entities/types"
	"github.com/containers/podman/v5/pkg/inspect"
	"github.com/jordigilh/korn/internal/konflux"
	"github.com/jordigilh/korn/testutils"
	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Test constants for snapshots
const (
	testSnapshotName     = "test-snapshot"
	finishedSnapshotName = "finished-snapshot"
	pendingSnapshotName  = "pending-snapshot"
	failedSnapshotName   = "failed-snapshot"
	otherAppSnapshotName = "other-app-snapshot"
	testSHA              = "abc123def456"
	testContainerImage   = "registry.test.com/bundle@sha256:def456"
)

var _ = Describe("Snapshot functionality", func() {
	var (
		scheme            *runtime.Scheme
		ns                *corev1.Namespace
		kornInstance      *konflux.Korn
		testClientBuilder *fake.ClientBuilder
	)

	BeforeEach(func() {
		logrus.SetOutput(GinkgoWriter)
		logrus.SetLevel(logrus.DebugLevel)
		scheme = createFakeScheme()
		ns = newNamespace(testutils.TestNamespace)

		kornInstance = &konflux.Korn{
			Namespace:       testutils.TestNamespace,
			ApplicationName: testutils.TestAppName,
			SnapshotName:    testSnapshotName,
			SHA:             testSHA,
		}
	})

	Context("ListSnapshots functionality", func() {
		DescribeTable("should handle various snapshot scenarios",
			func(applicationName string, snapshots []runtime.Object, expectedCount int, expectError bool, description string) {
				kornInstance.ApplicationName = applicationName
				// Create a fresh client builder for each test to avoid state pollution
				testClientBuilder = fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ns).WithRuntimeObjects(snapshots...).WithIndex(&applicationapiv1alpha1.Snapshot{}, "metadata.name", testutils.FilterBySnapshotName)
				kornInstance.KubeClient = testClientBuilder.Build()

				if expectedCount > 0 {
					logrus.Debugf("expectedCount: %d", expectedCount)
					labels := client.MatchingLabels{"pac.test.appstudio.openshift.io/event-type": "push"}
					list := applicationapiv1alpha1.SnapshotList{}
					err := kornInstance.KubeClient.List(context.TODO(), &list, &client.ListOptions{Namespace: kornInstance.Namespace}, labels)
					if err != nil {
						logrus.Errorf("error listing snapshots: %v", err)
					}
					Expect(list.Items).To(HaveLen(expectedCount))
				}
				result, err := kornInstance.ListSnapshots()
				if expectError {
					Expect(err).To(HaveOccurred(), description)
					Expect(result).To(BeNil(), description)
				} else {
					Expect(err).ToNot(HaveOccurred(), description)
					Expect(result).To(HaveLen(expectedCount), description)
				}
			},

			Entry("should return all push event snapshots when ApplicationName is empty",
				"", getSimpleSnapshots(), 2, false,
				"Should return all push event snapshots when no app filter"),
			Entry("should return empty list when no snapshots exist",
				"", []runtime.Object{}, 0, false,
				"Should handle empty snapshot list"),

			Entry("should return snapshots for specific application when components exist",
				testutils.TestAppName, append(getOperatorTestObjects(), getSimpleSnapshots()...), 2, false,
				"Should filter snapshots by application components"),

			Entry("should return error when application type cannot be determined",
				testutils.TestAppName, getSimpleSnapshots(), 0, true,
				"Should fail when application type unknown"),
		)
	})

	Context("GetSnapshot functionality", func() {
		DescribeTable("should handle snapshot retrieval scenarios",
			func(snapshots []runtime.Object, snapshotName, sha, applicationName string, expectError bool, expectedErrorSubstring string, description string) {
				kornInstance.SnapshotName = snapshotName
				kornInstance.SHA = sha
				kornInstance.ApplicationName = applicationName
				// Create a fresh client builder for each test to avoid state pollution
				testClientBuilder = fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ns).WithIndex(&applicationapiv1alpha1.Snapshot{}, "metadata.name", testutils.FilterBySnapshotName)
				if len(snapshots) > 0 {
					testClientBuilder = testClientBuilder.WithRuntimeObjects(snapshots...)
				}
				kornInstance.KubeClient = testClientBuilder.Build()

				result, err := kornInstance.GetSnapshot()

				if expectError {
					Expect(err).To(HaveOccurred(), description)
					Expect(result).To(BeNil(), description)
					if expectedErrorSubstring != "" {
						Expect(err.Error()).To(ContainSubstring(expectedErrorSubstring), description)
					}
				} else {
					Expect(err).ToNot(HaveOccurred(), description)
					Expect(result).ToNot(BeNil(), description)
					if snapshotName != "" {
						Expect(result.Name).To(Equal(snapshotName), description)
					}
				}
			},

			Entry("should return snapshot when it exists by name",
				getSingleTestSnapshot(), testSnapshotName, "", "", false, "",
				"Should get existing snapshot by name"),

			Entry("should return snapshot when it exists by SHA",
				getSingleTestSnapshot(), "", testSHA, testutils.TestAppName, false, "",
				"Should get existing snapshot by SHA"),

			Entry("should return snapshot when it exists by application",
				getSingleTestSnapshot(), "", "", testutils.TestAppName, false, "",
				"Should get existing snapshot by application"),

			Entry("should return error when snapshot not found by SHA",
				[]runtime.Object{}, "", "nonexistent", "", true, "snapshot with SHA nonexistent not found",
				"Should fail for non-existing snapshot"),

			Entry("should return error when no snapshots match criteria",
				[]runtime.Object{}, "", "", testutils.TestAppName, true, "snapshot with SHA  not found",
				"Should fail when no snapshots match"),
		)
	})
})

// hasSnapshotCompletedSuccessfully checks if a snapshot has completed successfully
// This is a local copy since the original function is not exported
func hasSnapshotCompletedSuccessfully(snapshot applicationapiv1alpha1.Snapshot) bool {
	for _, v := range snapshot.Status.Conditions {
		if v.Type == "AppStudioTestSucceeded" && v.Reason == "Finished" {
			return true
		}
	}
	return false
}

// Standalone function tests outside the main Describe block
var _ = Describe("Standalone snapshot functions", func() {
	Context("hasSnapshotCompletedSuccessfully functionality", func() {
		DescribeTable("should correctly identify snapshot completion status",
			func(snapshot applicationapiv1alpha1.Snapshot, expected bool, description string) {
				result := hasSnapshotCompletedSuccessfully(snapshot)
				Expect(result).To(Equal(expected), description)
			},

			Entry("should return true for successfully completed snapshot",
				*newFinishedSnapshot(finishedSnapshotName, testutils.TestNamespace, testutils.TestAppName, testutils.BundleComponentName),
				true, "Should identify successful completion"),

			Entry("should return false for pending snapshot",
				*newPendingSnapshot(pendingSnapshotName, testutils.TestNamespace, testutils.TestAppName, testutils.BundleComponentName),
				false, "Should identify pending status"),

			Entry("should return false for failed snapshot",
				*newFailedSnapshot(failedSnapshotName, testutils.TestNamespace, testutils.TestAppName, testutils.BundleComponentName),
				false, "Should identify failed status"),

			Entry("should return false for snapshot with no conditions",
				*newSnapshot("no-conditions", testutils.TestNamespace, testutils.TestAppName, testutils.BundleComponentName, testSHA, []metav1.Condition{}),
				false, "Should handle empty conditions"),
		)
	})

	Context("GetComponentPullspecFromSnapshot functionality", func() {
		It("should return component pullspec when component exists", func() {
			snapshot := newFinishedSnapshot("test-snapshot", testutils.TestNamespace, testutils.TestAppName, testutils.BundleComponentName)

			result, err := konflux.GetComponentPullspecFromSnapshot(*snapshot, testutils.BundleComponentName)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).To(Equal(testContainerImage))
		})

		It("should return error when component does not exist", func() {
			snapshot := newFinishedSnapshot("test-snapshot", testutils.TestNamespace, testutils.TestAppName, testutils.BundleComponentName)

			result, err := konflux.GetComponentPullspecFromSnapshot(*snapshot, "nonexistent-component")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("component reference"))
			Expect(err.Error()).To(ContainSubstring("not found"))
			Expect(result).To(BeEmpty())
		})
	})
})

// Helper functions for creating test snapshots
func newSnapshot(name, namespace, application, component, sha string, conditions []metav1.Condition) *applicationapiv1alpha1.Snapshot {
	return &applicationapiv1alpha1.Snapshot{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				testutils.EventTypeLabel:   testutils.PushEventType,
				testutils.ApplicationLabel: application,
				testutils.ComponentLabel:   component,
				testutils.SHALabel:         sha,
			},
			CreationTimestamp: metav1.Now(),
		},
		Spec: applicationapiv1alpha1.SnapshotSpec{
			Application: application,
			Components: []applicationapiv1alpha1.SnapshotComponent{
				{
					Name:           testutils.ControllerComponentName,
					ContainerImage: "registry.test.com/controller@sha256:abc123",
				},
				{
					Name:           component,
					ContainerImage: testContainerImage,
				},
			},
		},
		Status: applicationapiv1alpha1.SnapshotStatus{
			Conditions: conditions,
		},
	}
}

func newFinishedSnapshot(name, namespace, application, component string) *applicationapiv1alpha1.Snapshot {
	return newSnapshot(name, namespace, application, component, testSHA, []metav1.Condition{
		{
			Type:   "AppStudioTestSucceeded",
			Reason: "Finished",
			Status: metav1.ConditionTrue,
		},
	})
}

func newPendingSnapshot(name, namespace, application, component string) *applicationapiv1alpha1.Snapshot {
	return newSnapshot(name, namespace, application, component, testSHA, []metav1.Condition{
		{
			Type:   "AppStudioTestSucceeded",
			Reason: "InProgress",
			Status: metav1.ConditionUnknown,
		},
	})
}

func newFailedSnapshot(name, namespace, application, component string) *applicationapiv1alpha1.Snapshot {
	return newSnapshot(name, namespace, application, component, testSHA, []metav1.Condition{
		{
			Type:   "AppStudioTestSucceeded",
			Reason: "Failed",
			Status: metav1.ConditionFalse,
		},
	})
}

// Helper functions to get sets of snapshots for different test scenarios
func getSimpleSnapshots() []runtime.Object {
	return []runtime.Object{
		newFinishedSnapshot(finishedSnapshotName, testutils.TestNamespace, testutils.TestAppName, testutils.BundleComponentName),
		newFinishedSnapshot(otherAppSnapshotName, testutils.TestNamespace, "other-app", testutils.BundleComponentName),
	}
}

func getSingleTestSnapshot() []runtime.Object {
	return []runtime.Object{
		newFinishedSnapshot(testSnapshotName, testutils.TestNamespace, testutils.TestAppName, testutils.BundleComponentName),
	}
}

func getOperatorTestObjects() []runtime.Object {
	return []runtime.Object{
		testutils.NewOperatorApplication(testutils.TestAppName, testutils.TestNamespace),
		testutils.NewBundleComponent(testutils.BundleComponentName, testutils.TestNamespace, testutils.TestAppName),
		testutils.NewControllerComponent(testutils.ControllerComponentName, testutils.TestNamespace, testutils.TestAppName),
	}
}

// Mock image client implementations for testing validateSnapshotCandidacy
type mockImageClientValid struct{}

func (m *mockImageClientValid) GetImageData(image string) (*types.ImageInspectReport, error) {
	return &types.ImageInspectReport{
		ImageData: &inspect.ImageData{
			Labels: map[string]string{
				"controller": "registry.test.com/controller@sha256:abc123",
				"version":    "v1.0.0",
			},
		},
	}, nil
}

type mockImageClientError struct{}

func (m *mockImageClientError) GetImageData(image string) (*types.ImageInspectReport, error) {
	return nil, fmt.Errorf("failed to get image data for %s", image)
}

type mockImageClientMissingLabel struct{}

func (m *mockImageClientMissingLabel) GetImageData(image string) (*types.ImageInspectReport, error) {
	return &types.ImageInspectReport{
		ImageData: &inspect.ImageData{
			Labels: map[string]string{
				// Missing controller label
				"version": "v1.0.0",
			},
		},
	}, nil
}

type mockImageClientMismatchedSHA struct{}

func (m *mockImageClientMismatchedSHA) GetImageData(image string) (*types.ImageInspectReport, error) {
	return &types.ImageInspectReport{
		ImageData: &inspect.ImageData{
			Labels: map[string]string{
				"controller": "registry.test.com/controller@sha256:different123", // Different SHA
				"version":    "v1.0.0",
			},
		},
	}, nil
}

type mockImageClientVersionMismatch struct{}

func (m *mockImageClientVersionMismatch) GetImageData(image string) (*types.ImageInspectReport, error) {
	if strings.Contains(image, "controller") {
		return &types.ImageInspectReport{
			ImageData: &inspect.ImageData{
				Labels: map[string]string{
					"controller": "registry.test.com/controller@sha256:abc123",
					"version":    "v1.0.1", // Different version
				},
			},
		}, nil
	}
	return &types.ImageInspectReport{
		ImageData: &inspect.ImageData{
			Labels: map[string]string{
				"controller": "registry.test.com/controller@sha256:abc123",
				"version":    "v1.0.0",
			},
		},
	}, nil
}

// validateSnapshotCandidacy tests - testing through public API
var _ = Describe("validateSnapshotCandidacy functionality", func() {
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
		}
	})

	Context("Snapshot test status validation", func() {
		It("should return false when snapshot has not completed successfully", func() {
			// Create a snapshot with pending status
			snapshot := newPendingSnapshot("pending-snapshot", testutils.TestNamespace, testutils.TestAppName, testutils.BundleComponentName)

			// Create components
			components := []runtime.Object{
				testutils.NewBundleComponent(testutils.BundleComponentName, testutils.TestNamespace, testutils.TestAppName),
				testutils.NewControllerComponent(testutils.ControllerComponentName, testutils.TestNamespace, testutils.TestAppName),
			}

			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(append(components, snapshot)...)
			kornInstance.KubeClient = fakeClientBuilder.Build()
			kornInstance.PodClient = &mockImageClientValid{}

			// Test through GetSnapshotCandidateForRelease which uses validateSnapshotCandidacy internally
			result, err := kornInstance.GetSnapshotCandidateForRelease()

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})

		It("should return false when snapshot has failed", func() {
			// Create a snapshot with failed status
			snapshot := newFailedSnapshot("failed-snapshot", testutils.TestNamespace, testutils.TestAppName, testutils.BundleComponentName)

			// Create components
			components := []runtime.Object{
				testutils.NewBundleComponent(testutils.BundleComponentName, testutils.TestNamespace, testutils.TestAppName),
				testutils.NewControllerComponent(testutils.ControllerComponentName, testutils.TestNamespace, testutils.TestAppName),
			}

			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(append(components, snapshot)...)
			kornInstance.KubeClient = fakeClientBuilder.Build()
			kornInstance.PodClient = &mockImageClientValid{}

			// Test through GetSnapshotCandidateForRelease which uses validateSnapshotCandidacy internally
			result, err := kornInstance.GetSnapshotCandidateForRelease()

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Context("Bundle component pullspec retrieval", func() {
		It("should return error when bundle component not found in snapshot", func() {
			// Create a snapshot without the bundle component
			snapshot := newFinishedSnapshot("missing-bundle-snapshot", testutils.TestNamespace, testutils.TestAppName, "other-component")
			snapshot.Spec.Components = []applicationapiv1alpha1.SnapshotComponent{
				{
					Name:           "other-component",
					ContainerImage: "registry.test.com/other@sha256:abc123",
				},
			}

			// Create components
			components := []runtime.Object{
				testutils.NewBundleComponent(testutils.BundleComponentName, testutils.TestNamespace, testutils.TestAppName),
				testutils.NewControllerComponent(testutils.ControllerComponentName, testutils.TestNamespace, testutils.TestAppName),
			}

			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(append(components, snapshot)...)
			kornInstance.KubeClient = fakeClientBuilder.Build()
			kornInstance.PodClient = &mockImageClientValid{}

			// Test through GetSnapshotCandidateForRelease which uses validateSnapshotCandidacy internally
			result, err := kornInstance.GetSnapshotCandidateForRelease()

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Context("Bundle image data retrieval", func() {
		It("should return error when bundle image cannot be pulled", func() {
			// Create a valid snapshot
			snapshot := newFinishedSnapshot("valid-snapshot", testutils.TestNamespace, testutils.TestAppName, testutils.BundleComponentName)

			// Create components
			components := []runtime.Object{
				testutils.NewBundleComponent(testutils.BundleComponentName, testutils.TestNamespace, testutils.TestAppName),
				testutils.NewControllerComponent(testutils.ControllerComponentName, testutils.TestNamespace, testutils.TestAppName),
			}

			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(append(components, snapshot)...)
			kornInstance.KubeClient = fakeClientBuilder.Build()
			kornInstance.PodClient = &mockImageClientError{}

			// Test through GetSnapshotCandidateForRelease which uses validateSnapshotCandidacy internally
			result, err := kornInstance.GetSnapshotCandidateForRelease()

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Context("Component list retrieval", func() {
		It("should return error when component list retrieval fails", func() {
			// Create a valid snapshot
			snapshot := newFinishedSnapshot("valid-snapshot", testutils.TestNamespace, testutils.TestAppName, testutils.BundleComponentName)

			// Don't add any components to the fake client
			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(snapshot)
			kornInstance.KubeClient = fakeClientBuilder.Build()
			kornInstance.PodClient = &mockImageClientValid{}

			// Test through GetSnapshotCandidateForRelease which uses validateSnapshotCandidacy internally
			result, err := kornInstance.GetSnapshotCandidateForRelease()

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Context("Bundle reference label validation", func() {
		It("should return error when component missing bundle reference label", func() {
			// Create a valid snapshot
			snapshot := newFinishedSnapshot("valid-snapshot", testutils.TestNamespace, testutils.TestAppName, testutils.BundleComponentName)

			// Create components but controller component missing bundle label
			components := []runtime.Object{
				testutils.NewBundleComponent(testutils.BundleComponentName, testutils.TestNamespace, testutils.TestAppName),
				&applicationapiv1alpha1.Component{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testutils.ControllerComponentName,
						Namespace: testutils.TestNamespace,
						Labels: map[string]string{
							// Missing konflux.BundleReferenceLabel
							"appstudio.openshift.io/application": testutils.TestAppName,
						},
					},
					Spec: applicationapiv1alpha1.ComponentSpec{
						Application: testutils.TestAppName,
					},
				},
			}

			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(append(components, snapshot)...)
			kornInstance.KubeClient = fakeClientBuilder.Build()
			kornInstance.PodClient = &mockImageClientValid{}

			// Test through GetSnapshotCandidateForRelease which uses validateSnapshotCandidacy internally
			result, err := kornInstance.GetSnapshotCandidateForRelease()

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Context("Bundle label existence check", func() {
		It("should return false when bundle image missing expected component label", func() {
			// Create a valid snapshot
			snapshot := newFinishedSnapshot("valid-snapshot", testutils.TestNamespace, testutils.TestAppName, testutils.BundleComponentName)

			// Create components
			components := []runtime.Object{
				testutils.NewBundleComponent(testutils.BundleComponentName, testutils.TestNamespace, testutils.TestAppName),
				testutils.NewControllerComponent(testutils.ControllerComponentName, testutils.TestNamespace, testutils.TestAppName),
			}

			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(append(components, snapshot)...)
			kornInstance.KubeClient = fakeClientBuilder.Build()
			kornInstance.PodClient = &mockImageClientMissingLabel{}

			// Test through GetSnapshotCandidateForRelease which uses validateSnapshotCandidacy internally
			result, err := kornInstance.GetSnapshotCandidateForRelease()

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Context("Component pullspec retrieval", func() {
		It("should return error when component not found in snapshot", func() {
			// Create a snapshot missing the controller component
			snapshot := newFinishedSnapshot("missing-component-snapshot", testutils.TestNamespace, testutils.TestAppName, testutils.BundleComponentName)
			snapshot.Spec.Components = []applicationapiv1alpha1.SnapshotComponent{
				{
					Name:           testutils.BundleComponentName,
					ContainerImage: testContainerImage,
				},
				// Missing controller component
			}

			// Create components
			components := []runtime.Object{
				testutils.NewBundleComponent(testutils.BundleComponentName, testutils.TestNamespace, testutils.TestAppName),
				testutils.NewControllerComponent(testutils.ControllerComponentName, testutils.TestNamespace, testutils.TestAppName),
			}

			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(append(components, snapshot)...)
			kornInstance.KubeClient = fakeClientBuilder.Build()
			kornInstance.PodClient = &mockImageClientValid{}

			// Test through GetSnapshotCandidateForRelease which uses validateSnapshotCandidacy internally
			result, err := kornInstance.GetSnapshotCandidateForRelease()

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Context("SHA256 digest comparison", func() {
		It("should return false when bundle label SHA256 doesn't match snapshot component SHA256", func() {
			// Create a valid snapshot
			snapshot := newFinishedSnapshot("valid-snapshot", testutils.TestNamespace, testutils.TestAppName, testutils.BundleComponentName)

			// Create components
			components := []runtime.Object{
				testutils.NewBundleComponent(testutils.BundleComponentName, testutils.TestNamespace, testutils.TestAppName),
				testutils.NewControllerComponent(testutils.ControllerComponentName, testutils.TestNamespace, testutils.TestAppName),
			}

			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(append(components, snapshot)...)
			kornInstance.KubeClient = fakeClientBuilder.Build()
			kornInstance.PodClient = &mockImageClientMismatchedSHA{}

			// Test through GetSnapshotCandidateForRelease which uses validateSnapshotCandidacy internally
			result, err := kornInstance.GetSnapshotCandidateForRelease()

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Context("Component image data retrieval", func() {
		It("should return error when component image cannot be pulled", func() {
			// Create a valid snapshot
			snapshot := newFinishedSnapshot("valid-snapshot", testutils.TestNamespace, testutils.TestAppName, testutils.BundleComponentName)

			// Create components
			components := []runtime.Object{
				testutils.NewBundleComponent(testutils.BundleComponentName, testutils.TestNamespace, testutils.TestAppName),
				testutils.NewControllerComponent(testutils.ControllerComponentName, testutils.TestNamespace, testutils.TestAppName),
			}

			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(append(components, snapshot)...)
			kornInstance.KubeClient = fakeClientBuilder.Build()
			kornInstance.PodClient = &mockImageClientError{}

			// Test through GetSnapshotCandidateForRelease which uses validateSnapshotCandidacy internally
			result, err := kornInstance.GetSnapshotCandidateForRelease()

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Context("Version label consistency", func() {
		It("should return false when component and bundle have different version labels", func() {
			// Create a valid snapshot
			snapshot := newFinishedSnapshot("valid-snapshot", testutils.TestNamespace, testutils.TestAppName, testutils.BundleComponentName)

			// Create components
			components := []runtime.Object{
				testutils.NewBundleComponent(testutils.BundleComponentName, testutils.TestNamespace, testutils.TestAppName),
				testutils.NewControllerComponent(testutils.ControllerComponentName, testutils.TestNamespace, testutils.TestAppName),
			}

			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(append(components, snapshot)...)
			kornInstance.KubeClient = fakeClientBuilder.Build()
			kornInstance.PodClient = &mockImageClientVersionMismatch{}

			// Test through GetSnapshotCandidateForRelease which uses validateSnapshotCandidacy internally
			result, err := kornInstance.GetSnapshotCandidateForRelease()

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})

	Context("Successful validation", func() {
		It("should return true when all validations pass", func() {
			// Create a valid snapshot
			snapshot := newFinishedSnapshot("valid-snapshot", testutils.TestNamespace, testutils.TestAppName, testutils.BundleComponentName)

			// Create components
			components := []runtime.Object{
				testutils.NewBundleComponent(testutils.BundleComponentName, testutils.TestNamespace, testutils.TestAppName),
				testutils.NewControllerComponent(testutils.ControllerComponentName, testutils.TestNamespace, testutils.TestAppName),
			}
			application := testutils.NewApplication("test-app", "test-namespace", map[string]string{"korn.redhat.io/application": "operator"})

			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(append(components, snapshot, application)...)
			kornInstance.KubeClient = fakeClientBuilder.Build()
			kornInstance.PodClient = &mockImageClientValid{}

			// Test through GetSnapshotCandidateForRelease which uses validateSnapshotCandidacy internally
			result, err := kornInstance.GetSnapshotCandidateForRelease()

			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
		})
	})

	Context("Edge cases", func() {
		It("should return true when only bundle component exists (empty component list)", func() {
			// Create a valid snapshot
			snapshot := newFinishedSnapshot("valid-snapshot", testutils.TestNamespace, testutils.TestAppName, testutils.BundleComponentName)

			// Create only bundle component (no other components)
			components := []runtime.Object{
				testutils.NewBundleComponent(testutils.BundleComponentName, testutils.TestNamespace, testutils.TestAppName),
			}

			application := testutils.NewApplication("test-app", "test-namespace", map[string]string{"korn.redhat.io/application": "operator"})
			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(append(components, snapshot, application)...)
			kornInstance.KubeClient = fakeClientBuilder.Build()
			kornInstance.PodClient = &mockImageClientValid{}

			// Test through GetSnapshotCandidateForRelease which uses validateSnapshotCandidacy internally
			result, err := kornInstance.GetSnapshotCandidateForRelease()

			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeNil())
		})

		It("should handle multiple components with validation failures", func() {
			// Create a valid snapshot
			snapshot := newFinishedSnapshot("valid-snapshot", testutils.TestNamespace, testutils.TestAppName, testutils.BundleComponentName)

			// Create multiple components including one with missing bundle label
			components := []runtime.Object{
				testutils.NewBundleComponent(testutils.BundleComponentName, testutils.TestNamespace, testutils.TestAppName),
				testutils.NewControllerComponent(testutils.ControllerComponentName, testutils.TestNamespace, testutils.TestAppName),
				&applicationapiv1alpha1.Component{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "another-component",
						Namespace: testutils.TestNamespace,
						Labels: map[string]string{
							// Missing bundle label
							"appstudio.openshift.io/application": testutils.TestAppName,
						},
					},
					Spec: applicationapiv1alpha1.ComponentSpec{
						Application: testutils.TestAppName,
					},
				},
			}

			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(append(components, snapshot)...)
			kornInstance.KubeClient = fakeClientBuilder.Build()
			kornInstance.PodClient = &mockImageClientValid{}

			// Test through GetSnapshotCandidateForRelease which uses validateSnapshotCandidacy internally
			result, err := kornInstance.GetSnapshotCandidateForRelease()

			Expect(err).To(HaveOccurred())
			Expect(result).To(BeNil())
		})
	})
})
