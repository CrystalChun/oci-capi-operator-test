# Components

These are all of the components this operator will deploy.

Each component encapsulates all of the subcomponents required
to deploy it.

## Autoscaler

Cluster autoscaler deployment uses the helm library.

Additional components outside of the helm chart include:
- ClusterRole CR
- ClusterRoleBinding CR

## CAPI

CAPI deployment uses the clusterctl library.

Additional components outside of the default clusterctl generation include:
- SecurityContextConstraints (SCC) CR for both CAPI and CAPOCI
- Namespace CR, default is `capi-system`
- ClusterRoleBinding CR
- Secret for the service account 

## CAPOCI

CAPOCI deployment uses the clusterctl library.

Additional components outside of the default clusterctl generation include:
- Namespace CR, default is `cluster-api-provider-oci-system`
- Secret CR (auth config secret to authenticate to OCI)

## CRDs

The CRDs are deployed in an init container for this operator. This
ensures the CRDs exist in the cluster ahead of the operator.

It includes all of the CAPI, CAPOCI CRDs.

## Enable Autoscaler

These are the CAPI CRs required to enable autoscaling. It includes:
- Cluster CR
- OCICluster CR
- MachineDeployment CR
- OCIMachineTemplate CR

