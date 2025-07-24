package snapshot_test

import (
	"context"
	"fmt"

	"github.com/blang/semver/v4"
	"github.com/containers/podman/v5/pkg/domain/entities/types"
	"github.com/containers/podman/v5/pkg/inspect"
	"github.com/jordigilh/korn/cmd/get/snapshot"
	"github.com/jordigilh/korn/internal"
	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/urfave/cli/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Get Snapshot Command", func() {
	var (
		fakeClientBuilder *fake.ClientBuilder
		scheme            *runtime.Scheme
		ns                *corev1.Namespace
		ctx               context.Context
		cmd               *cli.Command
	)

	BeforeEach(func() {
		scheme = createFakeScheme()
		ns = newNamespace("test-namespace")
		fakeClientBuilder = fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ns)

		ctx = context.WithValue(context.Background(), internal.NamespaceCtxType, "test-namespace")
		ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

		mockPodClient := &mockImageClient{}
		ctx = context.WithValue(ctx, internal.PodmanCliCtxType, mockPodClient)

		mockGitClient := &mockGitClient{}
		ctx = context.WithValue(ctx, internal.GitCliCtxType, mockGitClient)

		cmd = snapshot.GetCommand()
	})

	Context("List all snapshots", func() {
		DescribeTable("should list snapshots successfully",
			func(snapshots []runtime.Object, description string) {
				if len(snapshots) > 0 {
					fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(snapshots...)
				}
				ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

				args := []string{""}
				err := cmd.Run(ctx, args)

				Expect(err).ToNot(HaveOccurred(), description)
			},

			Entry("with no snapshots",
				[]runtime.Object{},
				"Should handle empty snapshot list"),

			Entry("with single successful snapshot",
				[]runtime.Object{
					&applicationapiv1alpha1.Snapshot{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-snapshot",
							Namespace: "test-namespace",
							Labels: map[string]string{
								"pac.test.appstudio.openshift.io/event-type": "push",
								"appstudio.openshift.io/application":         "test-app",
							},
						},
						Spec: applicationapiv1alpha1.SnapshotSpec{
							Application: "test-app",
							Components: []applicationapiv1alpha1.SnapshotComponent{
								{
									Name:           "test-component",
									ContainerImage: "registry.test.com/component@sha256:abc123",
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
					},
				},
				"Should list single successful snapshot"),

			Entry("with multiple snapshots",
				[]runtime.Object{
					&applicationapiv1alpha1.Snapshot{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "snapshot1",
							Namespace: "test-namespace",
							Labels: map[string]string{
								"pac.test.appstudio.openshift.io/event-type": "push",
								"appstudio.openshift.io/application":         "test-app",
							},
						},
						Spec: applicationapiv1alpha1.SnapshotSpec{
							Application: "test-app",
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
					},
					&applicationapiv1alpha1.Snapshot{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "snapshot2",
							Namespace: "test-namespace",
							Labels: map[string]string{
								"pac.test.appstudio.openshift.io/event-type": "push",
								"appstudio.openshift.io/application":         "test-app",
							},
						},
						Spec: applicationapiv1alpha1.SnapshotSpec{
							Application: "test-app",
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
					},
				},
				"Should list multiple snapshots"),
		)
	})

	Context("Filter snapshots with --application flag", func() {
		DescribeTable("should filter snapshots by application",
			func(snapshots []runtime.Object, appName string, withError bool, description string) {
				if len(snapshots) > 0 {
					fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(snapshots...)
					ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())
				}

				args := []string{"", "--app", appName}
				err := cmd.Run(ctx, args)

				if withError {
					Expect(err).To(HaveOccurred(), description)
				} else {
					Expect(err).ToNot(HaveOccurred(), description)
				}
			},

			Entry("with matching application",
				[]runtime.Object{
					&applicationapiv1alpha1.Application{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-app",
							Namespace: "test-namespace",
							Labels: map[string]string{
								"korn.redhat.io/application": "operator",
							},
						},
					},
					&applicationapiv1alpha1.Application{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "other-app",
							Namespace: "test-namespace",
							Labels: map[string]string{
								"korn.redhat.io/application": "fbc",
							},
						},
					},
					&applicationapiv1alpha1.Component{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-bundle",
							Namespace: "test-namespace",
							Labels: map[string]string{
								"korn.redhat.io/component": "bundle",
							},
						},
						Spec: applicationapiv1alpha1.ComponentSpec{
							Application: "test-app",
						},
					},
					&applicationapiv1alpha1.Component{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "other-bundle",
							Namespace: "test-namespace",
							Labels: map[string]string{
								"korn.redhat.io/component": "bundle",
							},
						},
						Spec: applicationapiv1alpha1.ComponentSpec{
							Application: "other-app",
						},
					},
					&applicationapiv1alpha1.Snapshot{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "snapshot1",
							Namespace: "test-namespace",
							Labels: map[string]string{
								"pac.test.appstudio.openshift.io/event-type": "push",
								"appstudio.openshift.io/application":         "test-app",
							},
						},
						Spec: applicationapiv1alpha1.SnapshotSpec{
							Application: "test-app",
						},
					},
					&applicationapiv1alpha1.Snapshot{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "snapshot2",
							Namespace: "test-namespace",
							Labels: map[string]string{
								"pac.test.appstudio.openshift.io/event-type": "push",
								"appstudio.openshift.io/application":         "other-app",
							},
						},
						Spec: applicationapiv1alpha1.SnapshotSpec{
							Application: "other-app",
						},
					},
				},
				"test-app",
				false,
				"Should filter by application name"),

			Entry("with no matching application",
				[]runtime.Object{
					&applicationapiv1alpha1.Application{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "other-app",
							Namespace: "test-namespace",
							Labels: map[string]string{
								"korn.redhat.io/application": "operator",
							},
						},
					},
					&applicationapiv1alpha1.Component{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "other-bundle",
							Namespace: "test-namespace",
							Labels: map[string]string{
								"korn.redhat.io/component": "bundle",
							},
						},
						Spec: applicationapiv1alpha1.ComponentSpec{
							Application: "other-app",
						},
					},
					&applicationapiv1alpha1.Snapshot{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "snapshot1",
							Namespace: "test-namespace",
							Labels: map[string]string{
								"pac.test.appstudio.openshift.io/event-type": "push",
								"appstudio.openshift.io/application":         "other-app",
							},
						},
						Spec: applicationapiv1alpha1.SnapshotSpec{
							Application: "other-app",
						},
					},
				},
				"test-app",
				true,
				"Should return an error for non-matching application"),
		)
	})

	Context("Filter snapshots with --sha flag", func() {
		It("should filter by SHA when provided", func() {
			snapshots := []runtime.Object{
				&applicationapiv1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-app",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"korn.redhat.io/application": "operator",
						},
					},
				},
				&applicationapiv1alpha1.Component{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-bundle",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"korn.redhat.io/component": "bundle",
						},
					},
					Spec: applicationapiv1alpha1.ComponentSpec{
						Application: "test-app",
					},
				},
				&applicationapiv1alpha1.Snapshot{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "snapshot-sha123",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"pac.test.appstudio.openshift.io/event-type": "push",
							"pac.test.appstudio.openshift.io/sha":        "sha123",
							"appstudio.openshift.io/application":         "test-app",
						},
					},
					Spec: applicationapiv1alpha1.SnapshotSpec{
						Application: "test-app",
					},
				},
			}
			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(snapshots...)
			ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

			args := []string{"", "--app", "test-app", "--sha", "sha123"}
			err := cmd.Run(ctx, args)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("Filter snapshots with --version flag", func() {
		It("should filter by version when provided", func() {
			snapshots := []runtime.Object{
				&applicationapiv1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-app",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"korn.redhat.io/application": "operator",
						},
					},
				},
				&applicationapiv1alpha1.Component{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-bundle",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"korn.redhat.io/component": "bundle",
						},
					},
					Spec: applicationapiv1alpha1.ComponentSpec{
						Application: "test-app",
					},
				},
				&applicationapiv1alpha1.Snapshot{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "snapshot-v1.0.0",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"pac.test.appstudio.openshift.io/event-type": "push",
							"appstudio.openshift.io/application":         "test-app",
							"appstudio.openshift.io/component":           "test-bundle",
						},
					},
					Spec: applicationapiv1alpha1.SnapshotSpec{
						Application: "test-app",
						Components: []applicationapiv1alpha1.SnapshotComponent{
							{
								Name: "test-bundle",
								Source: applicationapiv1alpha1.ComponentSource{
									ComponentSourceUnion: applicationapiv1alpha1.ComponentSourceUnion{
										GitSource: &applicationapiv1alpha1.GitSource{
											URL:      "https://github.com/test-app/test-bundle",
											Revision: "v1.0.0",
										},
									},
								},
							},
						},
					},
				},
			}
			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(snapshots...)
			ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

			args := []string{"", "--app", "test-app", "--version", "v1.0.0"}
			err := cmd.Run(ctx, args)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("Get candidate snapshot with --candidate flag", func() {
		It("should get latest candidate snapshot", func() {
			snapshots := []runtime.Object{
				&applicationapiv1alpha1.Application{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-app",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"korn.redhat.io/application": "operator",
						},
					},
				},
				&applicationapiv1alpha1.Component{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-bundle",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"korn.redhat.io/component": "bundle",
						},
					},
					Spec: applicationapiv1alpha1.ComponentSpec{
						Application: "test-app",
					},
				},
				&applicationapiv1alpha1.Snapshot{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "candidate-snapshot",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"pac.test.appstudio.openshift.io/event-type": "push",
							"appstudio.openshift.io/application":         "test-app",
							"appstudio.openshift.io/component":           "test-bundle",
						},
					},
					Spec: applicationapiv1alpha1.SnapshotSpec{
						Application: "test-app",
						Components: []applicationapiv1alpha1.SnapshotComponent{
							{
								Name: "test-bundle",
								Source: applicationapiv1alpha1.ComponentSource{
									ComponentSourceUnion: applicationapiv1alpha1.ComponentSourceUnion{
										GitSource: &applicationapiv1alpha1.GitSource{
											URL:      "https://github.com/test-app/test-bundle",
											Revision: "v1.0.0",
										},
									},
								},
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
				},
			}
			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(snapshots...)
			ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

			args := []string{"", "--app", "test-app", "--candidate"}
			err := cmd.Run(ctx, args)
			Expect(err).ToNot(HaveOccurred())
		})

	})

	Context("Get specific snapshot by name", func() {
		DescribeTable("should get snapshot by name",
			func(snapshots []runtime.Object, snapshotName string, expectError bool, description string) {
				fakeClientBuilder = fake.NewClientBuilder().WithScheme(scheme).WithIndex(&applicationapiv1alpha1.Snapshot{}, "metadata.name", filterBySnapshotName)
				if len(snapshots) > 0 {
					fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(snapshots...)
				}
				ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

				args := []string{"", snapshotName}
				err := cmd.Run(ctx, args)

				if expectError {
					Expect(err).To(HaveOccurred(), description)
				} else {
					Expect(err).ToNot(HaveOccurred(), description)
				}
			},

			Entry("existing snapshot",
				[]runtime.Object{
					&applicationapiv1alpha1.Snapshot{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-snapshot",
							Namespace: "test-namespace",
							Labels: map[string]string{
								"pac.test.appstudio.openshift.io/event-type": "push",
								"appstudio.openshift.io/application":         "test-app",
							},
						},
						Spec: applicationapiv1alpha1.SnapshotSpec{
							Application: "test-app",
						},
					},
				},
				"test-snapshot",
				false,
				"Should get existing snapshot"),

			Entry("non-existing snapshot",
				[]runtime.Object{},
				"non-existing-snapshot",
				true,
				"Should fail for non-existing snapshot"),
		)
	})

	Context("Flag aliases", func() {
		DescribeTable("should work with aliases for each flag", func(flagName, alias string) {
			var found bool
			for _, f := range cmd.Flags {
				switch tmp := f.(type) {
				case *cli.StringFlag:
					if tmp.Name == flagName {
						Expect(tmp.Aliases).To(BeEquivalentTo([]string{alias}))
						found = true
					}
				case *cli.BoolFlag:
					if tmp.Name == flagName {
						Expect(tmp.Aliases).To(BeEquivalentTo([]string{alias}))
						found = true
					}
				}
			}
			Expect(found).To(BeTrue())
		},
			Entry("applicaiton flag", "application", "app"),
			Entry("candidate flag", "candidate", "c"),
		)
	})

})

// Mock image client that implements the ImageClient interface
type mockImageClient struct{}

func (m *mockImageClient) GetImageData(image string) (*types.ImageInspectReport, error) {
	// Create a mock ImageInspectReport with the Labels field populated
	return &types.ImageInspectReport{
		ImageData: &inspect.ImageData{
			Labels: map[string]string{
				"controller": "registry.test.com/controller@sha256:abc123",
				"version":    "1.0.0",
			},
		},
	}, nil
}

// Mock git client that implements the GitCommitVersioner interface
type mockGitClient struct{}

func (m *mockGitClient) GetVersion(repoURL, commitHash string) (*semver.Version, error) {
	// Return a mock version for testing
	version, _ := semver.ParseTolerant("1.0.0")
	return &version, nil
}

func (m *mockGitClient) Cleanup() {
	// No cleanup needed for mock
	fmt.Print("Cleanup called")
}

var (
	_ = internal.GitCommitVersioner(&mockGitClient{})
	_ = internal.ImageClient(&mockImageClient{})
)

func newNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
}

func filterBySnapshotName(obj client.Object) []string {
	return []string{string(obj.(*applicationapiv1alpha1.Snapshot).Name)}
}
