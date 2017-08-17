package actions_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.ibm.com/Bluemix/kubernetes-cpi/actions"
	"github.ibm.com/Bluemix/kubernetes-cpi/cpi"
	"github.ibm.com/Bluemix/kubernetes-cpi/kubecluster/fakes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/runtime"
	"k8s.io/client-go/testing"
)

var _ = Describe("SetDiskMetadata", func() {
	var (
		fakeClient   *fakes.Client
		fakeProvider *fakes.ClientProvider
		diskCID        cpi.DiskCID
		metadata     map[string]string

		diskMetadataSetter *actions.DiskMetadataSetter
	)

	BeforeEach(func() {
		fakeClient = fakes.NewClient()
		fakeClient.ContextReturns("bosh")
		fakeClient.NamespaceReturns("bosh-namespace")

		fakeProvider = &fakes.ClientProvider{}
		fakeProvider.NewReturns(fakeClient, nil)

		diskCID = actions.NewDiskCID("bosh", "fake-id")
		metadata = map[string]string{
			"director": "bosh",
			"deployment": "cf-kube",
			"instance_index": "0",
			"attached_at": "2017-08-17T03:51:15Z",
			"instance_group": "bosh",
		}

		fakeClient.Clientset = *fake.NewSimpleClientset(
			&v1.PersistentVolumeClaim{ObjectMeta: v1.ObjectMeta{
				Name:      "disk-fake-id",
				Namespace: "bosh-namespace",
				Labels: map[string]string{
					"bosh.cloudfoundry.org/disk-id": "fake-id",
				},
			}},
		)

		diskMetadataSetter = &actions.DiskMetadataSetter{ClientProvider: fakeProvider}
	})

	It("gets a client for the appropriate context", func() {
		err := diskMetadataSetter.SetDiskMetadata(diskCID, metadata)
		Expect(err).NotTo(HaveOccurred())

		Expect(fakeProvider.NewCallCount()).To(Equal(1))
		Expect(fakeProvider.NewArgsForCall(0)).To(Equal("bosh"))
	})

	It("retrieves the disk", func() {
		err := diskMetadataSetter.SetDiskMetadata(diskCID, metadata)
		Expect(err).NotTo(HaveOccurred())

		matches := fakeClient.MatchingActions("get", "persistentvolumeclaims")
		Expect(matches).To(HaveLen(1))

		Expect(matches[0].(testing.GetAction).GetName()).To(Equal("disk-fake-id"))
	})

	It("patches the persistent volume claim with prefixed labels and omits invalid labels", func() {
		err := diskMetadataSetter.SetDiskMetadata(diskCID, metadata)
		Expect(err).NotTo(HaveOccurred())

		matches := fakeClient.MatchingActions("patch", "persistentvolumeclaims")
		Expect(matches).To(HaveLen(1))

		patch := matches[0].(testing.PatchActionImpl)
		Expect(patch.GetName()).To(Equal("disk-fake-id"))
		Expect(patch.GetPatch()).To(MatchJSON(`{
				"metadata": {
					"labels": {
                        "bosh.cloudfoundry.org/attached_at": "2017-08-17T03_51_15Z",
                        "bosh.cloudfoundry.org/deployment": "cf-kube",
                        "bosh.cloudfoundry.org/director": "bosh",
                        "bosh.cloudfoundry.org/instance_group": "bosh",
                        "bosh.cloudfoundry.org/instance_index": "0"
					}
				}
			}`,
		))
	})

	Context("when getting the client fails", func() {
		BeforeEach(func() {
			fakeProvider.NewReturns(nil, errors.New("boom"))
		})

		It("gets a client for the appropriate context", func() {
			err := diskMetadataSetter.SetDiskMetadata(diskCID, metadata)
			Expect(err).To(MatchError("boom"))
		})
	})

	Context("when getting the persistent volume claim fails", func() {
		BeforeEach(func() {
			fakeClient.PrependReactor("get", "persistentvolumeclaims", func(action testing.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("get-pvc-welp")
			})
		})

		It("returns an error", func() {
			err := diskMetadataSetter.SetDiskMetadata(diskCID, metadata)
			Expect(err).To(MatchError("get-pvc-welp"))
			Expect(fakeClient.MatchingActions("get", "persistentvolumeclaims")).To(HaveLen(1))
		})
	})

	Context("when patching the persistent volume claim fails", func() {
		BeforeEach(func() {
			fakeClient.PrependReactor("patch", "persistentvolumeclaims", func(action testing.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("patch-pvc-welp")
			})
		})

		It("returns an error", func() {
			err := diskMetadataSetter.SetDiskMetadata(diskCID, metadata)
			Expect(err).To(MatchError("patch-pvc-welp"))
			Expect(fakeClient.MatchingActions("patch", "persistentvolumeclaims")).To(HaveLen(1))
		})
	})
})
