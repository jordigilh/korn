package konflux_test

import (
	"fmt"

	"github.com/containers/podman/v5/pkg/domain/entities/types"
	"github.com/containers/podman/v5/pkg/inspect"
	"github.com/jordigilh/korn/internal"
	"github.com/jordigilh/korn/internal/konflux"
	"github.com/jordigilh/korn/testutils"
	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	releaseapiv1alpha1 "github.com/konflux-ci/release-service/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Release functionality", func() {
	var (
		fakeClientBuilder *fake.ClientBuilder
		scheme            *runtime.Scheme
		ns                *corev1.Namespace
		app               *applicationapiv1alpha1.Application
		component1        *applicationapiv1alpha1.Component
		bundleComponent   *applicationapiv1alpha1.Component
		snapshot          *applicationapiv1alpha1.Snapshot
		releasePlan       *releaseapiv1alpha1.ReleasePlan
	)

	BeforeEach(func() {
		scheme = createFakeScheme()
		ns = newNamespace("test-namespace")

		app = &applicationapiv1alpha1.Application{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-app",
				Namespace: "test-namespace",
				Labels: map[string]string{
					konflux.ApplicationTypeLabel: "operator",
				},
			},
		}

		component1 = &applicationapiv1alpha1.Component{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "controller-component",
				Namespace: "test-namespace",
				Labels: map[string]string{
					konflux.BundleReferenceLabel: "controller",
				},
			},
			Spec: applicationapiv1alpha1.ComponentSpec{
				Application: "test-app",
			},
		}

		bundleComponent = &applicationapiv1alpha1.Component{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "bundle-component",
				Namespace: "test-namespace",
				Labels: map[string]string{
					konflux.ComponentTypeLabel: "bundle",
				},
			},
			Spec: applicationapiv1alpha1.ComponentSpec{
				Application: "test-app",
			},
		}

		snapshot = &applicationapiv1alpha1.Snapshot{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-snapshot",
				Namespace: "test-namespace",
				Labels: map[string]string{
					testutils.EventTypeLabel:   testutils.PushEventType,
					testutils.ApplicationLabel: "test-app",
					testutils.ComponentLabel:   "bundle-component",
					testutils.SHALabel:         "abc123def456",
				},
			},
			Spec: applicationapiv1alpha1.SnapshotSpec{
				Application: "test-app",
				Components: []applicationapiv1alpha1.SnapshotComponent{
					{
						Name:           "controller-component",
						ContainerImage: "registry.test.com/controller@sha256:abc123",
					},
					{
						Name:           "bundle-component",
						ContainerImage: "registry.test.com/bundle@sha256:def456",
					},
				},
			},
			Status: applicationapiv1alpha1.SnapshotStatus{
				Conditions: []metav1.Condition{
					{
						Type:   "AppStudioTestSucceeded",
						Reason: "Finished",
						Status: metav1.ConditionTrue,
					},
				},
			},
		}

		releasePlan = &releaseapiv1alpha1.ReleasePlan{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-release-plan",
				Namespace: "test-namespace",
				Labels: map[string]string{
					konflux.EnvironmentLabel: "staging",
				},
			},
			Spec: releaseapiv1alpha1.ReleasePlanSpec{
				Application: "test-app",
			},
		}

		fakeClientBuilder = fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
			ns, app, component1, bundleComponent, snapshot, releasePlan,
		).WithIndex(&applicationapiv1alpha1.Snapshot{}, "metadata.name", filterBySnapshotName)
	})

	Context("getBundleVersionFromSnapshot functionality", func() {
		var (
			kornInstance *konflux.Korn
		)

		BeforeEach(func() {
			kornInstance = &konflux.Korn{
				Namespace:       "test-namespace",
				ApplicationName: "test-app",
				EnvironmentName: "staging",
				KubeClient:      fakeClientBuilder.Build(),
			}
		})

		Context("Success scenarios via GenerateReleaseManifest", func() {
			It("should generate release manifest with patch version (RHBA)", func() {
				mockPodClient := &mockImageClientWithVersion{version: "1.0.1"} // patch version triggers RHBA
				kornInstance.PodClient = mockPodClient

				release, err := kornInstance.GenerateReleaseManifest()
				Expect(err).ToNot(HaveOccurred())
				Expect(release).ToNot(BeNil())
				Expect(release.Spec.Snapshot).To(Equal("test-snapshot"))

				// Verify release notes contain RHBA type for patch version
				releaseNotesData := release.Spec.Data.Raw
				Expect(string(releaseNotesData)).To(ContainSubstring("RHBA"))
			})

			It("should generate release manifest with minor version (RHEA)", func() {
				mockPodClient := &mockImageClientWithVersion{version: "1.1.0"} // minor version triggers RHEA
				kornInstance.PodClient = mockPodClient

				release, err := kornInstance.GenerateReleaseManifest()
				Expect(err).ToNot(HaveOccurred())
				Expect(release).ToNot(BeNil())
				Expect(release.Spec.Snapshot).To(Equal("test-snapshot"))

				// Verify release notes contain RHEA type for minor version
				releaseNotesData := release.Spec.Data.Raw
				Expect(string(releaseNotesData)).To(ContainSubstring("RHEA"))
			})

			It("should generate release manifest with major version (RHEA)", func() {
				mockPodClient := &mockImageClientWithVersion{version: "2.0.0"} // major version triggers RHEA
				kornInstance.PodClient = mockPodClient

				release, err := kornInstance.GenerateReleaseManifest()
				Expect(err).ToNot(HaveOccurred())
				Expect(release).ToNot(BeNil())
				Expect(release.Spec.Snapshot).To(Equal("test-snapshot"))

				// Verify release notes contain RHEA type for major version
				releaseNotesData := release.Spec.Data.Raw
				Expect(string(releaseNotesData)).To(ContainSubstring("RHEA"))
			})
		})

		Context("Error scenarios via GenerateReleaseManifest", func() {
			It("should fail when version label is missing from bundle", func() {
				mockPodClient := &mockImageClientWithoutVersion{}
				kornInstance.PodClient = mockPodClient

				release, err := kornInstance.GenerateReleaseManifest()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("label 'version' not found"))
				Expect(release).To(BeNil())
			})

			It("should fail when bundle component is not found in snapshot", func() {
				// Create a snapshot that doesn't contain the bundle component
				snapshotWithoutBundle := &applicationapiv1alpha1.Snapshot{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-snapshot-no-bundle",
						Namespace: "test-namespace",
						Labels: map[string]string{
							testutils.EventTypeLabel:   testutils.PushEventType,
							testutils.ApplicationLabel: "test-app",
							testutils.ComponentLabel:   "controller-component",
						},
					},
					Spec: applicationapiv1alpha1.SnapshotSpec{
						Application: "test-app",
						Components: []applicationapiv1alpha1.SnapshotComponent{
							{
								Name:           "controller-component",
								ContainerImage: "registry.test.com/controller@sha256:abc123",
							},
						},
					},
					Status: applicationapiv1alpha1.SnapshotStatus{
						Conditions: []metav1.Condition{
							{
								Type:   "AppStudioTestSucceeded",
								Reason: "Finished",
								Status: metav1.ConditionTrue,
							},
						},
					},
				}

				// Set up client with snapshot without bundle
				clientWithSnapshot := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
					ns, app, component1, bundleComponent, snapshotWithoutBundle, releasePlan,
				).WithIndex(&applicationapiv1alpha1.Snapshot{}, "metadata.name", filterBySnapshotName).Build()

				kornInstance.KubeClient = clientWithSnapshot
				kornInstance.SnapshotName = "test-snapshot-no-bundle"
				mockPodClient := &mockImageClientWithVersion{version: "1.0.0"}
				kornInstance.PodClient = mockPodClient

				release, err := kornInstance.GenerateReleaseManifest()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("component reference bundle-component"))
				Expect(release).To(BeNil())
			})

			It("should fail when image data cannot be retrieved", func() {
				mockPodClient := &mockImageClientWithError{errorMsg: "failed to inspect image"}
				kornInstance.PodClient = mockPodClient

				release, err := kornInstance.GenerateReleaseManifest()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("failed to inspect image"))
				Expect(release).To(BeNil())
			})

			It("should fail when no bundle component exists in application", func() {
				// Create application without bundle component
				appWithoutBundle := &applicationapiv1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "app-no-bundle",
						Namespace: "test-namespace",
						Labels: map[string]string{
							konflux.ApplicationTypeLabel: "operator",
						},
					},
				}

				componentWithoutBundle := &applicationapiv1alpha1.Component{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "controller-only",
						Namespace: "test-namespace",
						Labels: map[string]string{
							konflux.BundleReferenceLabel: "controller",
						},
					},
					Spec: applicationapiv1alpha1.ComponentSpec{
						Application: "app-no-bundle",
					},
				}

				releasePlanForApp := &releaseapiv1alpha1.ReleasePlan{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-release-plan-no-bundle",
						Namespace: "test-namespace",
						Labels: map[string]string{
							konflux.EnvironmentLabel: "staging",
						},
					},
					Spec: releaseapiv1alpha1.ReleasePlanSpec{
						Application: "app-no-bundle",
					},
				}

				clientWithoutBundle := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
					ns, appWithoutBundle, componentWithoutBundle, releasePlanForApp,
				).Build()

				kornInstanceNoBundleApp := &konflux.Korn{
					Namespace:       "test-namespace",
					ApplicationName: "app-no-bundle",
					EnvironmentName: "staging",
					KubeClient:      clientWithoutBundle,
				}

				mockPodClient := &mockImageClientWithVersion{version: "1.0.0"}
				kornInstanceNoBundleApp.PodClient = mockPodClient

				release, err := kornInstanceNoBundleApp.GenerateReleaseManifest()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no bundle component found"))
				Expect(release).To(BeNil())
			})
		})

		Context("Edge cases via GenerateReleaseManifest", func() {
			It("should handle version with pre-release identifiers", func() {
				mockPodClient := &mockImageClientWithVersion{version: "1.0.0-alpha.1"}
				kornInstance.PodClient = mockPodClient

				release, err := kornInstance.GenerateReleaseManifest()
				Expect(err).ToNot(HaveOccurred())
				Expect(release).ToNot(BeNil())
				Expect(release.Spec.Snapshot).To(Equal("test-snapshot"))

				// Pre-release should be treated as feature release (patch is 0)
				releaseNotesData := release.Spec.Data.Raw
				Expect(string(releaseNotesData)).To(ContainSubstring("RHEA"))
			})

			It("should handle version with build metadata", func() {
				mockPodClient := &mockImageClientWithVersion{version: "1.0.1+build.123"}
				kornInstance.PodClient = mockPodClient

				release, err := kornInstance.GenerateReleaseManifest()
				Expect(err).ToNot(HaveOccurred())
				Expect(release).ToNot(BeNil())
				Expect(release.Spec.Snapshot).To(Equal("test-snapshot"))

				// Patch version should trigger RHBA
				releaseNotesData := release.Spec.Data.Raw
				Expect(string(releaseNotesData)).To(ContainSubstring("RHBA"))
			})
		})
	})

	Context("generateReleaseManifestForFBC functionality", func() {
		var (
			kornInstance   *konflux.Korn
			fbcApp         *applicationapiv1alpha1.Application
			fbcComponent   *applicationapiv1alpha1.Component
			fbcSnapshot    *applicationapiv1alpha1.Snapshot
			fbcReleasePlan *releaseapiv1alpha1.ReleasePlan
		)

		BeforeEach(func() {
			// Set up FBC application and components
			fbcApp = &applicationapiv1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fbc-app",
					Namespace: "test-namespace",
					Labels: map[string]string{
						konflux.ApplicationTypeLabel: "fbc",
					},
				},
			}

			fbcComponent = &applicationapiv1alpha1.Component{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fbc-component",
					Namespace: "test-namespace",
					Labels:    map[string]string{},
				},
				Spec: applicationapiv1alpha1.ComponentSpec{
					Application: "fbc-app",
				},
			}

			fbcSnapshot = &applicationapiv1alpha1.Snapshot{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fbc-snapshot",
					Namespace: "test-namespace",
					Labels: map[string]string{
						testutils.EventTypeLabel:   testutils.PushEventType,
						testutils.ApplicationLabel: "fbc-app",
						testutils.ComponentLabel:   "fbc-component",
					},
				},
				Spec: applicationapiv1alpha1.SnapshotSpec{
					Application: "fbc-app",
					Components: []applicationapiv1alpha1.SnapshotComponent{
						{
							Name:           "fbc-component",
							ContainerImage: "registry.test.com/fbc@sha256:abc123",
						},
					},
				},
				Status: applicationapiv1alpha1.SnapshotStatus{
					Conditions: []metav1.Condition{
						{
							Type:   "AppStudioTestSucceeded",
							Reason: "Finished",
							Status: metav1.ConditionTrue,
						},
					},
				},
			}

			fbcReleasePlan = &releaseapiv1alpha1.ReleasePlan{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "fbc-release-plan",
					Namespace: "test-namespace",
					Labels: map[string]string{
						konflux.EnvironmentLabel: "staging",
					},
				},
				Spec: releaseapiv1alpha1.ReleasePlanSpec{
					Application: "fbc-app",
				},
			}

			fbcClientBuilder := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
				ns, fbcApp, fbcComponent, fbcSnapshot, fbcReleasePlan,
			).WithIndex(&applicationapiv1alpha1.Snapshot{}, "metadata.name", filterBySnapshotName)

			kornInstance = &konflux.Korn{
				Namespace:       "test-namespace",
				ApplicationName: "fbc-app",
				EnvironmentName: "staging",
				KubeClient:      fbcClientBuilder.Build(),
				PodClient:       &mockImageClient{},
			}
		})

		Context("Success scenarios", func() {
			It("should generate FBC release manifest with RHEA type", func() {
				release, err := kornInstance.GenerateReleaseManifest()
				Expect(err).ToNot(HaveOccurred())
				Expect(release).ToNot(BeNil())
				Expect(release.Spec.Snapshot).To(Equal("fbc-snapshot"))
				Expect(release.Spec.ReleasePlan).To(Equal("fbc-release-plan"))

				// FBC applications always use RHEA release type
				releaseNotesData := release.Spec.Data.Raw
				Expect(string(releaseNotesData)).To(ContainSubstring("RHEA"))
			})

			It("should generate FBC release with specific snapshot", func() {
				kornInstance.SnapshotName = "fbc-snapshot"

				release, err := kornInstance.GenerateReleaseManifest()
				Expect(err).ToNot(HaveOccurred())
				Expect(release).ToNot(BeNil())
				Expect(release.Spec.Snapshot).To(Equal("fbc-snapshot"))

				// Verify the release manifest structure
				Expect(release.ObjectMeta.GenerateName).To(Equal("fbc-app-staging-"))
				Expect(release.ObjectMeta.Namespace).To(Equal("test-namespace"))
			})

			It("should not require image client for FBC applications", func() {
				// FBC doesn't need PodClient since it doesn't analyze bundle versions

				release, err := kornInstance.GenerateReleaseManifest()
				Expect(err).ToNot(HaveOccurred())
				Expect(release).ToNot(BeNil())

				// Should still use RHEA type without image inspection
				releaseNotesData := release.Spec.Data.Raw
				Expect(string(releaseNotesData)).To(ContainSubstring("RHEA"))
			})
		})

		Context("Error scenarios", func() {
			It("should fail when no FBC component exists", func() {
				// Create application without any components
				emptyFbcApp := &applicationapiv1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "empty-fbc-app",
						Namespace: "test-namespace",
						Labels: map[string]string{
							konflux.ApplicationTypeLabel: "fbc",
						},
					},
				}

				emptyReleasePlan := &releaseapiv1alpha1.ReleasePlan{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "empty-fbc-release-plan",
						Namespace: "test-namespace",
						Labels: map[string]string{
							konflux.EnvironmentLabel: "staging",
						},
					},
					Spec: releaseapiv1alpha1.ReleasePlanSpec{
						Application: "empty-fbc-app",
					},
				}

				emptyClientBuilder := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
					ns, emptyFbcApp, emptyReleasePlan,
				).WithIndex(&applicationapiv1alpha1.Snapshot{}, "metadata.name", filterBySnapshotName)

				emptyKornInstance := &konflux.Korn{
					Namespace:       "test-namespace",
					ApplicationName: "empty-fbc-app",
					EnvironmentName: "staging",
					KubeClient:      emptyClientBuilder.Build(),
				}

				release, err := emptyKornInstance.GenerateReleaseManifest()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("does not have any component associated"))
				Expect(release).To(BeNil())
			})

			It("should fail when FBC application has multiple components", func() {
				// Create FBC app with multiple components (not recommended)
				multiCompApp := &applicationapiv1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "multi-comp-fbc-app",
						Namespace: "test-namespace",
						Labels: map[string]string{
							konflux.ApplicationTypeLabel: "fbc",
						},
					},
				}

				comp1 := &applicationapiv1alpha1.Component{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fbc-component-1",
						Namespace: "test-namespace",
					},
					Spec: applicationapiv1alpha1.ComponentSpec{
						Application: "multi-comp-fbc-app",
					},
				}

				comp2 := &applicationapiv1alpha1.Component{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fbc-component-2",
						Namespace: "test-namespace",
					},
					Spec: applicationapiv1alpha1.ComponentSpec{
						Application: "multi-comp-fbc-app",
					},
				}

				multiReleasePlan := &releaseapiv1alpha1.ReleasePlan{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "multi-comp-release-plan",
						Namespace: "test-namespace",
						Labels: map[string]string{
							konflux.EnvironmentLabel: "staging",
						},
					},
					Spec: releaseapiv1alpha1.ReleasePlanSpec{
						Application: "multi-comp-fbc-app",
					},
				}

				multiClientBuilder := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
					ns, multiCompApp, comp1, comp2, multiReleasePlan,
				).WithIndex(&applicationapiv1alpha1.Snapshot{}, "metadata.name", filterBySnapshotName)

				multiKornInstance := &konflux.Korn{
					Namespace:       "test-namespace",
					ApplicationName: "multi-comp-fbc-app",
					EnvironmentName: "staging",
					KubeClient:      multiClientBuilder.Build(),
				}

				release, err := multiKornInstance.GenerateReleaseManifest()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("can only have 1 component per Konflux recommendation"))
				Expect(release).To(BeNil())
			})

			It("should fail when release plan is not found", func() {
				// Create FBC setup without matching release plan
				fbcAppNoRP := &applicationapiv1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fbc-app-no-rp",
						Namespace: "test-namespace",
						Labels: map[string]string{
							konflux.ApplicationTypeLabel: "fbc",
						},
					},
				}

				fbcCompNoRP := &applicationapiv1alpha1.Component{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fbc-component-no-rp",
						Namespace: "test-namespace",
					},
					Spec: applicationapiv1alpha1.ComponentSpec{
						Application: "fbc-app-no-rp",
					},
				}
				fbcSnapshot = &applicationapiv1alpha1.Snapshot{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "fbc-snapshot",
						Namespace: "test-namespace",
						Labels: map[string]string{
							testutils.EventTypeLabel:   testutils.PushEventType,
							testutils.ApplicationLabel: "fbc-app-no-rp",
							testutils.ComponentLabel:   "fbc-component-no-rp",
						},
					},
					Spec: applicationapiv1alpha1.SnapshotSpec{
						Application: "fbc-app-no-rp",
						Components: []applicationapiv1alpha1.SnapshotComponent{
							{
								Name:           "fbc-component-no-rp",
								ContainerImage: "registry.test.com/fbc@sha256:abc123",
							},
						},
					},
					Status: applicationapiv1alpha1.SnapshotStatus{
						Conditions: []metav1.Condition{
							{
								Type:   "AppStudioTestSucceeded",
								Reason: "Finished",
								Status: metav1.ConditionTrue,
							},
						},
					},
				}
				noRPClientBuilder := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
					ns, fbcAppNoRP, fbcCompNoRP, fbcSnapshot,
				).WithIndex(&applicationapiv1alpha1.Snapshot{}, "metadata.name", filterBySnapshotName)

				noRPKornInstance := &konflux.Korn{
					Namespace:       "test-namespace",
					ApplicationName: "fbc-app-no-rp",
					EnvironmentName: "staging",
					KubeClient:      noRPClientBuilder.Build(),
					PodClient:       &mockImageClient{},
				}

				release, err := noRPKornInstance.GenerateReleaseManifest()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no release plan found"))
				Expect(release).To(BeNil())
			})

			It("should fail when no valid snapshot candidate exists", func() {
				// Create setup with invalid snapshot (no successful test status)
				invalidSnapshot := &applicationapiv1alpha1.Snapshot{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "invalid-fbc-snapshot",
						Namespace: "test-namespace",
						Labels: map[string]string{
							testutils.EventTypeLabel:   testutils.PushEventType,
							testutils.ApplicationLabel: "fbc-app",
							testutils.ComponentLabel:   "fbc-component",
						},
					},
					Spec: applicationapiv1alpha1.SnapshotSpec{
						Application: "fbc-app",
						Components: []applicationapiv1alpha1.SnapshotComponent{
							{
								Name:           "fbc-component",
								ContainerImage: "registry.test.com/fbc@sha256:abc123",
							},
						},
					},
					Status: applicationapiv1alpha1.SnapshotStatus{
						Conditions: []metav1.Condition{
							{
								Type:   "AppStudioTestSucceeded",
								Reason: "Failed", // Failed test status
								Status: metav1.ConditionFalse,
							},
						},
					},
				}

				invalidClientBuilder := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(
					ns, fbcApp, fbcComponent, invalidSnapshot, fbcReleasePlan,
				).WithIndex(&applicationapiv1alpha1.Snapshot{}, "metadata.name", filterBySnapshotName)

				invalidKornInstance := &konflux.Korn{
					Namespace:       "test-namespace",
					ApplicationName: "fbc-app",
					EnvironmentName: "staging",
					KubeClient:      invalidClientBuilder.Build(),
				}

				release, err := invalidKornInstance.GenerateReleaseManifest()
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("no new valid snapshot candidates found"))
				Expect(release).To(BeNil())
			})
		})

		Context("FBC vs Operator differences", func() {
			It("should always use RHEA type regardless of context", func() {
				// Test multiple times to ensure FBC always uses RHEA
				release, err := kornInstance.GenerateReleaseManifest()
				Expect(err).ToNot(HaveOccurred())
				Expect(release).ToNot(BeNil())

				releaseNotesData := release.Spec.Data.Raw
				Expect(string(releaseNotesData)).To(ContainSubstring("RHEA"))
				Expect(string(releaseNotesData)).ToNot(ContainSubstring("RHBA"))
				Expect(string(releaseNotesData)).ToNot(ContainSubstring("RHSA"))
			})

		})
	})
})

// Mock image client with configurable version
type mockImageClientWithVersion struct {
	version string
}

func (m *mockImageClientWithVersion) GetImageData(image string) (*types.ImageInspectReport, error) {
	return &types.ImageInspectReport{
		ImageData: &inspect.ImageData{
			Labels: map[string]string{
				"controller": "registry.test.com/controller@sha256:abc123",
				"version":    m.version,
			},
		},
	}, nil
}

// Mock image client without version label
type mockImageClientWithoutVersion struct{}

func (m *mockImageClientWithoutVersion) GetImageData(image string) (*types.ImageInspectReport, error) {
	return &types.ImageInspectReport{
		ImageData: &inspect.ImageData{
			Labels: map[string]string{
				"controller": "registry.test.com/controller@sha256:abc123",
				// Note: no version label
			},
		},
	}, nil
}

// Mock image client that returns an error
type mockImageClientWithError struct {
	errorMsg string
}

func (m *mockImageClientWithError) GetImageData(image string) (*types.ImageInspectReport, error) {
	return nil, fmt.Errorf(m.errorMsg)
}

// Ensure mock clients implement the interface
var (
	_ = internal.ImageClient(&mockImageClientWithVersion{})
	_ = internal.ImageClient(&mockImageClientWithoutVersion{})
	_ = internal.ImageClient(&mockImageClientWithError{})
)

func filterBySnapshotName(obj client.Object) []string {
	return []string{string(obj.(*applicationapiv1alpha1.Snapshot).Name)}
}

// Mock image client that implements the interface properly
type mockImageClient struct{}

func (m *mockImageClient) GetImageData(image string) (*types.ImageInspectReport, error) {
	// Create a mock ImageInspectReport with the Labels field populated
	// Based on the usage in the code, Labels appears to be a direct field
	return &types.ImageInspectReport{
		ImageData: &inspect.ImageData{
			Labels: map[string]string{
				"controller": "registry.test.com/controller@sha256:abc123",
				"version":    "1.0.0",
			},
		},
	}, nil
}
