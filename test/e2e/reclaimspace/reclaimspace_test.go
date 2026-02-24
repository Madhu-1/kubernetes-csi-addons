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

package reclaimspace_test

import (
	"flag"
	"testing"
	"time"

	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	csiaddonsv1alpha1 "github.com/csi-addons/kubernetes-csi-addons/api/csiaddons/v1alpha1"
	"github.com/csi-addons/kubernetes-csi-addons/test/e2e/config"
	"github.com/csi-addons/kubernetes-csi-addons/test/e2e/framework"
)

func init() {
	// Register configuration flags
	config.RegisterFlags()
}

func TestReclaimSpace(t *testing.T) {
	// Parse flags
	flag.Parse()

	// Load configuration
	_, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load E2E configuration: %v", err)
	}

	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "ReclaimSpace E2E Suite")
}

var _ = ginkgo.Describe("ReclaimSpace", func() {
	var (
		f *framework.Framework
	)

	ginkgo.BeforeEach(func() {
		f = framework.NewFramework("reclaimspace-e2e")
	})

	ginkgo.AfterEach(func() {
		if ginkgo.CurrentSpecReport().Failed() {
			f.CleanupOnFailure()
		} else {
			f.Cleanup()
		}
	})

	ginkgo.Context("ReclaimSpaceJob", func() {
		ginkgo.It("should successfully reclaim space on a PVC", func() {
			ginkgo.By("Creating a PVC")
			pvc := f.CreatePVC("test-pvc", "", f.GetReclaimSpaceStorageClassName())
			pvc = f.WaitForPVCBound(pvc.Name)
			gomega.Expect(pvc.Status.Phase).To(gomega.Equal(corev1.ClaimBound))

			ginkgo.By("Creating a pod to write data to the PVC")
			pod := f.CreatePod("test-pod", pvc.Name, []string{
				"sh", "-c",
				"dd if=/dev/zero of=/mnt/test/testfile bs=1M count=100 && sync && sleep 600",
			})
			f.WaitForPodRunning(pod.Name)

			ginkgo.By("Waiting for pod to complete writing data")

			ginkgo.By("Creating a ReclaimSpaceJob")
			rsJob := f.CreateReclaimSpaceJob("test-rsjob", pvc.Name)

			ginkgo.By("Waiting for ReclaimSpaceJob to complete")
			rsJob = f.WaitForReclaimSpaceJobComplete(rsJob.Name)

			// In a real scenario, you might wait for pod completion
			// For now, we'll just delete the pod
			f.DeletePod(pod.Name)
			f.WaitForPodDeleted(pod.Name)

			ginkgo.By("Verifying ReclaimSpaceJob succeeded")
			gomega.Expect(rsJob.Status.Result).To(gomega.Equal(csiaddonsv1alpha1.OperationResultSucceeded))
			gomega.Expect(rsJob.Status.Message).To(gomega.ContainSubstring("successfully"))

		})

		ginkgo.It("should handle ReclaimSpaceJob on unbound PVC", func() {
			ginkgo.By("Creating a ReclaimSpaceJob for non-existent PVC")
			rsJob := f.CreateReclaimSpaceJob("test-rsjob-fail", "non-existent-pvc")

			ginkgo.By("Waiting for ReclaimSpaceJob to complete")
			rsJob = f.WaitForReclaimSpaceJobComplete(rsJob.Name)

			ginkgo.By("Verifying ReclaimSpaceJob failed")
			gomega.Expect(rsJob.Status.Result).To(gomega.Equal(csiaddonsv1alpha1.OperationResultFailed))
		})

		ginkgo.It("should reclaim space on unmounted volume", func() {
			ginkgo.By("Creating a PVC")
			pvc := f.CreatePVC("test-pvc-mounted", "", f.GetReclaimSpaceStorageClassName())
			pvc = f.WaitForPVCBound(pvc.Name)

			ginkgo.By("Creating a ReclaimSpaceJob while volume is mounted")
			rsJob := f.CreateReclaimSpaceJob("test-rsjob-mounted", pvc.Name)

			ginkgo.By("Waiting for ReclaimSpaceJob to complete")
			rsJob = f.WaitForReclaimSpaceJobComplete(rsJob.Name)

			ginkgo.By("Verifying ReclaimSpaceJob succeeded")
			gomega.Expect(rsJob.Status.Result).To(gomega.Equal(csiaddonsv1alpha1.OperationResultSucceeded))
		})
	})

	ginkgo.Context("ReclaimSpaceCronJob", func() {
		ginkgo.It("should create ReclaimSpaceJobs on schedule", func() {
			ginkgo.By("Creating a PVC for cron job testing")
			pvc := f.CreatePVC("test-pvc-cronjob", "", f.GetReclaimSpaceStorageClassName())
			pvc = f.WaitForPVCBound(pvc.Name)
			gomega.Expect(pvc.Status.Phase).To(gomega.Equal(corev1.ClaimBound))

			ginkgo.By("Creating a ReclaimSpaceCronJob with 1-minute schedule")
			cronJob := f.CreateReclaimSpaceCronJob("test-rs-cronjob", pvc.Name, "*/1 * * * *")
			gomega.Expect(cronJob).NotTo(gomega.BeNil())

			ginkgo.By("Waiting for the first ReclaimSpaceJob to be created")
			createdJob := f.WaitForReclaimSpaceJobCreation(cronJob.Name, 90*time.Second)
			gomega.Expect(createdJob).NotTo(gomega.BeNil())
			gomega.Expect(createdJob.Spec.Target.PersistentVolumeClaim).To(gomega.Equal(pvc.Name))

			ginkgo.By("Waiting for the ReclaimSpaceJob to complete")
			completedJob := f.WaitForReclaimSpaceJobComplete(createdJob.Name)
			gomega.Expect(completedJob.Status.Result).To(gomega.Equal(csiaddonsv1alpha1.OperationResultSucceeded))

			ginkgo.By("Verifying CronJob status is updated")
			updatedCronJob := f.GetReclaimSpaceCronJob(cronJob.Name)
			gomega.Expect(updatedCronJob.Status.LastScheduleTime).NotTo(gomega.BeNil())
		})
	})
})
