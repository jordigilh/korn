package release_test

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jordigilh/korn/cmd/waitfor/release"
	"github.com/jordigilh/korn/internal"
	releaseapiv1alpha1 "github.com/konflux-ci/release-service/api/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/urfave/cli/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	dfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("Waitfor Release Command", func() {
	var (
		fakeClientBuilder *fake.ClientBuilder
		scheme            *runtime.Scheme
		ns                *corev1.Namespace
		ctx               context.Context
		cmd               *cli.Command
		fw                *watch.FakeWatcher
	)

	BeforeEach(func() {
		scheme = createFakeScheme()
		ns = newNamespace("test-namespace")
		fakeClientBuilder = fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(ns)

		// Create a fake dynamic client with proper list kinds
		scheme := runtime.NewScheme()
		dynamicClient := dfake.NewSimpleDynamicClient(scheme)
		// Setup a fake watch
		fw = watch.NewFake()

		// Inject the fake watch into the client
		dynamicClient.PrependWatchReactor("releases", func(action k8stesting.Action) (bool, watch.Interface, error) {
			return true, fw, nil
		})
		ctx = context.WithValue(context.Background(), internal.NamespaceCtxType, "test-namespace")
		ctx = context.WithValue(ctx, internal.DynamicCliCtxType, dynamicClient)

		cmd = release.WaitForCommand()
	})

	Context("Wait for release completion", func() {
		DescribeTable("should wait for release successfully",
			func(releases []runtime.Object, releaseName string, expectedError bool, description string) {
				if len(releases) > 0 {
					fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(releases[0])
				}
				ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

				args := []string{"", releaseName}
				// Simulate an ADD event after a short delay
				if len(releases) > 0 {
					go func() {
						for _, r := range releases {
							b, err := json.Marshal(r)
							Expect(err).NotTo(HaveOccurred())
							m := map[string]any{}
							err = json.Unmarshal(b, &m)
							Expect(err).NotTo(HaveOccurred())
							time.Sleep(time.Second)
							fw.Modify(&unstructured.Unstructured{Object: m})
						}
					}()
				}
				err := cmd.Run(ctx, args)

				if expectedError {
					Expect(err).To(HaveOccurred(), description)
				} else {
					Expect(err).ToNot(HaveOccurred(), description)
				}
			},

			Entry("existing successful release",
				[]runtime.Object{
					&releaseapiv1alpha1.Release{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-release",
							Namespace: "test-namespace",
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
							Artifacts: &runtime.RawExtension{
								Raw: []byte(`{"artifacts": []}`),
							},
						},
					},
				},
				"test-release",
				false,
				"Should wait for existing successful release"),

			Entry("existing failed release",
				[]runtime.Object{
					&releaseapiv1alpha1.Release{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-release",
							Namespace: "test-namespace",
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
				},
				"test-release",
				true,
				"Should handle failed release"),

			Entry("non-existing release",
				[]runtime.Object{},
				"non-existing-release",
				true,
				"Should fail for non-existing release"),
			Entry("progressing state",
				[]runtime.Object{
					&releaseapiv1alpha1.Release{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "progressing-release",
							Namespace: "test-namespace",
						},
						Spec: releaseapiv1alpha1.ReleaseSpec{
							Snapshot:    "test-snapshot",
							ReleasePlan: "test-releaseplan",
						},
						Status: releaseapiv1alpha1.ReleaseStatus{
							Conditions: []metav1.Condition{
								{
									Type:   "Released",
									Reason: "Progressing",
									Status: metav1.ConditionUnknown,
								},
							},
						},
					},
					&releaseapiv1alpha1.Release{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "progressing-release",
							Namespace: "test-namespace",
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
							Artifacts: &runtime.RawExtension{
								Raw: []byte(`{"artifacts": []}`),
							},
						},
					},
				},
				"progressing-release",
				false,
				"Should wait for progressing release"),
			Entry("handle release without conditions",
				[]runtime.Object{
					&releaseapiv1alpha1.Release{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-release",
							Namespace: "test-namespace",
						},
						Spec: releaseapiv1alpha1.ReleaseSpec{
							Snapshot:    "test-snapshot",
							ReleasePlan: "test-releaseplan",
						},
						Status: releaseapiv1alpha1.ReleaseStatus{
							Conditions: []metav1.Condition{},
						},
					},
					&releaseapiv1alpha1.Release{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "test-release",
							Namespace: "test-namespace",
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
							Artifacts: &runtime.RawExtension{
								Raw: []byte(`{"artifacts": []}`),
							},
						},
					},
				},
				"test-release",
				false,
				"handle a release without conditions"),
		)
	})

	Context("Edge cases", func() {
		It("should handle missing release name argument", func() {
			ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

			args := []string{"", ""}
			err := cmd.Run(ctx, args)
			Expect(err).To(HaveOccurred())
		})

		It("should handle invalid timeout values", func() {
			releases := []runtime.Object{
				&releaseapiv1alpha1.Release{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-release",
						Namespace: "test-namespace",
					},
					Spec: releaseapiv1alpha1.ReleaseSpec{
						Snapshot:    "test-snapshot",
						ReleasePlan: "test-releaseplan",
					},
				},
			}
			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(releases...)
			ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

			args := []string{"", "test-release", "--timeout", "invalid"}
			err := cmd.Run(ctx, args)
			Expect(err).To(HaveOccurred())
		})

		It("should handle zero timeout", func() {
			releases := []runtime.Object{
				&releaseapiv1alpha1.Release{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-release",
						Namespace: "test-namespace",
					},
					Spec: releaseapiv1alpha1.ReleaseSpec{
						Snapshot:    "test-snapshot",
						ReleasePlan: "test-releaseplan",
					},
				},
			}
			fakeClientBuilder = fakeClientBuilder.WithRuntimeObjects(releases...)
			ctx = context.WithValue(ctx, internal.KubeCliCtxType, fakeClientBuilder.Build())

			args := []string{"", "test-release", "--timeout", "0"}
			err := cmd.Run(ctx, args)
			Expect(err).ToNot(HaveOccurred())
		})
	})

})

func newNamespace(name string) *corev1.Namespace {
	return &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: name}}
}
