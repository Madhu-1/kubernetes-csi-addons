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

package networkfence_test

import (
	"flag"
	"testing"

	ginkgo "github.com/onsi/ginkgo/v2"
	gomega "github.com/onsi/gomega"

	csiaddonsv1alpha1 "github.com/csi-addons/kubernetes-csi-addons/api/csiaddons/v1alpha1"
	"github.com/csi-addons/kubernetes-csi-addons/test/e2e/config"
	"github.com/csi-addons/kubernetes-csi-addons/test/e2e/framework"
)

func init() {
	// Register configuration flags
	config.RegisterFlags()
}

func TestNetworkFence(t *testing.T) {
	// Parse flags
	flag.Parse()

	// Load configuration
	_, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load E2E configuration: %v", err)
	}

	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "NetworkFence E2E Suite")
}

var _ = ginkgo.Describe("NetworkFence", func() {
	var (
		f *framework.Framework
	)

	ginkgo.BeforeEach(func() {
		f = framework.NewFramework("networkfence-e2e")
	})

	ginkgo.AfterEach(func() {
		if ginkgo.CurrentSpecReport().Failed() {
			f.CleanupOnFailure()
		} else {
			f.Cleanup()
		}
	})

	ginkgo.Context("NetworkFence Operations", func() {
		ginkgo.It("should fence network access with CIDRs from config", func() {
			ginkgo.By("Getting CIDRs from configuration")
			cidrs := f.GetNetworkFenceCIDRs()
			gomega.Expect(cidrs).NotTo(gomega.BeEmpty(), "CIDRs must be configured in e2e-config.yaml")

			ginkgo.By("Creating a NetworkFence with Fenced state")
			nf := f.CreateNetworkFence(
				"test-nf-fenced",
				cidrs,
				csiaddonsv1alpha1.Fenced,
			)

			ginkgo.By("Verifying NetworkFence was created")
			gomega.Expect(nf.Name).To(gomega.Equal("test-nf-fenced"))
			gomega.Expect(nf.Spec.FenceState).To(gomega.Equal(csiaddonsv1alpha1.Fenced))
			gomega.Expect(nf.Spec.Cidrs).To(gomega.Equal(cidrs))

			ginkgo.By("Waiting for NetworkFence operation to complete")
			nf = f.WaitForNetworkFenceResult(nf.Name, csiaddonsv1alpha1.FencingOperationResultSucceeded)
			gomega.Expect(nf.Status.Result).To(gomega.Equal(csiaddonsv1alpha1.FencingOperationResultSucceeded))
			gomega.Expect(nf.Status.Message).To(gomega.ContainSubstring("fencing operation successful"))
		})

		ginkgo.It("should unfence network access", func() {
			ginkgo.By("Getting CIDRs from configuration")
			cidrs := f.GetNetworkFenceCIDRs()
			gomega.Expect(cidrs).NotTo(gomega.BeEmpty(), "CIDRs must be configured in e2e-config.yaml")

			ginkgo.By("Creating a NetworkFence with Unfenced state")
			nf := f.CreateNetworkFence(
				"test-nf-unfenced",
				cidrs,
				csiaddonsv1alpha1.Unfenced,
			)

			ginkgo.By("Verifying NetworkFence was created")
			gomega.Expect(nf.Name).To(gomega.Equal("test-nf-unfenced"))
			gomega.Expect(nf.Spec.FenceState).To(gomega.Equal(csiaddonsv1alpha1.Unfenced))

			ginkgo.By("Waiting for NetworkFence operation to complete")
			nf = f.WaitForNetworkFenceResult(nf.Name, csiaddonsv1alpha1.FencingOperationResultSucceeded)
			gomega.Expect(nf.Status.Result).To(gomega.Equal(csiaddonsv1alpha1.FencingOperationResultSucceeded))
			gomega.Expect(nf.Status.Message).To(gomega.ContainSubstring("unfencing operation successful"))
		})

		ginkgo.It("should handle multiple CIDR blocks from config", func() {
			ginkgo.By("Getting CIDRs from configuration")
			cidrs := f.GetNetworkFenceCIDRs()
			gomega.Expect(cidrs).NotTo(gomega.BeEmpty(), "CIDRs must be configured in e2e-config.yaml")

			ginkgo.By("Creating a NetworkFence with multiple CIDRs")
			nf := f.CreateNetworkFence(
				"test-nf-multiple-cidrs",
				cidrs,
				csiaddonsv1alpha1.Fenced,
			)

			ginkgo.By("Verifying all CIDRs are configured")
			gomega.Expect(nf.Spec.Cidrs).To(gomega.HaveLen(len(cidrs)))
			gomega.Expect(nf.Spec.Cidrs).To(gomega.Equal(cidrs))

			ginkgo.By("Waiting for NetworkFence operation to complete")
			nf = f.WaitForNetworkFenceResult(nf.Name, csiaddonsv1alpha1.FencingOperationResultSucceeded)
			gomega.Expect(nf.Status.Result).To(gomega.Equal(csiaddonsv1alpha1.FencingOperationResultSucceeded))
		})

		ginkgo.It("should transition from fenced to unfenced", func() {
			ginkgo.By("Getting CIDRs from configuration")
			cidrs := f.GetNetworkFenceCIDRs()
			gomega.Expect(cidrs).NotTo(gomega.BeEmpty(), "CIDRs must be configured in e2e-config.yaml")

			ginkgo.By("Creating a NetworkFence with Fenced state")
			nf := f.CreateNetworkFence(
				"test-nf-transition",
				cidrs,
				csiaddonsv1alpha1.Fenced,
			)

			ginkgo.By("Waiting for fencing operation to complete")
			nf = f.WaitForNetworkFenceResult(nf.Name, csiaddonsv1alpha1.FencingOperationResultSucceeded)
			gomega.Expect(nf.Status.Result).To(gomega.Equal(csiaddonsv1alpha1.FencingOperationResultSucceeded))
			gomega.Expect(nf.Status.Message).To(gomega.ContainSubstring("fencing operation successful"))

			ginkgo.By("Updating NetworkFence to Unfenced state")
			nf.Spec.FenceState = csiaddonsv1alpha1.Unfenced
			nf = f.UpdateNetworkFence(nf)
			gomega.Expect(nf.Spec.FenceState).To(gomega.Equal(csiaddonsv1alpha1.Unfenced))

			ginkgo.By("Waiting for unfencing operation to complete")
			nf = f.WaitForNetworkFenceResult(nf.Name, csiaddonsv1alpha1.FencingOperationResultSucceeded)
			gomega.Expect(nf.Status.Result).To(gomega.Equal(csiaddonsv1alpha1.FencingOperationResultSucceeded))
			gomega.Expect(nf.Status.Message).To(gomega.ContainSubstring("unfencing operation successful"))
		})
	})

	ginkgo.Context("NetworkFence with Secrets", func() {
		ginkgo.It("should create NetworkFence with secret configuration", func() {
			ginkgo.By("Getting secret configuration from config")
			secretConfig := f.GetNetworkFenceSecret()

			// Skip test if no secret is configured
			if secretConfig.Name == "" || secretConfig.Namespace == "" {
				ginkgo.Skip("Secret configuration not provided in e2e-config.yaml")
			}

			ginkgo.By("Getting CIDRs from configuration")
			cidrs := f.GetNetworkFenceCIDRs()
			gomega.Expect(cidrs).NotTo(gomega.BeEmpty(), "CIDRs must be configured in e2e-config.yaml")

			ginkgo.By("Creating a NetworkFence with secret")
			nf := f.CreateNetworkFence(
				"test-nf-with-secret",
				cidrs,
				csiaddonsv1alpha1.Fenced,
			)

			ginkgo.By("Verifying NetworkFence was created with secret")
			gomega.Expect(nf.Name).To(gomega.Equal("test-nf-with-secret"))
			gomega.Expect(nf.Spec.FenceState).To(gomega.Equal(csiaddonsv1alpha1.Fenced))
			gomega.Expect(nf.Spec.Secret.Name).To(gomega.Equal(secretConfig.Name))
			gomega.Expect(nf.Spec.Secret.Namespace).To(gomega.Equal(secretConfig.Namespace))

			ginkgo.By("Waiting for NetworkFence operation to complete")
			nf = f.WaitForNetworkFenceResult(nf.Name, csiaddonsv1alpha1.FencingOperationResultSucceeded)
			gomega.Expect(nf.Status.Result).To(gomega.Equal(csiaddonsv1alpha1.FencingOperationResultSucceeded))
			gomega.Expect(nf.Status.Message).To(gomega.ContainSubstring("fencing operation successful"))
		})

		ginkgo.It("should transition NetworkFence with secret from fenced to unfenced", func() {
			ginkgo.By("Getting secret configuration from config")
			secretConfig := f.GetNetworkFenceSecret()

			// Skip test if no secret is configured
			if secretConfig.Name == "" || secretConfig.Namespace == "" {
				ginkgo.Skip("Secret configuration not provided in e2e-config.yaml")
			}

			ginkgo.By("Getting CIDRs from configuration")
			cidrs := f.GetNetworkFenceCIDRs()
			gomega.Expect(cidrs).NotTo(gomega.BeEmpty(), "CIDRs must be configured in e2e-config.yaml")

			ginkgo.By("Creating a NetworkFence with secret in Fenced state")
			nf := f.CreateNetworkFence(
				"test-nf-secret-transition",
				cidrs,
				csiaddonsv1alpha1.Fenced,
			)

			ginkgo.By("Verifying secret is configured")
			gomega.Expect(nf.Spec.Secret.Name).To(gomega.Equal(secretConfig.Name))
			gomega.Expect(nf.Spec.Secret.Namespace).To(gomega.Equal(secretConfig.Namespace))

			ginkgo.By("Waiting for fencing operation to complete")
			nf = f.WaitForNetworkFenceResult(nf.Name, csiaddonsv1alpha1.FencingOperationResultSucceeded)
			gomega.Expect(nf.Status.Result).To(gomega.Equal(csiaddonsv1alpha1.FencingOperationResultSucceeded))

			ginkgo.By("Updating NetworkFence to Unfenced state")
			nf.Spec.FenceState = csiaddonsv1alpha1.Unfenced
			nf = f.UpdateNetworkFence(nf)
			gomega.Expect(nf.Spec.FenceState).To(gomega.Equal(csiaddonsv1alpha1.Unfenced))

			ginkgo.By("Waiting for unfencing operation to complete")
			nf = f.WaitForNetworkFenceResult(nf.Name, csiaddonsv1alpha1.FencingOperationResultSucceeded)
			gomega.Expect(nf.Status.Result).To(gomega.Equal(csiaddonsv1alpha1.FencingOperationResultSucceeded))
			gomega.Expect(nf.Status.Message).To(gomega.ContainSubstring("unfencing operation successful"))
		})
	})

	ginkgo.Context("NetworkFenceClass", func() {
		ginkgo.It("should use NetworkFenceClass for configuration", func() {
			ginkgo.By("Getting NetworkFenceClass configuration")
			provisioner := f.GetNetworkFenceProvisioner()
			gomega.Expect(provisioner).NotTo(gomega.BeEmpty(), "Provisioner must be configured")
			parameters := f.GetNetworkFenceParameters()
			cidrs := f.GetNetworkFenceCIDRs()
			gomega.Expect(cidrs).NotTo(gomega.BeEmpty(), "CIDRs must be configured in e2e-config.yaml")

			ginkgo.By("Creating a NetworkFenceClass")
			nfc := f.CreateNetworkFenceClass(
				"test-nf-class",
				provisioner,
				parameters,
			)
			gomega.Expect(nfc.Name).To(gomega.Equal("test-nf-class"))
			gomega.Expect(nfc.Spec.Provisioner).To(gomega.Equal(provisioner))

			ginkgo.By("Creating a NetworkFence using NetworkFenceClass")
			nf := f.CreateNetworkFenceWithClass(
				"test-nf-with-class",
				cidrs,
				csiaddonsv1alpha1.Fenced,
				nfc.Name,
			)

			ginkgo.By("Verifying NetworkFence was created with class")
			gomega.Expect(nf.Spec.NetworkFenceClassName).To(gomega.Equal(nfc.Name))
			gomega.Expect(nf.Spec.Cidrs).To(gomega.Equal(cidrs))
			gomega.Expect(nf.Spec.FenceState).To(gomega.Equal(csiaddonsv1alpha1.Fenced))

			ginkgo.By("Waiting for NetworkFence operation to complete")
			nf = f.WaitForNetworkFenceResult(nf.Name, csiaddonsv1alpha1.FencingOperationResultSucceeded)
			gomega.Expect(nf.Status.Result).To(gomega.Equal(csiaddonsv1alpha1.FencingOperationResultSucceeded))
		})

		ginkgo.It("should transition NetworkFence with class from fenced to unfenced", func() {
			ginkgo.By("Getting NetworkFenceClass configuration")
			provisioner := f.GetNetworkFenceProvisioner()
			gomega.Expect(provisioner).NotTo(gomega.BeEmpty(), "Provisioner must be configured")
			parameters := f.GetNetworkFenceParameters()
			cidrs := f.GetNetworkFenceCIDRs()
			gomega.Expect(cidrs).NotTo(gomega.BeEmpty(), "CIDRs must be configured in e2e-config.yaml")

			ginkgo.By("Creating a NetworkFenceClass")
			nfc := f.CreateNetworkFenceClass(
				"test-nf-class-transition",
				provisioner,
				parameters,
			)

			ginkgo.By("Creating a NetworkFence with Fenced state using class")
			nf := f.CreateNetworkFenceWithClass(
				"test-nf-class-transition",
				cidrs,
				csiaddonsv1alpha1.Fenced,
				nfc.Name,
			)

			ginkgo.By("Waiting for fencing operation to complete")
			nf = f.WaitForNetworkFenceResult(nf.Name, csiaddonsv1alpha1.FencingOperationResultSucceeded)
			gomega.Expect(nf.Status.Result).To(gomega.Equal(csiaddonsv1alpha1.FencingOperationResultSucceeded))

			ginkgo.By("Updating NetworkFence to Unfenced state")
			nf.Spec.FenceState = csiaddonsv1alpha1.Unfenced
			nf = f.UpdateNetworkFence(nf)

			ginkgo.By("Waiting for unfencing operation to complete")
			nf = f.WaitForNetworkFenceResult(nf.Name, csiaddonsv1alpha1.FencingOperationResultSucceeded)
			gomega.Expect(nf.Status.Result).To(gomega.Equal(csiaddonsv1alpha1.FencingOperationResultSucceeded))
			gomega.Expect(nf.Status.Message).To(gomega.ContainSubstring("unfencing operation successful"))
		})
	})
})
