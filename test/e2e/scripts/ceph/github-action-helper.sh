#!/usr/bin/env bash

# source https://github.com/rook/rook/blob/v1.19.1/tests/scripts/github-action-helper.sh

set -xeEo pipefail

REPO_DIR="/tmp/rook)"
mkdir -p "${REPO_DIR}"

#############
# VARIABLES #
#############
: "${FUNCTION:=${1}}"

# Architecture detection
ARCH=$(uname -m)
if [ "$ARCH" = "aarch64" ] || [ "$ARCH" = "arm64" ]; then
	ARCH_SUFFIX="arm64"
else
	ARCH_SUFFIX="amd64"
fi

#############
# FUNCTIONS #
#############
function install_deps() {
	sudo wget https://github.com/mikefarah/yq/releases/download/3.4.1/yq_linux_${ARCH_SUFFIX} -O /usr/local/bin/yq
	sudo chmod +x /usr/local/bin/yq
}

function print_k8s_cluster_status() {
	echo "=== Kubernetes Cluster Status ==="
	kubectl cluster-info || true
	kubectl version --short || true
	kubectl get nodes -o wide || true
	kubectl get pods --all-namespaces || true
	echo "================================="
}

# Helper function to retry kubectl commands
function kubectl_retry() {
	local retries=5
	local count=0
	until kubectl "$@"; do
		exit_code=$?
		count=$((count + 1))
		if [ $count -lt $retries ]; then
			echo "kubectl command failed with exit code $exit_code. Retrying in 5 seconds... (attempt $count/$retries)"
			sleep 5
		else
			echo "kubectl command failed after $retries attempts"
			return $exit_code
		fi
	done
}

function block_dev() {
	declare -g DEFAULT_BLOCK_DEV
	: "${DEFAULT_BLOCK_DEV:=/dev/$(block_dev_basename)}"

	echo "$DEFAULT_BLOCK_DEV"
}

function block_dev_basename() {
	declare -g DEFAULT_BLOCK_DEV_BASENAME
	: "${DEFAULT_BLOCK_DEV_BASENAME:=$(find_extra_block_dev)}"

	echo "$DEFAULT_BLOCK_DEV_BASENAME"
}

function create_extra_disk() {
	sudo apt install -y targetcli-fb open-iscsi
	truncate -s 75G ~/iscsi-disk.img
	sudo targetcli /backstores/fileio create disk1 ~/iscsi-disk.img 75G
	local target_iqn=iqn.2026-02.target.local:disk1
	sudo targetcli /iscsi create ${target_iqn}
	sudo targetcli /iscsi/${target_iqn}/tpg1/luns create /backstores/fileio/disk1
	local init_iqn=iqn.2026-02.initiator.local
	echo "InitiatorName=${init_iqn}" | sudo tee /etc/iscsi/initiatorname.iscsi >/dev/null
	sudo targetcli /iscsi/${target_iqn}/tpg1/acls create ${init_iqn}
	sudo targetcli /iscsi/${target_iqn}/tpg1/acls/${init_iqn} create tpg_lun_or_backstore=lun0 mapped_lun=0
	sudo iscsiadm -m discovery -t sendtargets -p 127.0.0.1
	sudo iscsiadm -m node --login
}

function use_local_disk() {
	BLOCK_DATA_PART="$(block_dev)1"
	sudo apt purge snapd -y
	sudo dmsetup version || true
	sudo swapoff --all --verbose
	if mountpoint -q /mnt; then
		sudo umount /mnt
		# search for the device since it keeps changing between sda and sdb
		sudo wipefs --all --force "$BLOCK_DATA_PART"
	else
		# it's the hosted runner!
		sudo sgdisk --zap-all -- "$(block_dev)"
		sudo dd if=/dev/zero of="$(block_dev)" bs=1M count=10 oflag=direct,dsync
		sudo parted -s "$(block_dev)" mklabel gpt
	fi
	sudo lsblk
}

function find_extra_block_dev() {
	# shellcheck disable=SC2005 # redirect doesn't work with sudo, so use echo
	echo "$(sudo lsblk)" >/dev/stderr # print lsblk output to stderr for debugging in case of future errors
	# relevant lsblk --pairs example: (MOUNTPOINT identifies boot partition)(PKNAME is Parent dev ID)
	# NAME="sda15" SIZE="106M" TYPE="part" MOUNTPOINT="/boot/efi" PKNAME="sda"
	# NAME="sdb"   SIZE="75G"  TYPE="disk" MOUNTPOINT=""          PKNAME=""
	# NAME="sdb1"  SIZE="75G"  TYPE="part" MOUNTPOINT="/mnt"      PKNAME="sdb"
	boot_dev="$(sudo lsblk --noheading --list --output MOUNTPOINT,PKNAME | grep boot | awk 'NR==1{print $2}')"
	echo "  == find_extra_block_dev(): boot_dev='$boot_dev'" >/dev/stderr # debug in case of future errors
	# --nodeps ignores partitions
	extra_dev="$(sudo lsblk --noheading --list --nodeps --output KNAME | grep -Ev "($boot_dev|loop|nbd)" | head -1)"
	if [ -z "$extra_dev" ]; then
		create_extra_disk >/dev/stderr
		extra_dev="$(sudo lsblk --noheading --list --nodeps --output KNAME | grep -Ev "($boot_dev|loop|nbd)" | head -1)"
	fi
	echo "  == find_extra_block_dev(): extra_dev='$extra_dev'" >/dev/stderr # debug in case of future errors
	echo "$extra_dev"                                                       # output of function
}

function deploy_first_ceph_cluster() {
	DEVICE_NAME="$(find_extra_block_dev)"
	cd "${REPO_DIR}/deploy/examples"

	yq -i '
  .data.CSI_ENABLE_CSIADDONS = "true" |
  .data.ROOK_CSIADDONS_IMAGE = "quay.io/csiaddons/k8s-sidecar:test"
  ' operator.yaml
	kubectl_retry create -f crds.yaml
	kubectl_retry create -f common.yaml
	kubectl_retry create -f operator.yaml
	kubectl_retry create -f csi-operator.yaml
	yq e -i 'select(.kind != "CephBlockPool")' csi/rbd/storageclass-test.yaml
	kubectl_retry create -f csi/rbd/storageclass-test.yaml

	yq w -i -d0 cluster-test.yaml spec.dashboard.enabled false
	yq w -i -d0 cluster-test.yaml spec.storage.useAllDevices false
	yq w -i -d0 cluster-test.yaml spec.storage.deviceFilter "${DEVICE_NAME}"1
	kubectl_retry create -f cluster-test.yaml
	kubectl_retry create -f toolbox.yaml
	sed -i "/resources:/,/ # priorityClassName:/d" rbdmirror.yaml
	kubectl_retry create -f rbdmirror.yaml

	wait_for_operator_pod_to_be_ready_state
	wait_for_mon rook-ceph
	wait_for_osd_pod_to_be_ready_state rook-ceph
}

function deploy_second_ceph_cluster() {
	DEVICE_NAME="$(find_extra_block_dev)"
	cd "${REPO_DIR}/deploy/examples"
	NAMESPACE=rook-ceph-secondary envsubst <common-second-cluster.yaml | kubectl create -f -
	sed -i 's/namespace: rook-ceph/namespace: rook-ceph-secondary/g' cluster-test.yaml
	yq w -i -d0 cluster-test.yaml spec.storage.deviceFilter "${DEVICE_NAME}"2
	yq w -i -d0 cluster-test.yaml spec.dataDirHostPath "/var/lib/rook-external"
	kubectl_retry create -f cluster-test.yaml
	yq w -i toolbox.yaml metadata.namespace rook-ceph-secondary
	kubectl_retry create -f toolbox.yaml
	sed -i 's/namespace: rook-ceph/namespace: rook-ceph-secondary/g' rbdmirror.yaml
	kubectl_retry create -f rbdmirror.yaml

	wait_for_mon rook-ceph-secondary
	wait_for_osd_pod_to_be_ready_state rook-ceph-secondary
}

function checkout_rook_release() {
	local tag="$1"
	local repo_url="https://github.com/rook/rook.git"
	echo "Cloning Rook repository with tag: ${tag}..."
	git clone \
		--branch "${tag}" \
		--single-branch \
		--depth 1 \
		"${repo_url}" \
		"${REPO_DIR}"
}

wait_for_osd_pod_to_be_ready_state() {
	local namespace="$1"
	timeout 200 bash -c "
		    until [ \$(kubectl get pod -l app=rook-ceph-osd -n \"$namespace\" -o custom-columns=READY:status.containerStatuses[*].ready | grep -c true) -eq 1 ]; do
		      echo \"waiting for the osd pods to be in ready state\"
			  kubectl -n \"$namespace\" get po
		      sleep 1
		    done
	"
	timeout_command_exit_code
}

wait_for_operator_pod_to_be_ready_state() {
	timeout 100 bash -c "
		    until [ \$(kubectl get pod -l app=rook-ceph-operator -n rook-ceph -o custom-columns=READY:status.containerStatuses[*].ready | grep -c true) -eq 1 ]; do
		      echo \"waiting for the operator to be in ready state\"
			  kubectl -n \"$namespace\" get po
		      sleep 1
		    done
	"
	timeout_command_exit_code
}

wait_for_mon() {
	local namespace="$1"
	timeout 150 bash -c "
		    until [ \$(kubectl -n \"$namespace\" get deploy -l app=rook-ceph-mon,mon_canary!=true | grep rook-ceph-mon | wc -l | awk '{print \$1}' ) -eq 1 ]; do
		      echo \"waiting for one mon deployment to exist\"
			  kubectl -n \"$namespace\" get po
		      sleep 2
		    done
	"
	timeout_command_exit_code
}

timeout_command_exit_code() {
	# timeout command return exit status 124 if command times out
	if [ $? -eq 124 ]; then
		echo "Timeout reached"
		exit 1
	fi
}

#######################################
# Enable mirrored pool on clusters
# Arguments:
#   $1 -> primary namespace (e.g. rook-ceph)
#   $2 -> secondary namespace (e.g. rook-ceph-secondary)
#######################################
enable_mirroring_cluster() {
	local PRIMARY_NS="$1"
	local SECONDARY_NS="$2"
	local POOL_NAME="replicapool"

	cd "${REPO_DIR}/deploy/examples"

	echo "Enabling mirroring on primary cluster (${PRIMARY_NS})..."

	yq w -i pool-test.yaml spec.mirroring.enabled true
	yq w -i pool-test.yaml spec.mirroring.mode image
	yq w -i pool-test.yaml metadata.namespace "${PRIMARY_NS}"

	kubectl_retry create -f pool-test.yaml

	echo "Waiting for pool to become Ready on primary cluster..."
	timeout 180 sh -c "until [ \"\$(kubectl -n ${PRIMARY_NS} get cephblockpool ${POOL_NAME} -o jsonpath='{.status.phase}' | grep -c Ready)\" -eq 1 ]; do sleep 1; done"

	yq w -i pool-test.yaml spec.mirroring.enabled true
	yq w -i pool-test.yaml spec.mirroring.mode image
	yq w -i pool-test.yaml metadata.namespace "${SECONDARY_NS}"

	kubectl_retry create -f pool-test.yaml

	echo "Waiting for pool to become Ready on secondary cluster..."
	timeout 180 sh -c "until [ \"\$(kubectl -n ${SECONDARY_NS} get cephblockpool ${POOL_NAME} -o jsonpath='{.status.phase}' | grep -c Ready)\" -eq 1 ]; do sleep 1; done"

	echo "Copying peer token secret to secondary cluster..."

	kubectl_retry -n "${PRIMARY_NS}" get secret pool-peer-token-${POOL_NAME} -o yaml >peer-secret.yaml

	yq delete --inplace peer-secret.yaml metadata.ownerReferences
	yq write --inplace peer-secret.yaml metadata.namespace "${SECONDARY_NS}"
	yq write --inplace peer-secret.yaml metadata.name pool-peer-token-${POOL_NAME}-config

	kubectl_retry create --namespace="${SECONDARY_NS}" -f peer-secret.yaml

	echo "Registering peer secret on secondary cluster..."

	kubectl_retry patch -n "${SECONDARY_NS}" cephblockpool ${POOL_NAME} --type merge \
		-p "{\"spec\":{\"mirroring\":{\"peers\":{\"secretNames\":[\"pool-peer-token-${POOL_NAME}-config\"]}}}}"

	echo "Copying peer token secret to primary cluster..."

	kubectl_retry -n "${SECONDARY_NS}" get secret pool-peer-token-${POOL_NAME} -o yaml >peer-secret.yaml

	yq delete --inplace peer-secret.yaml metadata.ownerReferences
	yq write --inplace peer-secret.yaml metadata.namespace "${PRIMARY_NS}"
	yq write --inplace peer-secret.yaml metadata.name pool-peer-token-${POOL_NAME}-config

	kubectl_retry create --namespace="${PRIMARY_NS}" -f peer-secret.yaml

	echo "Registering peer secret on primary cluster..."

	kubectl_retry patch -n "${PRIMARY_NS}" cephblockpool ${POOL_NAME} --type merge \
		-p "{\"spec\":{\"mirroring\":{\"peers\":{\"secretNames\":[\"pool-peer-token-${POOL_NAME}-config\"]}}}}"

	echo "Verifying mirroring health on both clusters..."
	verify_mirroring_health "${PRIMARY_NS}" "${POOL_NAME}"
	verify_mirroring_health "${SECONDARY_NS}" "${POOL_NAME}"
}

#######################################
# Verify mirroring health for a pool
# Arguments:
#   $1 -> namespace (e.g. rook-ceph or rook-ceph-secondary)
#   $2 -> pool name (e.g. replicapool)
#######################################
verify_mirroring_health() {
	local NAMESPACE="$1"
	local POOL_NAME="$2"
	local TOOLBOX_POD

	echo "Checking mirroring health in namespace ${NAMESPACE}..."

	# Get the toolbox pod name
	TOOLBOX_POD=$(kubectl -n "${NAMESPACE}" get pod -l app=rook-ceph-tools -o jsonpath='{.items[0].metadata.name}')

	if [ -z "${TOOLBOX_POD}" ]; then
		echo "ERROR: Toolbox pod not found in namespace ${NAMESPACE}"
		return 1
	fi

	echo "Using toolbox pod: ${TOOLBOX_POD}"

	# Wait for mirroring to be healthy (timeout after 180 seconds)
	timeout 180 bash -c "
		until kubectl -n \"${NAMESPACE}\" exec \"${TOOLBOX_POD}\" -- rbd mirror pool status ${POOL_NAME} --format=json | jq -e '.summary.health == \"OK\"' > /dev/null 2>&1; do
			echo \"Waiting for mirroring health to be OK in ${NAMESPACE}...\"
			kubectl -n \"${NAMESPACE}\" exec \"${TOOLBOX_POD}\" -- rbd mirror pool status ${POOL_NAME} || true
			sleep 5
		done
	"

	if [ $? -eq 124 ]; then
		echo "ERROR: Timeout waiting for mirroring health to be OK in ${NAMESPACE}"
		kubectl -n "${NAMESPACE}" exec "${TOOLBOX_POD}" -- rbd mirror pool status ${POOL_NAME} || true
		return 1
	fi

	echo "Mirroring health is OK in ${NAMESPACE}"
	kubectl -n "${NAMESPACE}" exec "${TOOLBOX_POD}" -- rbd mirror pool status ${POOL_NAME}
}

########
# MAIN #
########

FUNCTION="$1"
shift # remove function arg now that we've recorded it
# call the function with the remainder of the user-provided args
if ! $FUNCTION "$@"; then
	echo "Call to $FUNCTION was not successful" >&2
	exit 1
fi
