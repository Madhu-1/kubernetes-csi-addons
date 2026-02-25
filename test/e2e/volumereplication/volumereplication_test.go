/*
Copyright 2024 The Kubernetes-CSI-Addons Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package volumereplication_test

import (
	"context"
	"flag"
	"fmt"
	"testing"

	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	corev1 "k8s.io/api/core/v1"

	replicationv1alpha1 "github.com/csi-addons/kubernetes-csi-addons/api/replication.storage/v1alpha1"
	"github.com/csi-addons/kubernetes-csi-addons/test/e2e/config"
	"github.com/csi-addons/kubernetes-csi-addons/test/e2e/framework"
)

func init() {
	// Register configuration flags
	config.RegisterFlags()
}

func TestVolumeReplication(t *testing.T) {
	// Parse flags
	flag.Parse()

	// Load configuration
	_, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load E2E configuration: %v", err)
	}

	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "VolumeReplication E2E Suite")
}

var _ = ginkgo.Describe("VolumeReplication", ginkgo.Ordered, func() {
	var (
		f *framework.Framework
	)

	ginkgo.BeforeAll(func() {
		f = framework.NewFramework("volumereplication-e2e")
	})

	ginkgo.AfterEach(func() {
		// Only cleanup resources created in each test, not the namespace
		if ginkgo.CurrentSpecReport().Failed() {
			f.CleanupOnFailure()
		} else {
			f.Cleanup()
		}
	})

	ginkgo.AfterAll(func() {
		// Cleanup namespace after all tests complete
		if f != nil && f.Namespace != nil && f.Config.DeleteNamespace {
			ginkgo.By(fmt.Sprintf("Deleting namespace %s after all tests", f.Namespace.Name))
			ctx, cancel := context.WithTimeout(context.Background(), f.Config.Timeout)
			defer cancel()
			err := f.Client.Delete(ctx, f.Namespace)
			if err != nil && !apierrors.IsNotFound(err) {
				ginkgo.By(fmt.Sprintf("Warning: Failed to delete namespace: %v", err))
			}
		}
	})

	ginkgo.Context("Primary-Secondary Replication", func() {
		ginkgo.It("should promote volume to primary", func() {
			ginkgo.By("Getting VolumeReplicationClass configuration")
			provisioner := f.GetVolumeReplicationProvisioner()
			gomega.Expect(provisioner).NotTo(gomega.BeEmpty(), "Provisioner must be configured")
			parameters := f.GetVolumeReplicationParameters()

			ginkgo.By("Creating a VolumeReplicationClass")
			vrc := f.CreateVolumeReplicationClass(
				"test-vrc",
				provisioner,
				parameters,
			)

			ginkgo.By("Creating a PVC")
			pvc := f.CreatePVC("test-pvc", "", f.GetVolumeReplicationStorageClassName())
			pvc = f.WaitForPVCBound(pvc.Name)
			gomega.Expect(pvc.Status.Phase).To(gomega.Equal(corev1.ClaimBound))

			ginkgo.By("Creating a VolumeReplication in primary state")
			vr := f.CreateVolumeReplication(
				"test-vr",
				pvc.Name,
				vrc.Name,
				replicationv1alpha1.Primary,
			)

			ginkgo.By("Waiting for VolumeReplication to reach primary state")
			vr = f.WaitForVolumeReplicationState(vr.Name, replicationv1alpha1.PrimaryState)
			gomega.Expect(vr.Status.State).To(gomega.Equal(replicationv1alpha1.PrimaryState))
		})

		ginkgo.It("should demote volume to secondary", func() {
			ginkgo.By("Getting VolumeReplicationClass configuration")
			provisioner := f.GetVolumeReplicationProvisioner()
			gomega.Expect(provisioner).NotTo(gomega.BeEmpty(), "Provisioner must be configured")
			parameters := f.GetVolumeReplicationParameters()

			ginkgo.By("Creating a VolumeReplicationClass")
			vrc := f.CreateVolumeReplicationClass(
				"test-vrc-secondary",
				provisioner,
				parameters,
			)

			ginkgo.By("Creating a PVC")
			pvc := f.CreatePVC("test-pvc-secondary", "", f.GetVolumeReplicationStorageClassName())
			pvc = f.WaitForPVCBound(pvc.Name)

			ginkgo.By("Creating a VolumeReplication in secondary state")
			vr := f.CreateVolumeReplication(
				"test-vr-secondary",
				pvc.Name,
				vrc.Name,
				replicationv1alpha1.Secondary,
			)

			ginkgo.By("Waiting for VolumeReplication to reach secondary state")
			vr = f.WaitForVolumeReplicationState(vr.Name, replicationv1alpha1.SecondaryState)
			gomega.Expect(vr.Status.State).To(gomega.Equal(replicationv1alpha1.SecondaryState))
		})

		ginkgo.It("should transition from primary to secondary", func() {
			ginkgo.By("Getting VolumeReplicationClass configuration")
			provisioner := f.GetVolumeReplicationProvisioner()
			gomega.Expect(provisioner).NotTo(gomega.BeEmpty(), "Provisioner must be configured")
			parameters := f.GetVolumeReplicationParameters()

			ginkgo.By("Creating a VolumeReplicationClass")
			vrc := f.CreateVolumeReplicationClass(
				"test-vrc-transition",
				provisioner,
				parameters,
			)

			ginkgo.By("Creating a PVC")
			pvc := f.CreatePVC("test-pvc-transition", "", f.GetVolumeReplicationStorageClassName())
			pvc = f.WaitForPVCBound(pvc.Name)

			ginkgo.By("Creating a VolumeReplication in primary state")
			vr := f.CreateVolumeReplication(
				"test-vr-transition",
				pvc.Name,
				vrc.Name,
				replicationv1alpha1.Primary,
			)

			ginkgo.By("Waiting for VolumeReplication to reach primary state")
			vr = f.WaitForVolumeReplicationState(vr.Name, replicationv1alpha1.PrimaryState)

			ginkgo.By("Updating VolumeReplication to secondary state")
			vr.Spec.ReplicationState = replicationv1alpha1.Secondary
			f.UpdateVolumeReplication(vr)

			ginkgo.By("Waiting for VolumeReplication to reach secondary state")
			vr = f.WaitForVolumeReplicationState(vr.Name, replicationv1alpha1.SecondaryState)
			gomega.Expect(vr.Status.State).To(gomega.Equal(replicationv1alpha1.SecondaryState))
		})
	})
})
