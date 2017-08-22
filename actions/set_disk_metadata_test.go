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
		diskCID      cpi.DiskCID
		metadata     actions.Metadata
		labels       map[string]string
		annotations  map[string]string

		diskMetadataSetter *actions.DiskMetadataSetter
	)

	BeforeEach(func() {
		fakeClient = fakes.NewClient()
		fakeClient.ContextReturns("bosh")
		fakeClient.NamespaceReturns("bosh-namespace")

		fakeProvider = &fakes.ClientProvider{}
		fakeProvider.NewReturns(fakeClient, nil)

		diskCID = actions.NewDiskCID("bosh", "fake-id")
		labels = map[string]string{
			"director":       "bosh",
			"deployment":     "cf-kube",
			"instance_index": "0",
			"attached_at":    "2017-08-17T03:51:15Z",
			"instance_group": "bosh",
		}

		annotations = map[string]string{
			"release_repo": "https://github.com/myorg/myrepo",
		}

		fakeClient.Clientset = *fake.NewSimpleClientset(
			&v1.PersistentVolumeClaim{ObjectMeta: v1.ObjectMeta{
				Name:      "disk-fake-id",
				Namespace: "bosh-namespace",
				Labels: map[string]string{
					"bosh.cloudfoundry.org/disk-id": "fake-id",
				},
				Annotations: map[string]string{
					"bosh.cloudfoundry.org/api_url": "api.ng.bluemix.net",
				},
			}},
		)

		diskMetadataSetter = &actions.DiskMetadataSetter{ClientProvider: fakeProvider}
	})

	JustBeforeEach(func() {
		metadata = actions.Metadata{
			Labels:      labels,
			Annotations: annotations,
		}
	})

	Describe("Setting labels", func() {
		It("Updates the metadata labels", func() {
			pvc, _ := fakeClient.PersistentVolumeClaims().Get("disk-fake-id")
			Expect(pvc.ObjectMeta.Labels).To(HaveLen(1))

			err := diskMetadataSetter.SetDiskMetadata(diskCID, metadata)
			Expect(err).NotTo(HaveOccurred())

			matches := fakeClient.MatchingActions("update", "persistentvolumeclaims")
			Expect(matches).To(HaveLen(1))

			pvc, _ = fakeClient.PersistentVolumeClaims().Get("disk-fake-id")

			Expect(pvc.ObjectMeta.Labels).To(HaveLen(6))
			Expect(pvc.ObjectMeta.Labels["bosh.cloudfoundry.org/attached_at"]).To(Equal("2017-08-17T03_51_15Z"))
			Expect(pvc.ObjectMeta.Labels["bosh.cloudfoundry.org/disk-id"]).To(Equal("fake-id"))
			Expect(pvc.ObjectMeta.Labels["bosh.cloudfoundry.org/director"]).To(Equal("bosh"))
			Expect(pvc.ObjectMeta.Labels["bosh.cloudfoundry.org/deployment"]).To(Equal("cf-kube"))
			Expect(pvc.ObjectMeta.Labels["bosh.cloudfoundry.org/instance_index"]).To(Equal("0"))
			Expect(pvc.ObjectMeta.Labels["bosh.cloudfoundry.org/instance_group"]).To(Equal("bosh"))
		})

		Context("When the disk does not have any prior labels", func() {
			BeforeEach(func() {
				fakeClient.Clientset = *fake.NewSimpleClientset(
					&v1.PersistentVolumeClaim{ObjectMeta: v1.ObjectMeta{
						Name:      "disk-fake-id",
						Namespace: "bosh-namespace",
					}},
				)
			})

			It("Updates the labels", func() {
				err := diskMetadataSetter.SetDiskMetadata(diskCID, metadata)
				Expect(err).NotTo(HaveOccurred())

				pvc, _ := fakeClient.PersistentVolumeClaims().Get("disk-fake-id")
				Expect(pvc.ObjectMeta.Labels).To(HaveLen(5))
				Expect(pvc.ObjectMeta.Labels["bosh.cloudfoundry.org/attached_at"]).To(Equal("2017-08-17T03_51_15Z"))
				Expect(pvc.ObjectMeta.Labels["bosh.cloudfoundry.org/director"]).To(Equal("bosh"))
				Expect(pvc.ObjectMeta.Labels["bosh.cloudfoundry.org/deployment"]).To(Equal("cf-kube"))
				Expect(pvc.ObjectMeta.Labels["bosh.cloudfoundry.org/instance_index"]).To(Equal("0"))
				Expect(pvc.ObjectMeta.Labels["bosh.cloudfoundry.org/instance_group"]).To(Equal("bosh"))
			})
		})

		Context("When the label key is not a qualified name", func() {
			BeforeEach(func() {
				labels["wizzy?wig*"] = "labeltobeskipped"
			})

			It("It does not add any of the requested labels", func() {
				err := diskMetadataSetter.SetDiskMetadata(diskCID, metadata)
				Expect(err).To(HaveOccurred())

				pvc, _ := fakeClient.PersistentVolumeClaims().Get("disk-fake-id")

				Expect(pvc.ObjectMeta.Labels).To(HaveLen(1))
				Expect(pvc.ObjectMeta.Labels["bosh.cloudfoundry.org/disk-id"]).To(Equal("fake-id"))
				_, exists := pvc.ObjectMeta.Labels["bosh.cloudfoundry.org/deployment"]

				Expect(exists).To(BeFalse())
			})

			It("Returns a descriptive error message", func() {
				err := diskMetadataSetter.SetDiskMetadata(diskCID, metadata)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Error setting disk metadata: label"))
			})
		})

		Context("When the label value is not valid", func() {
			BeforeEach(func() {
				labels["valid_key"] = "This l4bel might be skipped!"
			})

			It("It does not add any of the requested labels", func() {
				err := diskMetadataSetter.SetDiskMetadata(diskCID, metadata)
				Expect(err).To(HaveOccurred())

				pvc, _ := fakeClient.PersistentVolumeClaims().Get("disk-fake-id")

				Expect(pvc.ObjectMeta.Labels).To(HaveLen(1))
				Expect(pvc.ObjectMeta.Labels["bosh.cloudfoundry.org/disk-id"]).To(Equal("fake-id"))
				_, exists := pvc.ObjectMeta.Labels["bosh.cloudfoundry.org/deployment"]

				Expect(exists).To(BeFalse())
			})

			It("Returns a descriptive error message", func() {
				err := diskMetadataSetter.SetDiskMetadata(diskCID, metadata)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Error setting disk metadata: label"))
			})
		})
	})

	Describe("Setting annotations", func() {
		It("Updates the annotations", func() {
			err := diskMetadataSetter.SetDiskMetadata(diskCID, metadata)
			Expect(err).NotTo(HaveOccurred())

			matches := fakeClient.MatchingActions("update", "persistentvolumeclaims")
			Expect(matches).To(HaveLen(1))

			pvc := matches[0].(testing.UpdateAction).GetObject().(*v1.PersistentVolumeClaim)

			Expect(pvc.ObjectMeta.Annotations).To(HaveLen(2))
			Expect(pvc.ObjectMeta.Annotations["release_repo"]).To(Equal("https://github.com/myorg/myrepo"))
			Expect(pvc.ObjectMeta.Annotations["bosh.cloudfoundry.org/api_url"]).To(Equal("api.ng.bluemix.net"))
		})

		Context("When the disk does not have any prior annotations", func() {
			BeforeEach(func() {
				fakeClient.Clientset = *fake.NewSimpleClientset(
					&v1.PersistentVolumeClaim{ObjectMeta: v1.ObjectMeta{
						Name:      "disk-fake-id",
						Namespace: "bosh-namespace",
						Labels: map[string]string{
							"bosh.cloudfoundry.org/disk-id": "fake-id",
						},
					}},
				)
			})

			It("Updates the annotations", func() {
				err := diskMetadataSetter.SetDiskMetadata(diskCID, metadata)
				Expect(err).NotTo(HaveOccurred())

				matches := fakeClient.MatchingActions("update", "persistentvolumeclaims")
				Expect(matches).To(HaveLen(1))

				pvc := matches[0].(testing.UpdateAction).GetObject().(*v1.PersistentVolumeClaim)

				Expect(pvc.ObjectMeta.Annotations).To(HaveLen(1))
				Expect(pvc.ObjectMeta.Annotations["release_repo"]).To(Equal("https://github.com/myorg/myrepo"))

				pvc, _ = fakeClient.PersistentVolumeClaims().Get("disk-fake-id")
				Expect(pvc.ObjectMeta.Annotations).To(HaveLen(1))
				Expect(pvc.ObjectMeta.Annotations["release_repo"]).To(Equal("https://github.com/myorg/myrepo"))
			})
		})

		Context("When the annotation key is not a qualified name", func() {
			BeforeEach(func() {
				annotations["not_a_valid_dns_subdomain?/some_org"] = "annotation-to-be-skipped"
			})

			It("It does not add any of the requested annotations", func() {
				err := diskMetadataSetter.SetDiskMetadata(diskCID, metadata)
				Expect(err).To(HaveOccurred())

				matches := fakeClient.MatchingActions("update", "persistentvolumeclaims")
				Expect(matches).To(HaveLen(0))

				pvc, _ := fakeClient.PersistentVolumeClaims().Get("disk-fake-id")

				Expect(pvc.ObjectMeta.Annotations).To(HaveLen(1))
				Expect(pvc.ObjectMeta.Annotations["bosh.cloudfoundry.org/api_url"]).To(Equal("api.ng.bluemix.net"))
				_, exists := pvc.ObjectMeta.Annotations["bosh.cloudfoundry.org/release_repo"]

				Expect(exists).To(BeFalse())
			})

			It("Returns a descriptive error message", func() {
				err := diskMetadataSetter.SetDiskMetadata(diskCID, metadata)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Error setting disk metadata: annotation"))
			})
		})
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

	Context("when updating the persistent volume claim fails", func() {
		BeforeEach(func() {
			fakeClient.PrependReactor("update", "persistentvolumeclaims", func(action testing.Action) (bool, runtime.Object, error) {
				return true, nil, errors.New("update-pvc-welp")
			})
		})

		It("returns an error", func() {
			err := diskMetadataSetter.SetDiskMetadata(diskCID, metadata)
			Expect(err).To(MatchError("update-pvc-welp"))
			Expect(fakeClient.MatchingActions("update", "persistentvolumeclaims")).To(HaveLen(1))
		})
	})
})
