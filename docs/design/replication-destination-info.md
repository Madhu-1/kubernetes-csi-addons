# Design: Replication Destination Info

## Problem Statement

Some storage providers have different volume IDs and volume group IDs on the
source and destination sides of a replication relationship. The current spec
and implementation provide no mechanism for the storage provider (SP) to
communicate destination details back to the Container Orchestrator (CO).

For volume group replication with dynamic grouping, this problem is
compounded: `EnableVolumeReplication` is called once for the group, but group
membership changes dynamically (volumes added/removed via label selectors).
The destination group membership evolves without any lifecycle RPC being
called, so the CO has no way to discover updated destination volume/group
mappings.

## Proposal

Introduce a new RPC `GetReplicationDestinationInfo` with a corresponding
capability `GET_REPLICATION_DESTINATION_INFO`. This RPC can be called at any
time to retrieve the current destination volume or volume group details,
including per-volume ID mappings for dynamic groups.

## Spec Changes (csi-addons/spec)

### 1. New Capability: `GET_REPLICATION_DESTINATION_INFO`

A new capability enum value is added under `VolumeReplication.Type` in the
`Capability` message:

```protobuf
message VolumeReplication {
  enum Type {
    UNKNOWN = 0;
    VOLUME_REPLICATION = 1;
    // GET_REPLICATION_DESTINATION_INFO indicates that the CSI-driver
    // supports getting the destination volume or volume group details
    // for a replication relationship. This capability is relevant for
    // storage providers where source and destination volume or volume
    // group identifiers differ.
    GET_REPLICATION_DESTINATION_INFO = 2;
  }
  Type type = 1;
}
```

This capability is OPTIONAL. Storage providers where source and destination
volume IDs are always identical do not need to advertise this capability.
When the CO detects this capability, it knows that the driver supports the
`GetReplicationDestinationInfo` RPC and will call it after replication
operations to retrieve destination-side identifiers.

### 2. New RPC: `GetReplicationDestinationInfo`

A new RPC is added to the `Controller` service:

```protobuf
service Controller {
  // ... existing RPCs ...
  // GetReplicationDestinationInfo RPC call to get the destination
  // volume or volume group details for an existing replication.
  rpc GetReplicationDestinationInfo (GetReplicationDestinationInfoRequest)
  returns (GetReplicationDestinationInfoResponse) {}
}
```

### 3. New Messages

#### GetReplicationDestinationInfoRequest

```protobuf
// GetReplicationDestinationInfoRequest holds the required information to
// get the destination volume or volume group details for an existing
// replication.
message GetReplicationDestinationInfoRequest {
  // The source volume or volume group for which destination
  // details are requested.
  // This field is REQUIRED.
  ReplicationSource replication_source = 1;
  // Secrets required by the plugin to complete the request.
  map<string, string> secrets = 2 [(csi.v1.csi_secret) = true];
}
```

#### GetReplicationDestinationInfoResponse

```protobuf
// GetReplicationDestinationInfoResponse holds the destination volume
// or volume group details for an existing replication.
message GetReplicationDestinationInfoResponse {
  // The destination details for the replication.
  // This field is REQUIRED.
  ReplicationDestination replication_destination = 1;
}
```

#### ReplicationDestination

```protobuf
// Specifies the destination details for a replication. One of the
// type fields MUST be specified.
message ReplicationDestination {
  // VolumeDestination contains the destination details for a
  // replicated volume.
  message VolumeDestination {
    // The destination volume ID on the remote/target cluster.
    // This field is REQUIRED.
    string volume_id = 1;
  }
  // VolumeGroupDestination contains the destination details for a
  // replicated volume group.
  message VolumeGroupDestination {
    // The destination volume group ID on the remote/target cluster.
    // This field is REQUIRED.
    string volume_group_id = 1;
    // Mapping of source volume IDs to their corresponding
    // destination volume IDs. Key is source volume_id,
    // value is destination volume_id.
    // This field is OPTIONAL.
    // This map reflects the current group membership at the
    // time of the call, accounting for dynamic group changes.
    map<string, string> volume_ids = 2;
  }

  oneof type {
    // Volume destination type
    VolumeDestination volume = 1;
    // Volume group destination type
    VolumeGroupDestination volumegroup = 2;
  }
}
```

### 4. Error Scheme for `GetReplicationDestinationInfo`

| Condition                                | gRPC Code             | Description                                                                        | Recovery Behavior                                                                                                     |
| ---------------------------------------- | --------------------- | ---------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------- |
| Missing required field                   | 3 INVALID_ARGUMENT    | A required field is missing from the request.                                      | Caller MUST fix the request by adding the missing required field before retrying.                                     |
| Replication Source does not exist        | 5 NOT_FOUND           | The specified source does not exist.                                               | Caller MUST verify that the `replication_source` is correct and accessible before retrying with exponential back off. |
| Replication Source is not replicated     | 9 FAILED_PRECONDITION | Destination information could not be retrieved because replication is not enabled. | Caller SHOULD ensure that replication is enabled on the `replication_source`.                                         |
| Operation pending for Replication Source | 10 ABORTED            | There is already an operation pending for the specified `replication_source`.      | Caller SHOULD ensure no other calls are pending for the `replication_source`, then retry with exponential back off.   |
| Call not implemented                     | 12 UNIMPLEMENTED      | The invoked RPC is not implemented by the Plugin.                                  | Caller MUST NOT retry.                                                                                                |
| Not authenticated                        | 16 UNAUTHENTICATED    | The invoked RPC does not carry valid secrets for authentication.                   | Caller SHALL fix or regalvanize the secrets, then retry.                                                              |
| Error is Unknown                         | 2 UNKNOWN             | An unknown error occurred.                                                         | Caller MUST study the logs before retrying.                                                                           |

## kubernetes-csi-addons Changes

### 1. Proto Definition

Add the RPC to the `Replication` service and add the following messages:

```protobuf
// GetReplicationDestinationInfoRequest holds the required information to
// get the destination volume or volume group details for an existing
// replication.
message GetReplicationDestinationInfoRequest {
  // The source volume or volume group for which destination
  // details are requested.
  // This field is REQUIRED.
  ReplicationSource replication_source = 1;
  // Secrets required by the plugin to complete the request.
  string secret_name = 2;
  string secret_namespace = 3;
 
}

// GetReplicationDestinationInfoResponse holds the destination volume
// or volume group details for an existing replication.
message GetReplicationDestinationInfoResponse {
  // The destination details for the replication.
  // This field is REQUIRED.
  ReplicationDestination replication_destination = 1;
}

// Specifies the destination details for a replication. One of the
// type fields MUST be specified.
message ReplicationDestination {
  // VolumeDestination contains the destination details for a
  // replicated volume.
  message VolumeDestination {
    // The destination volume ID on the remote/target cluster.
    // This field is REQUIRED.
    string volume_id = 1;
  }
  // VolumeGroupDestination contains the destination details for a
  // replicated volume group.
  message VolumeGroupDestination {
    // The destination volume group ID on the remote/target cluster.
    // This field is REQUIRED.
    string volume_group_id = 1;
    // Mapping of source volume IDs to their corresponding
    // destination volume IDs. Key is source volume_id,
    // value is destination volume_id.
    // This field is OPTIONAL.
    map<string, string> volume_ids = 2;
  }

  oneof type {
    // Volume destination type
    VolumeDestination volume = 1;
    // Volume group destination type
    VolumeGroupDestination volumegroup = 2;
  }
}
```

### 2. Client Interface

Add to the `VolumeReplication` interface:

```go
// GetReplicationDestinationInfo RPC call to get destination info.
GetReplicationDestinationInfo(id, secretName, secretNamespace string) (*proto.GetReplicationDestinationInfoResponse, error)
```

### 3. Client Implementations

Implement `GetReplicationDestinationInfo` using `ReplicationSource_Volume`.

Implement `GetReplicationDestinationInfo` using
`ReplicationSource_VolumeGroup`.

### 4. Replication Wrapper

Add `GetDestinationInfo()` method following the same pattern as `GetInfo()`:

```go
func (r *Replication) GetDestinationInfo() *Response {
    id, err := r.getID()
    if err != nil {
        return &Response{Error: err}
    }
    resp, err := r.Params.Replication.GetReplicationDestinationInfo(
        id,
        r.Params.SecretName,
        r.Params.SecretNamespace,
    )
    return &Response{Response: resp, Error: err}
}
```

### 5. API Types

Add a new condition type and associated constants:

```go
// Condition type
const (
    // ... existing conditions ...
    ConditionDestinationInfoAvailable = "DestinationInfoAvailable"
)

// Messages
const (
    // ... existing messages ...
    MessageDestinationInfoAvailable = "destination info is available"
    MessageDestinationInfoPending   = "destination info is pending update"
    MessageDestinationInfoFailed    = "failed to get destination info"
)

// Reasons
const (
    // ... existing reasons ...
    DestinationInfoUpdated = "DestinationInfoUpdated"
    DestinationInfoPending = "DestinationInfoPending"
    FailedToGetDestinationInfo = "FailedToGetDestinationInfo"
)
```

Add destination fields to `VolumeReplicationStatus`:

```go
type VolumeReplicationStatus struct {
    // ... existing fields ...

    // DestinationVolumeID is the volume ID on the destination/target side.
    // This field is set when the SP reports different source and destination
    // volume IDs.
    // +optional
    DestinationVolumeID string `json:"destinationVolumeID,omitempty"`
}
```

Add destination fields to `VolumeGroupReplicationStatus`:

```go
type VolumeGroupReplicationStatus struct {
    VolumeReplicationStatus `json:",inline"`
    // ... existing fields ...

    // DestinationVolumeGroupID is the volume group ID on the
    // destination/target side.
    // +optional
    DestinationVolumeGroupID string `json:"destinationVolumeGroupID,omitempty"`

    // DestinationVolumeIDs is a mapping of source volume IDs to their
    // corresponding destination volume IDs. Key is source volume_id,
    // value is destination volume_id. This map reflects the current
    // group membership, accounting for dynamic group changes.
    // +optional
    DestinationVolumeIDs map[string]string `json:"destinationVolumeIDs,omitempty"`
}
```

### 6. Status Condition Helpers

```go
// setDestinationInfoAvailableCondition sets DestinationInfoAvailable=True
func setDestinationInfoAvailableCondition(conditions *[]metav1.Condition,
    observedGeneration int64, dataSource string) {
    source := getSource(dataSource)
    setStatusCondition(conditions, &metav1.Condition{
        Message:            fmt.Sprintf("%s %s", source, v1alpha1.MessageDestinationInfoAvailable),
        Type:               v1alpha1.ConditionDestinationInfoAvailable,
        Reason:             v1alpha1.DestinationInfoUpdated,
        ObservedGeneration: observedGeneration,
        Status:             metav1.ConditionTrue,
    })
}

// setDestinationInfoPendingCondition sets DestinationInfoAvailable=False
// (destination details are stale or not yet fetched)
func setDestinationInfoPendingCondition(conditions *[]metav1.Condition,
    observedGeneration int64, dataSource string) {
    source := getSource(dataSource)
    setStatusCondition(conditions, &metav1.Condition{
        Message:            fmt.Sprintf("%s %s", source, v1alpha1.MessageDestinationInfoPending),
        Type:               v1alpha1.ConditionDestinationInfoAvailable,
        Reason:             v1alpha1.DestinationInfoPending,
        ObservedGeneration: observedGeneration,
        Status:             metav1.ConditionFalse,
    })
}

// setDestinationInfoFailedCondition sets DestinationInfoAvailable=False
// with error details
func setDestinationInfoFailedCondition(conditions *[]metav1.Condition,
    observedGeneration int64, dataSource, errorMessage string) {
    source := getSource(dataSource)
    setStatusCondition(conditions, &metav1.Condition{
        Message:            fmt.Sprintf("%s %s: %s", source, v1alpha1.MessageDestinationInfoFailed, errorMessage),
        Type:               v1alpha1.ConditionDestinationInfoAvailable,
        Reason:             v1alpha1.FailedToGetDestinationInfo,
        ObservedGeneration: observedGeneration,
        Status:             metav1.ConditionFalse,
    })
}
```

### 7. Capability Check

Add a method to check if the driver supports `GET_REPLICATION_DESTINATION_INFO`:

```go
func (r *VolumeReplicationReconciler) supportsGetReplicationDestinationInfo(
    ctx context.Context, driverName string) bool {
    conn, err := r.Connpool.GetLeaderByDriver(ctx, r.Client, driverName)
    if err != nil {
        return false
    }
    for _, cap := range conn.Capabilities {
        if cap.GetVolumeReplication() == nil {
            continue
        }
        if cap.GetVolumeReplication().GetType() ==
            identity.Capability_VolumeReplication_GET_REPLICATION_DESTINATION_INFO {
            return true
        }
    }
    return false
}
```

### 8. Controller Flow - VolumeReplication

After a successful replication state change (promote/demote/resync/enable),
if the driver supports the capability, call `GetReplicationDestinationInfo`:

```go
// After successful replication operations, fetch destination info
if r.supportsGetReplicationDestinationInfo(ctx, driverName) {
    destInfo, err := r.getReplicationDestinationInfo(vr)
    if err != nil {
        setDestinationInfoFailedCondition(&instance.Status.Conditions,
            instance.Generation, instance.Spec.DataSource.Kind, err.Error())
    } else if destInfo != nil {
        if vol := destInfo.GetReplicationDestination().GetVolume(); vol != nil {
            instance.Status.DestinationVolumeID = vol.GetVolumeId()
        }
        setDestinationInfoAvailableCondition(&instance.Status.Conditions,
            instance.Generation, instance.Spec.DataSource.Kind)
    }
}
```

### 9. Controller Flow - VolumeGroupReplication (Dynamic Grouping)

This is the critical path for dynamic groups. The flow is:

```
Group membership changes (PVC labels change)
    |
    v
VGR reconcile detects PVC list change
    |
    v
VGRContent.Spec.Source.VolumeHandles is updated
    |
    v
Set DestinationInfoAvailable=False (destination details stale)
    |
    v
VGRContent controller calls ModifyVolumeGroupMembership
    |
    v
Call GetReplicationDestinationInfo
    |
    v
Update VGR status with new destination group ID + volume mappings
    |
    v
Set DestinationInfoAvailable=True
```

In the VGR reconciler, after updating `PersistentVolumeClaimsRefList` when
it differs from the previous list (indicating membership change):

```go
// When group membership changes, mark destination info as pending
if !reflect.DeepEqual(instance.Status.PersistentVolumeClaimsRefList, pvcRefList) {
    instance.Status.PersistentVolumeClaimsRefList = pvcRefList
    // Mark destination info as stale since group membership changed
    setDestinationInfoPendingCondition(&instance.Status.Conditions,
        instance.Generation, volumeGroupReplicationDataSource)
    // ...
}
```

Then, after the VR status is propagated back to VGR and the replication
operations complete, the VR controller (which handles the VGR's VR) will
call `GetReplicationDestinationInfo` for the group. The response
`VolumeGroupDestination` provides:

- Updated `volume_group_id` for the destination
- Updated `volume_ids` map (source -> destination) reflecting new membership

The VGR controller propagates these from VR status into VGR status and sets
`DestinationInfoAvailable=True`.

## Condition State Transitions

```
Initial state (no replication):

DestinationInfoAvailable: not present

After EnableVolumeReplication (if capability supported):

DestinationInfoAvailable: False (Reason: DestinationInfoPending)
        |
        v
GetReplicationDestinationInfo succeeds
DestinationInfoAvailable: True (Reason: DestinationInfoUpdated)

Dynamic group membership change:
    
DestinationInfoAvailable: True
        |
        v  
PVC list changes detected
DestinationInfoAvailable: False (Reason: DestinationInfoPending)
        |
        v 
GetReplicationDestinationInfo succeeds with updated mappings
DestinationInfoAvailable: True (Reason: DestinationInfoUpdated)

GetReplicationDestinationInfo fails:
DestinationInfoAvailable: False (Reason: FailedToGetDestinationInfo)
```

## Workflow: How Third-Party Tools Should Use Destination IDs

Third-party tools (DR orchestrators like Ramen, or any custom
disaster recovery controller) that consume VolumeReplication and
VolumeGroupReplication CRDs need to coordinate across **primary** and
**secondary** clusters. The destination IDs enable them to establish the
correct volume identity on the remote cluster during failover and failback.

### Terminology

| Term | Meaning |
|------|---------|
| Primary Cluster | The cluster where volumes are actively serving workloads |
| Secondary Cluster | The cluster receiving replicated data |
| DR Orchestrator | Third-party tool that manages failover/failback across clusters |
| Source Volume ID | The CSI volume handle on the primary cluster |
| Destination Volume ID | The CSI volume handle on the secondary cluster (may differ from source) |

### Workflow 1: Individual Volume Replication

#### Step 1: Setup Replication on Primary Cluster

The DR orchestrator creates the `VolumeReplication` CR on the primary
cluster referencing a PVC.

```yaml
apiVersion: replication.storage.openshift.io/v1alpha1
kind: VolumeReplication
metadata:
  name: vr-pvc-data
spec:
  volumeReplicationClass: rbd-replication
  dataSource:
    kind: PersistentVolumeClaim
    name: pvc-data
  replicationState: primary
```

#### Step 2: Read Destination Info from VR Status on Primary Cluster

Once replication is established and `DestinationInfoAvailable` is `True`,
the DR orchestrator reads the destination volume ID from VR status:

```yaml
status:
  state: Primary
  destinationVolumeID: "dest-vol-0001-abcdef"
  conditions:
  - type: DestinationInfoAvailable
    status: "True"
    reason: DestinationInfoUpdated
```

The DR orchestrator MUST:
1. Watch the `DestinationInfoAvailable` condition.
2. Only use destination IDs when the condition is `True`.
3. Store the mapping `source-vol-id -> destination-vol-id` in its own state
   (e.g., a DRPlacementControl or similar CR) so it is available during
   failover.

#### Step 3: Failover to Secondary Cluster

When a disaster is detected and failover is triggered, the DR orchestrator
performs the following steps **on the secondary cluster**:

1. **Create PV with the destination volume ID** - Use the destination volume
   ID (not the source volume ID) as the CSI volume handle:

   ```yaml
   apiVersion: v1
   kind: PersistentVolume
   metadata:
     name: pv-data-failover
   spec:
     csi:
       driver: example.csi.com
       volumeHandle: "dest-vol-0001-abcdef"  # destination ID, NOT source ID
       # ... other CSI attributes
     claimRef:
       name: pvc-data
       namespace: app-namespace
   ```

2. **Create PVC** bound to the PV above.

3. **Create VolumeReplication CR** on the secondary cluster and promote:

   ```yaml
   apiVersion: replication.storage.openshift.io/v1alpha1
   kind: VolumeReplication
   metadata:
     name: vr-pvc-data
   spec:
     volumeReplicationClass: rbd-replication
     dataSource:
       kind: PersistentVolumeClaim
       name: pvc-data
     replicationState: primary
   ```

4. **Demote on the old primary** (if still accessible):

   Update the VR on the old primary cluster to `replicationState: secondary`.

#### Step 4: Failback to Original Primary

When failing back, the DR orchestrator reverses the process:

1. Read `destinationVolumeID` from the VR status on the **current primary**
   (which was the secondary). This gives the volume ID on the original
   primary.
2. Demote on the current primary.
3. Create PV/PVC on the original primary using the correct volume ID.
4. Promote on the original primary.

### Workflow 2: Volume Group Replication (Dynamic Grouping)

Dynamic grouping adds complexity because the set of volumes in the group
changes over time as PVCs matching the label selector are added or removed.

#### Step 1: Setup Group Replication on Primary Cluster

```yaml
apiVersion: replication.storage.openshift.io/v1alpha1
kind: VolumeGroupReplication
metadata:
  name: vgr-app-data
spec:
  volumeGroupReplicationClassName: rbd-group-replication
  volumeReplicationClassName: rbd-replication
  source:
    selector:
      matchLabels:
        app: myapp
  replicationState: primary
```

#### Step 2: Read Destination Info from VGR Status on Primary Cluster

Once replication is established, the VGR status contains the full mapping:

```yaml
status:
  state: Primary
  destinationVolumeGroupID: "dest-group-0001"
  destinationVolumeIDs:
    "src-vol-001": "dest-vol-001"
    "src-vol-002": "dest-vol-002"
    "src-vol-003": "dest-vol-003"
  persistentVolumeClaimsRefList:
  - name: pvc-data-1
  - name: pvc-data-2
  - name: pvc-data-3
  conditions:
  - type: DestinationInfoAvailable
    status: "True"
    reason: DestinationInfoUpdated
```

The DR orchestrator MUST:

1. Watch the `DestinationInfoAvailable` condition on the VGR.
2. Only use destination IDs when the condition is `True`.
3. Store the complete `destinationVolumeIDs` map and
   `destinationVolumeGroupID` in its own state.
4. **Correlate source volume IDs to PVCs** by looking up each PVC's bound
   PV and matching its `spec.csi.volumeHandle` against the source volume ID
   keys in the map.

#### Step 3: Handle Dynamic Group Membership Changes on Primary Cluster

When a new PVC with matching labels is created (or an existing PVC's labels
change), the group membership updates dynamically:

1. The VGR controller detects the PVC list change.
2. `DestinationInfoAvailable` transitions to `False`
   (Reason: `DestinationInfoPending`).
3. The DR orchestrator observes this and MUST:
   - **Stop relying on the current destination mapping** as it is stale.
   - **Wait** for `DestinationInfoAvailable` to become `True` again.
4. After `ModifyVolumeGroupMembership` completes and
   `GetReplicationDestinationInfo` returns updated mappings,
   `DestinationInfoAvailable` transitions back to `True`.
5. The DR orchestrator reads the updated `destinationVolumeIDs` map which
   now includes entries for the newly added volumes (and removes entries for
   removed volumes).
6. The DR orchestrator updates its stored mapping accordingly.

```
  PVC-4 created with label app=myapp
      |
      v
  VGR reconcile: PVC list changed
  DestinationInfoAvailable: False (DestinationInfoPending)
      |  DR orchestrator sees condition=False, stops using stale mapping
      v
  VGRContent updated with new volume handle
  ModifyVolumeGroupMembership called
  GetReplicationDestinationInfo called
      |
      v
  DestinationInfoAvailable: True (DestinationInfoUpdated)
  destinationVolumeIDs now includes: "src-vol-004": "dest-vol-004"
      |
      v
  DR orchestrator reads updated mapping, stores it
```

#### Step 4: Failover to Secondary Cluster

The DR orchestrator performs failover using the **complete** destination
mapping. This requires recreating the full VolumeGroup resource hierarchy
on the secondary cluster with the correct destination identifiers.

##### Step 4a: Create PVs with Destination Volume IDs

For each volume in the group, create a PV on the secondary cluster using
the destination volume ID from the `destinationVolumeIDs` map:

```yaml
# For each entry in destinationVolumeIDs:
#   "src-vol-001": "dest-vol-001"
#   "src-vol-002": "dest-vol-002"
#   "src-vol-003": "dest-vol-003"

apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv-data-1-failover
spec:
  csi:
    driver: example.csi.com
    volumeHandle: "dest-vol-001"  # destination ID from the map
    # ... other CSI attributes (fsType, volumeAttributes, etc.)
  claimRef:
    name: pvc-data-1
    namespace: app-namespace
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv-data-2-failover
spec:
  csi:
    driver: example.csi.com
    volumeHandle: "dest-vol-002"
    # ...
  claimRef:
    name: pvc-data-2
    namespace: app-namespace
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv-data-3-failover
spec:
  csi:
    driver: example.csi.com
    volumeHandle: "dest-vol-003"
    # ...
  claimRef:
    name: pvc-data-3
    namespace: app-namespace
```

##### Step 4b: Create PVCs

Create PVCs bound to the PVs above, preserving the original PVC names and
labels so the application can start without configuration changes and the
label selector in the VGR can match them:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pvc-data-1
  namespace: app-namespace
  labels:
    app: myapp  # same labels as on primary, required for VGR selector
spec:
  volumeName: pv-data-1-failover
  # ... storageClassName, accessModes, resources
```

##### Step 4c: Create VolumeGroupReplicationContent with Destination Group Handle

The DR orchestrator MUST create the `VolumeGroupReplicationContent` on the
secondary cluster **before** creating the VGR, using the destination
volume group ID as the `volumeGroupReplicationHandle` and the destination
volume IDs as the `volumeHandles`:

```yaml
apiVersion: replication.storage.openshift.io/v1alpha1
kind: VolumeGroupReplicationContent
metadata:
  name: vgrcontent-failover
spec:
  provisioner: example.csi.com
  volumeGroupReplicationClassName: rbd-group-replication
  # Use the DESTINATION group handle, not the source group handle
  volumeGroupReplicationHandle: "dest-group-0001"
  source:
    # List the DESTINATION volume handles, not the source handles
    volumeHandles:
    - "dest-vol-001"
    - "dest-vol-002"
    - "dest-vol-003"
  # volumeGroupReplicationRef will be set once VGR is created
```

This is critical because:

- The CSI driver on the secondary cluster knows the volumes by their
  **destination** IDs. Using source IDs would cause the driver to fail
  to find the volumes.
- The `volumeGroupReplicationHandle` tells the driver which existing
  volume group to manage, without creating a new one.

##### Step 4d: Create VolumeGroupReplication and Promote

Create the VGR on the secondary cluster, referencing the pre-created
VGRContent:

```yaml
apiVersion: replication.storage.openshift.io/v1alpha1
kind: VolumeGroupReplication
metadata:
  name: vgr-app-data
  namespace: app-namespace
spec:
  volumeGroupReplicationClassName: rbd-group-replication
  volumeReplicationClassName: rbd-replication
  # Reference the pre-created VGRContent
  volumeGroupReplicationContentName: vgrcontent-failover
  source:
    selector:
      matchLabels:
        app: myapp
  replicationState: primary  # promote on secondary
```

The VGR controller on the secondary cluster will:

1. Find the existing VGRContent (already created with destination handles).
2. Skip creating a new volume group since
   `volumeGroupReplicationHandle` is already set.
3. Create a VR CR referencing the VGR.
4. The VR controller promotes the group to primary.

##### Step 4e: Demote on the Old Primary

If the old primary cluster is still accessible, update the VGR to
`replicationState: secondary`:

```yaml
# On old primary cluster
spec:
  replicationState: secondary
```

##### Failover Summary Diagram

```
Primary Cluster (old)                Secondary Cluster (new primary)
=====================                ==============================

VGR (vgr-app-data)                   PV (dest-vol-001) <-- Step 4a
  state: primary                     PV (dest-vol-002)
  destinationVolumeGroupID:          PV (dest-vol-003)
    "dest-group-0001"                    |
  destinationVolumeIDs:              PVC (pvc-data-1)  <-- Step 4b
    src-vol-001: dest-vol-001        PVC (pvc-data-2)
    src-vol-002: dest-vol-002        PVC (pvc-data-3)
    src-vol-003: dest-vol-003            |
        |                            VGRContent         <-- Step 4c
        | DR reads                     handle: "dest-group-0001"
        | destination                  volumeHandles:
        | info                           - dest-vol-001
        |                                - dest-vol-002
        +------>                         - dest-vol-003
                                         |
                                     VGR (vgr-app-data) <-- Step 4d
                                       state: primary
                                       contentName: vgrcontent-failover
                                         |
                                     VR (auto-created by VGR controller)
                                       state: primary
```

#### Step 5: Failback to Original Primary

When failing back, the DR orchestrator reverses the process. The key
difference from initial setup is that the DR orchestrator now reads
destination info from the **current primary** (the secondary cluster after
failover) to discover the volume IDs on the original primary.

##### Step 5a: Read Destination Info from Current Primary

On the secondary cluster (now acting as primary), read the VGR status:

```yaml
# On secondary cluster (current primary)
status:
  state: Primary
  destinationVolumeGroupID: "src-group-0001"  # points back to original primary
  destinationVolumeIDs:
    "dest-vol-001": "src-vol-001"  # dest->source mapping (reversed)
    "dest-vol-002": "src-vol-002"
    "dest-vol-003": "src-vol-003"
```

Note: The destination IDs from the secondary cluster's perspective point
to the original primary's volume IDs.

##### Step 5b: Demote on Current Primary

Update VGR on the secondary cluster to `replicationState: secondary`.

##### Step 5c: Recreate Resources on Original Primary

On the original primary cluster:

1. **Create PVs** using the volume IDs from the destination mapping
   (which are the original source IDs):

   ```yaml
   apiVersion: v1
   kind: PersistentVolume
   metadata:
     name: pv-data-1
   spec:
     csi:
       driver: example.csi.com
       volumeHandle: "src-vol-001"  # original source ID
   ```

2. **Create PVCs** bound to those PVs with matching labels.

3. **Create VolumeGroupReplicationContent** with the original group handle
   and source volume handles:

   ```yaml
   apiVersion: replication.storage.openshift.io/v1alpha1
   kind: VolumeGroupReplicationContent
   metadata:
     name: vgrcontent-failback
   spec:
     provisioner: example.csi.com
     volumeGroupReplicationClassName: rbd-group-replication
     volumeGroupReplicationHandle: "src-group-0001"
     source:
       volumeHandles:
       - "src-vol-001"
       - "src-vol-002"
       - "src-vol-003"
   ```

4. **Create VolumeGroupReplication** referencing the VGRContent and promote:

   ```yaml
   apiVersion: replication.storage.openshift.io/v1alpha1
   kind: VolumeGroupReplication
   metadata:
     name: vgr-app-data
   spec:
     volumeGroupReplicationClassName: rbd-group-replication
     volumeReplicationClassName: rbd-replication
     volumeGroupReplicationContentName: vgrcontent-failback
     source:
       selector:
         matchLabels:
           app: myapp
     replicationState: primary
   ```

### Summary: DR Orchestrator Responsibilities

| Responsibility | Primary Cluster | Secondary Cluster |
|---------------|----------------|-------------------|
| Watch condition | `DestinationInfoAvailable` on VR/VGR | N/A (until failover) |
| Store mapping | Read `destinationVolumeID` / `destinationVolumeIDs` from status | N/A |
| Handle staleness | Stop using mapping when condition is `False`, wait for `True` | N/A |
| Failover: create PVs | N/A | Use destination volume IDs as `volumeHandle` |
| Failover: create PVCs | N/A | Bind to PVs, preserve original PVC names |
| Failover: create VR/VGR | Demote to secondary | Create and promote to primary |
| Failback | Reverse: read destination IDs from current primary, create PVs | Demote to secondary |

### Workflow 3: When GET_REPLICATION_DESTINATION_INFO Is Not Supported

The `GET_REPLICATION_DESTINATION_INFO` capability is optional. Storage
providers where source and destination volume IDs are identical do not need
to implement this RPC. The DR orchestrator MUST handle this case gracefully.

#### How to Detect the Legacy Case

The DR orchestrator can determine that destination info is not available
through these indicators:

1. **`DestinationInfoAvailable` condition is absent** from the VR/VGR
   status conditions list entirely. This means the controller never
   attempted to fetch destination info because the driver does not advertise
   the `GET_REPLICATION_DESTINATION_INFO` capability.

2. **`destinationVolumeID` field is empty** on VR status (or
   `destinationVolumeGroupID` / `destinationVolumeIDs` are empty on VGR
   status).

The DR orchestrator SHOULD use the following check:

```go
func getDestinationVolumeID(vrStatus VolumeReplicationStatus, sourceVolumeID string) string {
    if vrStatus.DestinationVolumeID != "" {
        return vrStatus.DestinationVolumeID
    }
    // Capability not supported: source ID equals destination ID
    return sourceVolumeID
}
```

#### Individual Volume Replication Without Destination Info

##### Step 1: Setup (Same as Workflow 1)

Create VR CR on the primary cluster. No difference.

##### Step 2: Read VR Status on Primary Cluster

The VR status will have replication state and sync metrics but no
destination fields:

```yaml
status:
  state: Primary
  # destinationVolumeID is absent
  # DestinationInfoAvailable condition is absent
  conditions:
  - type: Completed
    status: "True"
    reason: Promoted
  - type: Replicating
    status: "True"
    reason: Replicating
```

The DR orchestrator notes that `destinationVolumeID` is empty and
`DestinationInfoAvailable` condition is absent. It concludes that source
and destination IDs are identical and records the source volume ID
(from `PV.spec.csi.volumeHandle`) as the volume ID to use on both
clusters.

##### Step 3: Failover to Secondary Cluster

The DR orchestrator uses the **source volume ID** directly as the volume
handle when creating the PV on the secondary cluster:

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv-data-failover
spec:
  csi:
    driver: example.csi.com
    volumeHandle: "src-vol-0001"  # same as source, since no destination info
    # ... other CSI attributes
  claimRef:
    name: pvc-data
    namespace: app-namespace
```

The remaining steps (create PVC, create VR, promote, demote old primary)
are identical to Workflow 1.

##### Step 4: Failback

Same as Workflow 1 Step 4, using the source volume ID since it is the same
on both clusters.

#### Volume Group Replication Without Destination Info

##### Step 1-2: Setup and Read Status (Same as Workflow 2)

Create VGR CR. The VGR status will have `persistentVolumeClaimsRefList`
but no destination fields:

```yaml
status:
  state: Primary
  # destinationVolumeGroupID is absent
  # destinationVolumeIDs is absent
  # DestinationInfoAvailable condition is absent
  persistentVolumeClaimsRefList:
  - name: pvc-data-1
  - name: pvc-data-2
  - name: pvc-data-3
```

The DR orchestrator concludes that source IDs equal destination IDs. It
builds the volume mapping by reading each PVC's bound PV and using
`PV.spec.csi.volumeHandle` as both source and destination ID.

##### Step 3: Handle Dynamic Group Membership Changes

Without destination info, the DR orchestrator cannot observe the
`DestinationInfoAvailable` condition (it is absent). Instead:

1. Watch `persistentVolumeClaimsRefList` on VGR status for membership
   changes.
2. When PVCs are added or removed, update the stored mapping by reading
   the new PVCs' bound PVs.
3. Since source ID equals destination ID, no additional RPC response is
   needed. The PV volume handle is the correct ID for both clusters.

##### Step 4: Failover

Since source and destination IDs are identical, the DR orchestrator uses
the source volume handles (from PVs on the primary cluster) directly.

###### Step 4a: Create PVs on Secondary Cluster

For each PVC in `persistentVolumeClaimsRefList`, read the bound PV's
`spec.csi.volumeHandle` and create PVs on the secondary cluster using the
same handle:

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv-data-1
spec:
  capacity:
    storage: 10Gi
  accessModes:
  - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  csi:
    driver: example.csi.com
    volumeHandle: "src-vol-001"  # same as source (no destination mapping)
    # ... other CSI attributes (fsType, volumeAttributes, nodeStageSecretRef)
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv-data-2
spec:
  capacity:
    storage: 10Gi
  accessModes:
  - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  csi:
    driver: example.csi.com
    volumeHandle: "src-vol-002"
---
apiVersion: v1
kind: PersistentVolume
metadata:
  name: pv-data-3
spec:
  capacity:
    storage: 10Gi
  accessModes:
  - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  csi:
    driver: example.csi.com
    volumeHandle: "src-vol-003"
```

###### Step 4b: Create PVCs on Secondary Cluster

Create PVCs that bind to the PVs above. The PVCs must have matching labels
so the VGR's label selector can discover them:

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: pvc-data-1
  namespace: app-namespace
  labels:
    app: myapp  # must match VGR source.selector.matchLabels
spec:
  accessModes:
  - ReadWriteOnce
  resources:
    requests:
      storage: 10Gi
  volumeName: pv-data-1  # bind to the pre-created PV
---
# Repeat for pvc-data-2 and pvc-data-3
```

###### Step 4c: Create VolumeGroupReplicationContent on Secondary Cluster

Since the group handle is the same on both clusters (source = destination),
use the original group handle from the primary cluster's VGRContent:

```yaml
apiVersion: replication.storage.openshift.io/v1alpha1
kind: VolumeGroupReplicationContent
metadata:
  name: vgrcontent-failover
spec:
  provisioner: example.csi.com
  volumeGroupReplicationClassName: rbd-group-replication
  volumeGroupReplicationHandle: "src-group-0001"  # same as source
  source:
    volumeHandles:
    - "src-vol-001"  # same as source (no destination mapping)
    - "src-vol-002"
    - "src-vol-003"
```

###### Step 4d: Create VolumeGroupReplication on Secondary Cluster

Create the VGR referencing the pre-created VGRContent and promote:

```yaml
apiVersion: replication.storage.openshift.io/v1alpha1
kind: VolumeGroupReplication
metadata:
  name: vgr-app-data
  namespace: app-namespace
spec:
  volumeGroupReplicationClassName: rbd-group-replication
  volumeReplicationClassName: rbd-replication
  volumeGroupReplicationContentName: vgrcontent-failover
  source:
    selector:
      matchLabels:
        app: myapp
  replicationState: primary
```

The VGR controller will:

1. Bind VGR to the pre-existing VGRContent.
2. Create individual VR CRs for each PVC.
3. Call `PromoteVolume` for each volume in the group.

###### Step 4e: Demote on Old Primary

Update VGR on the old primary cluster to `replicationState: secondary`.

##### Step 5: Failback

Since source and destination IDs are identical, failback is
straightforward:

1. **Demote** VGR on the current primary (secondary cluster) to secondary.
2. **Recreate resources** on the original primary using the same source
   volume IDs (since they are the same on both clusters). Follow the same
   pattern as Step 4a-4d with the original source IDs.
3. **Promote** VGR on the original primary.

#### Decision Flow for DR Orchestrator

```
Start: VR/VGR status available
    |
    v
Is DestinationInfoAvailable condition present?
    |                           |
   Yes                         No
    |                           |
    v                           v
Is condition status "True"?    Capability not supported.
    |           |              Use source volume ID as
   Yes         No              destination volume ID.
    |           |              (Legacy behavior)
    v           v
Use             Wait for
destination     condition to
IDs from        become "True"
status          before
fields.         proceeding.
```

### Important Considerations for DR Orchestrator Implementers

1. **Never assume source ID equals destination ID when destination info is
   available.** When `destinationVolumeID` / `destinationVolumeIDs` fields
   are populated, always use them. Only fall back to source ID when these
   fields are absent.

2. **Always check the `DestinationInfoAvailable` condition before using
   destination IDs.** If the condition is `False`, the mapping may be stale.
   If the condition is absent entirely, the capability is not supported and
   source IDs should be used.

3. **For volume groups, maintain the full mapping.** The
   `destinationVolumeIDs` map is the single source of truth for
   source-to-destination volume correlation. Individual volume IDs cannot be
   inferred from the group ID.

4. **Implement a unified volume ID resolution function.** The DR
   orchestrator should have a single code path that resolves the correct
   volume ID for a given cluster:

   ```go
   func resolveVolumeID(vrStatus, sourceVolumeID string) string {
       // If destination info is available, use it
       if vrStatus.DestinationVolumeID != "" {
           return vrStatus.DestinationVolumeID
       }
       // Otherwise, source and destination are identical
       return sourceVolumeID
   }
   ```

5. **During dynamic group changes when capability is supported, the DR
   orchestrator must not trigger failover while `DestinationInfoAvailable`
   is `False`.** The destination mapping is incomplete and failover would
   result in missing or incorrect PVs on the secondary cluster.

6. **During dynamic group changes when capability is NOT supported, the DR
   orchestrator can proceed with failover at any time.** Since source and
   destination IDs are identical, the PVC list in VGR status is sufficient
   to build the complete mapping. The orchestrator only needs to wait for
   `persistentVolumeClaimsRefList` to stabilize.