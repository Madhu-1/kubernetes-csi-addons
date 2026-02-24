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

package config

import (
	"flag"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// E2EConfig holds the configuration for e2e tests
type E2EConfig struct {
	// Kubeconfig is the path to the kubeconfig file
	Kubeconfig string `yaml:"kubeconfig"`

	// ReportDir is the directory to store test reports
	ReportDir string `yaml:"reportDir"`

	// Namespace is the namespace to run tests in
	Namespace string `yaml:"namespace"`

	// DeleteNamespace determines if namespace should be deleted after tests
	DeleteNamespace bool `yaml:"deleteNamespace"`

	// DeleteNamespaceOnFailure determines if namespace should be deleted on test failure
	DeleteNamespaceOnFailure bool `yaml:"deleteNamespaceOnFailure"`

	// Timeout is the global timeout for tests
	Timeout time.Duration `yaml:"timeout"`

	// CSIDriver contains CSI driver configuration
	CSIDriver CSIDriverConfig `yaml:"csiDriver"`

	// Storage contains storage configuration
	Storage StorageConfig `yaml:"storage"`

	// NetworkFence contains NetworkFence test configuration
	NetworkFence NetworkFenceConfig `yaml:"networkFence"`

	// VolumeReplication contains VolumeReplication test configuration
	VolumeReplication VolumeReplicationConfig `yaml:"volumeReplication"`

	// VolumeGroupReplication contains VolumeGroupReplication test configuration
	VolumeGroupReplication VolumeGroupReplicationConfig `yaml:"volumeGroupReplication"`

	// Tests contains test suite configuration
	Tests TestsConfig `yaml:"tests"`

	// Timeouts contains operation-specific timeouts
	Timeouts TimeoutsConfig `yaml:"timeouts"`
}

// CSIDriverConfig holds CSI driver configuration
type CSIDriverConfig struct {
	// Name is the name of the CSI driver
	Name string `yaml:"name"`

	// Namespace is the namespace where CSI driver is deployed
	Namespace string `yaml:"namespace"`
}

// StorageConfig holds storage configuration
type StorageConfig struct {
	// StorageClassName is the name of the default storage class to use
	StorageClassName string `yaml:"storageClassName"`

	// VolumeSize is the default size for test volumes
	VolumeSize string `yaml:"volumeSize"`

	// AccessMode is the default access mode for test volumes
	AccessMode string `yaml:"accessMode"`

	// VolumeBindingMode is the volume binding mode
	VolumeBindingMode string `yaml:"volumeBindingMode"`

	// ReclaimSpaceStorageClassName is the storage class for ReclaimSpace tests
	ReclaimSpaceStorageClassName string `yaml:"reclaimSpaceStorageClassName"`

	// EncryptionKeyRotationStorageClassName is the storage class for EncryptionKeyRotation tests
	EncryptionKeyRotationStorageClassName string `yaml:"encryptionKeyRotationStorageClassName"`

	// NetworkFenceStorageClassName is the storage class for NetworkFence tests
	NetworkFenceStorageClassName string `yaml:"networkFenceStorageClassName"`

	// VolumeReplicationStorageClassName is the storage class for VolumeReplication tests
	VolumeReplicationStorageClassName string `yaml:"volumeReplicationStorageClassName"`

	// VolumeGroupReplicationStorageClassName is the storage class for VolumeGroupReplication tests
	VolumeGroupReplicationStorageClassName string `yaml:"volumeGroupReplicationStorageClassName"`
}

// NetworkFenceConfig holds NetworkFence test configuration
type NetworkFenceConfig struct {
	// Cidrs contains the list of CIDR blocks for testing
	Cidrs []string `yaml:"cidrs"`

	// Provisioner is the provisioner name for NetworkFenceClass
	Provisioner string `yaml:"provisioner"`

	// Parameters for NetworkFenceClass
	Parameters map[string]string `yaml:"parameters"`

	// NetworkFenceParameters are parameters specific to NetworkFence resource
	// These are separate from NetworkFenceClass parameters
	NetworkFenceParameters map[string]string `yaml:"networkFenceParameters"`

	// Secret contains the secret configuration for NetworkFence
	Secret SecretConfig `yaml:"secret"`
}

// SecretConfig holds secret configuration
type SecretConfig struct {
	// Name is the name of the secret
	Name string `yaml:"name"`

	// Namespace is the namespace where the secret is located
	Namespace string `yaml:"namespace"`
}

// VolumeReplicationConfig holds VolumeReplication test configuration
type VolumeReplicationConfig struct {
	// Provisioner is the provisioner name for VolumeReplicationClass
	Provisioner string `yaml:"provisioner"`

	// Parameters for VolumeReplicationClass
	Parameters map[string]string `yaml:"parameters"`
}

// VolumeGroupReplicationConfig holds VolumeGroupReplication test configuration
type VolumeGroupReplicationConfig struct {
	// Provisioner is the provisioner name for VolumeGroupReplicationClass
	Provisioner string `yaml:"provisioner"`

	// Parameters for VolumeGroupReplicationClass
	Parameters map[string]string `yaml:"parameters"`
}

// TestsConfig holds test suite configuration
type TestsConfig struct {
	// Skip contains patterns of tests to skip
	Skip []string `yaml:"skip"`

	// Focus contains patterns of tests to focus on
	Focus []string `yaml:"focus"`

	// ReclaimSpace enables ReclaimSpace tests
	ReclaimSpace bool `yaml:"reclaimSpace"`

	// EncryptionKeyRotation enables EncryptionKeyRotation tests
	EncryptionKeyRotation bool `yaml:"encryptionKeyRotation"`

	// NetworkFence enables NetworkFence tests
	NetworkFence bool `yaml:"networkFence"`

	// VolumeReplication enables VolumeReplication tests
	VolumeReplication bool `yaml:"volumeReplication"`

	// VolumeGroupReplication enables VolumeGroupReplication tests
	VolumeGroupReplication bool `yaml:"volumeGroupReplication"`
}

// TimeoutsConfig holds operation-specific timeouts
type TimeoutsConfig struct {
	// PVCCreate is the timeout for PVC creation
	PVCCreate time.Duration `yaml:"pvcCreate"`

	// PVCBound is the timeout for PVC to become bound
	PVCBound time.Duration `yaml:"pvcBound"`

	// PodStart is the timeout for pod to start
	PodStart time.Duration `yaml:"podStart"`

	// PodDelete is the timeout for pod deletion
	PodDelete time.Duration `yaml:"podDelete"`

	// JobComplete is the timeout for job completion
	JobComplete time.Duration `yaml:"jobComplete"`

	// ReplicationSync is the timeout for replication sync
	ReplicationSync time.Duration `yaml:"replicationSync"`

	// OperationComplete is the timeout for generic operations
	OperationComplete time.Duration `yaml:"operationComplete"`
}

var (
	// TestConfig is the global test configuration
	TestConfig *E2EConfig

	// configFile is the path to the config file
	configFile string
)

// RegisterFlags registers command-line flags
func RegisterFlags() {
	flag.StringVar(&configFile, "e2e-config", "", "Path to e2e test configuration file")
}

// LoadConfig loads the configuration from file or creates default
func LoadConfig() (*E2EConfig, error) {
	config := DefaultConfig()

	// If config file is specified, load it
	if configFile != "" {
		data, err := os.ReadFile(configFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read config file: %w", err)
		}

		if err := yaml.Unmarshal(data, config); err != nil {
			return nil, fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Override with environment variables if set
	config.applyEnvironmentOverrides()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	TestConfig = config
	return config, nil
}

// DefaultConfig returns the default configuration
func DefaultConfig() *E2EConfig {
	return &E2EConfig{
		Kubeconfig:               os.Getenv("KUBECONFIG"),
		ReportDir:                "_output/e2e-reports",
		Namespace:                "",
		DeleteNamespace:          true,
		DeleteNamespaceOnFailure: false,
		Timeout:                  30 * time.Minute,
		CSIDriver: CSIDriverConfig{
			Name:      "rook-ceph.rbd.csi.ceph.com", // Madhu Remove this one
			Namespace: "default",
		},
		Storage: StorageConfig{
			StorageClassName:  "rook-ceph-block", // Madhu remove this one
			VolumeSize:        "1Gi",
			AccessMode:        "ReadWriteOnce",
			VolumeBindingMode: "Immediate",
		},
		NetworkFence: NetworkFenceConfig{
			Cidrs:       []string{"192.168.1.0/24", "10.0.0.0/8"},
			Provisioner: "rook-ceph.rbd.csi.ceph.com",
			Parameters: map[string]string{
				"clusterID": "rook-ceph",
				"csiaddons.openshift.io/networkfence-secret-name":      "rook-csi-rbd-provisioner",
				"csiaddons.openshift.io/networkfence-secret-namespace": "rook-ceph",
			},
			NetworkFenceParameters: map[string]string{
				"clusterID": "rook-ceph",
			},
			Secret: SecretConfig{
				Name:      "rook-csi-rbd-provisioner", //Madhu remove this one
				Namespace: "rook-ceph",                //remove this one
			},
		},
		VolumeReplication: VolumeReplicationConfig{
			Provisioner: "",
			Parameters:  map[string]string{},
		},
		VolumeGroupReplication: VolumeGroupReplicationConfig{
			Provisioner: "",
			Parameters:  map[string]string{},
		},
		Tests: TestsConfig{
			Skip:                   []string{},
			Focus:                  []string{},
			ReclaimSpace:           true,
			EncryptionKeyRotation:  true,
			NetworkFence:           true,
			VolumeReplication:      true,
			VolumeGroupReplication: true,
		},
		Timeouts: TimeoutsConfig{
			PVCCreate:         2 * time.Minute,
			PVCBound:          5 * time.Minute,
			PodStart:          5 * time.Minute,
			PodDelete:         2 * time.Minute,
			JobComplete:       10 * time.Minute,
			ReplicationSync:   15 * time.Minute,
			OperationComplete: 5 * time.Minute,
		},
	}
}

// applyEnvironmentOverrides applies environment variable overrides
func (c *E2EConfig) applyEnvironmentOverrides() {
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		c.Kubeconfig = kubeconfig
	}
	if namespace := os.Getenv("E2E_NAMESPACE"); namespace != "" {
		c.Namespace = namespace
	}
	if driverName := os.Getenv("CSI_DRIVER_NAME"); driverName != "" {
		c.CSIDriver.Name = driverName
	}
	if storageClass := os.Getenv("STORAGE_CLASS"); storageClass != "" {
		c.Storage.StorageClassName = storageClass
	}
	if reportDir := os.Getenv("E2E_REPORT_DIR"); reportDir != "" {
		c.ReportDir = reportDir
	}
	// Feature-specific storage class overrides
	if reclaimSpaceSC := os.Getenv("RECLAIMSPACE_STORAGE_CLASS"); reclaimSpaceSC != "" {
		c.Storage.ReclaimSpaceStorageClassName = reclaimSpaceSC
	}
	if encryptionKeyRotationSC := os.Getenv("ENCRYPTIONKEYROTATION_STORAGE_CLASS"); encryptionKeyRotationSC != "" {
		c.Storage.EncryptionKeyRotationStorageClassName = encryptionKeyRotationSC
	}
	if networkFenceSC := os.Getenv("NETWORKFENCE_STORAGE_CLASS"); networkFenceSC != "" {
		c.Storage.NetworkFenceStorageClassName = networkFenceSC
	}
	if volumeReplicationSC := os.Getenv("VOLUMEREPLICATION_STORAGE_CLASS"); volumeReplicationSC != "" {
		c.Storage.VolumeReplicationStorageClassName = volumeReplicationSC
	}
	if volumeGroupReplicationSC := os.Getenv("VOLUMEGROUPREPLICATION_STORAGE_CLASS"); volumeGroupReplicationSC != "" {
		c.Storage.VolumeGroupReplicationStorageClassName = volumeGroupReplicationSC
	}
}

// Validate validates the configuration
func (c *E2EConfig) Validate() error {
	if c.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	return nil
}
