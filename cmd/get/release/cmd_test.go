package release_test

import (
	"context"

	"github.com/jordigilh/korn/cmd/get/release"
	"github.com/jordigilh/korn/internal"
	applicationapiv1alpha1 "github.com/konflux-ci/application-api/api/v1alpha1"
	releaseapiv1alpha1 "github.com/konflux-ci/release-service/api/v1alpha1"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/urfave/cli/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Get Release Command", func() {
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

		cmd = release.GetCommand()
	})

	Context("List all releases", func() {
		DescribeTable("should list releases successfully",
			func(releases []runtime.Object, description string) {
				if len(releases) > 0 {
					fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(releases...)
				}
				ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

				args := []string{""}
				err := cmd.Run(ctx, args)

				Expect(err).ToNot(HaveOccurred(), description)
			},

			Entry("with no releases",
				[]runtime.Object{},
				"Should handle empty release list"),

			Entry("with single successful release",
				[]runtime.Object{
					&releaseapiv1alpha1.Release{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-release",
							Namespace: "test-namespace",
							Labels: map[string]string{
								"appstudio.openshift.io/application": "test-app",
								"appstudio.openshift.io/component":   "test-component",
							},
						},
						Spec: releaseapiv1alpha1.ReleaseSpec{
							Snapshot:    "test-snapshot",
							ReleasePlan: "test-releaseplan",
						},
						Status: releaseapiv1alpha1.ReleaseStatus{
							Conditions: []metav1.Condition{
								{
									Type:   "Released",
									Reason: "Succeeded",
									Status: metav1.ConditionTrue,
								},
							},
						},
					},
				},
				"Should list single successful release"),

			Entry("with multiple releases",
				[]runtime.Object{
					&releaseapiv1alpha1.Release{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "release1",
							Namespace: "test-namespace",
							Labels: map[string]string{
								"appstudio.openshift.io/application": "test-app",
								"appstudio.openshift.io/component":   "test-component",
							},
						},
						Spec: releaseapiv1alpha1.ReleaseSpec{
							Snapshot:    "snapshot1",
							ReleasePlan: "test-releaseplan",
						},
					},
					&releaseapiv1alpha1.Release{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "release2",
							Namespace: "test-namespace",
							Labels: map[string]string{
								"appstudio.openshift.io/application": "test-app",
								"appstudio.openshift.io/component":   "test-component",
							},
						},
						Spec: releaseapiv1alpha1.ReleaseSpec{
							Snapshot:    "snapshot2",
							ReleasePlan: "test-releaseplan",
						},
					},
				},
				"Should list multiple releases"),
		)
	})

	Context("Filter releases with --application flag", func() {
		DescribeTable("should filter releases by application",
			func(releases []runtime.Object, appName string, withError bool, description string) {
				if len(releases) > 0 {
					fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(releases...)
				}
				ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

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
					&releaseapiv1alpha1.Release{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "release1",
							Namespace: "test-namespace",
							Labels: map[string]string{
								"appstudio.openshift.io/application": "test-app",
								"appstudio.openshift.io/component":   "test-component",
							},
						},
						Spec: releaseapiv1alpha1.ReleaseSpec{
							Snapshot:    "snapshot1",
							ReleasePlan: "test-releaseplan",
						},
					},
					&releaseapiv1alpha1.Release{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "release2",
							Namespace: "test-namespace",
							Labels: map[string]string{
								"appstudio.openshift.io/application": "other-app",
								"appstudio.openshift.io/component":   "other-component",
							},
						},
						Spec: releaseapiv1alpha1.ReleaseSpec{
							Snapshot:    "snapshot2",
							ReleasePlan: "other-releaseplan",
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
					&releaseapiv1alpha1.Release{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "release1",
							Namespace: "test-namespace",
							Labels: map[string]string{
								"appstudio.openshift.io/application": "other-app",
								"appstudio.openshift.io/component":   "other-component",
							},
						},
						Spec: releaseapiv1alpha1.ReleaseSpec{
							Snapshot:    "snapshot1",
							ReleasePlan: "other-releaseplan",
						},
					},
				},
				"test-app",
				true,
				"Should return empty list for non-matching application"),
		)
	})

	Context("Get specific release", func() {
		DescribeTable("should get release by name",
			func(releases []runtime.Object, releaseName string, expectError bool, description string) {
				if len(releases) > 0 {
					fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(releases...)
				}
				ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

				args := []string{"", releaseName}
				err := cmd.Run(ctx, args)

				if expectError {
					Expect(err).To(HaveOccurred(), description)
				} else {
					Expect(err).ToNot(HaveOccurred(), description)
				}
			},

			Entry("existing release",
				[]runtime.Object{
					&releaseapiv1alpha1.Release{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-release",
							Namespace: "test-namespace",
							Labels: map[string]string{
								"appstudio.openshift.io/application": "test-app",
								"appstudio.openshift.io/component":   "test-component",
							},
						},
						Spec: releaseapiv1alpha1.ReleaseSpec{
							Snapshot:    "test-snapshot",
							ReleasePlan: "test-releaseplan",
						},
					},
				},
				"test-release",
				false,
				"Should get existing release"),

			Entry("non-existing release",
				[]runtime.Object{},
				"non-existing-release",
				true,
				"Should fail for non-existing release"),
		)
	})

	Context("Flag aliases", func() {
		It("should support the application flag", func() {
			var found bool
			for _, f := range cmd.Flags {
				tmp := f.(*cli.StringFlag)
				if tmp.Name == "application" {
					Expect(tmp.Aliases).To(BeEquivalentTo([]string{"app"}))
					found = true
				}
			}
			Expect(found).To(BeTrue())
		})
	})

	Context("Release status scenarios", func() {
		It("should handle successful releases", func() {
			releases := []runtime.Object{
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
				&releaseapiv1alpha1.Release{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "successful-release",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"appstudio.openshift.io/application": "test-app",
							"appstudio.openshift.io/component":   "test-component",
						},
					},
					Spec: releaseapiv1alpha1.ReleaseSpec{
						Snapshot:    "test-snapshot",
						ReleasePlan: "test-releaseplan",
					},
					Status: releaseapiv1alpha1.ReleaseStatus{
						Conditions: []metav1.Condition{
							{
								Type:   "Released",
								Reason: "Succeeded",
								Status: metav1.ConditionTrue,
							},
						},
					},
				},
			}
			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(releases...)
			ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

			args := []string{"", "--app", "test-app"}
			err := cmd.Run(ctx, args)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should handle failed releases", func() {
			releases := []runtime.Object{
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
				&releaseapiv1alpha1.Release{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "failed-release",
						Namespace: "test-namespace",
						Labels: map[string]string{
							"appstudio.openshift.io/application": "test-app",
							"appstudio.openshift.io/component":   "test-component",
						},
					},
					Spec: releaseapiv1alpha1.ReleaseSpec{
						Snapshot:    "test-snapshot",
						ReleasePlan: "test-releaseplan",
					},
					Status: releaseapiv1alpha1.ReleaseStatus{
						Conditions: []metav1.Condition{
							{
								Type:   "Released",
								Reason: "Failed",
								Status: metav1.ConditionFalse,
							},
						},
					},
				},
			}
			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(releases...)
			ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

			args := []string{"", "--app", "test-app"}
			err := cmd.Run(ctx, args)
			Expect(err).ToNot(HaveOccurred())
		})
	})
})

func newNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
}
