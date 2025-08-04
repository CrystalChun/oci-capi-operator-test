# OpenShift Cluster Autoscaling on Oracle Cloud Infrastructure with CAPI: A Complete Guide

OpenShift cluster autoscaling enables your Kubernetes clusters to dynamically adjust the number of worker nodes based on workload demands. This capability is essential for optimizing resource utilization and costs in cloud environments. In this comprehensive guide, we'll walk through implementing cluster autoscaling for OpenShift running on Oracle Cloud Infrastructure (OCI) using the Cluster API (CAPI) framework.

## Understanding the Architecture

The key components of this solution are:

- **Cluster API (CAPI)**: A Kubernetes project that provides declarative APIs and tooling to manage cluster lifecycle.
- **CAPOCI**: The OCI infrastructure provider for Cluster API.
- **cluster-autoscaler**: The component that monitors pod resource requests and scales nodes accordingly.

The architecture works by having the cluster-autoscaler monitor pending pods and communicate with CAPI to scale MachineDeployments, which in turn provision or terminate OCI compute instances.

## Prerequisites

Before starting, ensure you have:

1. [**Oracle Cloud CLI (oci-cli)**](https://docs.oracle.com/en-us/iaas/Content/API/SDKDocs/cliinstall.htm) is installed and configured with proper credentials.
2. [**OpenShift CLI (oc)**](https://mirror.openshift.com/pub/openshift-v4/clients/ocp/latest/openshift-client-linux.tar.gz) - The OpenShift command line tool.
3. [**clusterctl**](https://cluster-api.sigs.k8s.io/user/quick-start#install-clusterctl) - The Cluster API management tool.
4. [**Helm**](https://helm.sh/docs/intro/install/) - Package manager for Kubernetes, we will use it to provision cluster-autoscaler.
5. **An existing OpenShift cluster on OCI** with external platform integration enabled.
6. **Administrative access** to the OpenShift cluster.
7. **A custom RHCOS image** in your OCI tenancy [we'll cover this].
8. **Oracle Cloud Account** with API access.

## Manual Component Installation

For now, we must manually install and configure all components.
All this process can be streamlined by creating an operator that manages all the elements in this stack.

### Step 1: Create Custom RHCOS Image

You need a custom Red Hat CoreOS image in your OCI tenancy for the autoscaling nodes:

1. **Download RHCOS Image (OpenStack flavor) that matches your OpenShift version**:
    In this example, our version is 4.19.0
   ```bash
   curl -LO https://mirror.openshift.com/pub/openshift-v4/x86_64/dependencies/rhcos/4.19/4.19.0/rhcos-4.19.0-x86_64-openstack.x86_64.qcow2.gz
   gzip -d rhcos-4.19.0-x86_64-openstack.x86_64.qcow2
   ```

2. **Import to OCI**:
   - Follow Oracle's documentation: [Create a Custom Linux Image](https://docs.public.oneportal.content.oci.oraclecloud.com/en-us/iaas/compute-cloud-at-customer/topics/images/importing-custom-linux-imges.htm)
   - Upload the qcow2 file to OCI and create a custom image
   - Note the image OCID and/or name for later use

### Step 2: Install CAPI with OCI infrastructure provider (CAPOCI)

**Configure OCI values and credentials**:
```bash
export OCI_TENANCY_ID="$(grep -E "^tenancy=" ~/.oci/config | cut -d = -f 2)"
export OCI_USER_ID="$(grep -E "^user=" ~/.oci/config | cut -d = -f 2)"
export OCI_REGION="$(grep -E "^region=" ~/.oci/config | cut -d = -f 2)"
export OCI_CREDENTIALS_FINGERPRINT="$(grep -E "^fingerprint=" ~/.oci/config | cut -d = -f 2)"
export OCI_CREDENTIALS_KEY="$(grep -E "^key_file=" ~/.oci/config | cut -d = -f 2)"

export OCI_TENANCY_ID_B64="$(echo -n "$OCI_TENANCY_ID" | base64 -w0)"
export OCI_USER_ID_B64="$(echo -n "$OCI_USER_ID" | base64 -w0)"
export OCI_REGION_B64="$(echo -n "$OCI_REGION" | base64 -w0)"
export OCI_CREDENTIALS_FINGERPRINT_B64="$(echo -n "$OCI_CREDENTIALS_FINGERPRINT" | base64 -w0)"
export OCI_CREDENTIALS_KEY_B64="$(base64 -w0 < "$OCI_CREDENTIALS_KEY")"

# if Passphrase is present
export OCI_CREDENTIALS_PASSPHRASE="$(grep -E "^passphrase=" ~/.oci/config | cut -d = -f 2)"
export OCI_CREDENTIALS_PASSPHRASE_B64="$(echo -n "$OCI_CREDENTIALS_PASSPHRASE" | base64 -w0)"
```

**Create a custom SecurityContextConstraint to allow RunAsUser for the CAPI components**:
```bash
cat > oci-capi-scc.yaml << EOF
kind: SecurityContextConstraints
apiVersion: security.openshift.io/v1
metadata:
  name: oci-capi
runAsUser:
  type: RunAsAny
seLinuxContext:
  type: RunAsAny
seccompProfiles:
  - runtime/default
users:
  - system:serviceaccount:cluster-api-provider-oci-system:capoci-controller-manager
  - system:serviceaccount:capi-system:capi-manager
EOF

oc apply -f oci-capi-scc.yaml
```

**Install CAPI and CAPOCI using clusterctl**:

```bash
clusterctl init --bootstrap - --control-plane - --infrastructure oci
```

### Step 3: Install cluster-autoscaler

Install and configure cluster-autoscaler using Helm:

```bash
# Add autoscaler Helm repository
helm repo add autoscaler https://kubernetes.github.io/autoscaler
helm repo update

# Create values file for cluster-autoscaler
cat > cluster-autoscaler-values.yaml << EOF
cloudProvider: clusterapi
fullnameOverride: oci-cluster-autoscaler
autoDiscovery:
  namespace: capi-system
rbac:
  create: true
  serviceAccount:
    create: true
    name: oci-cluster-autoscaler
EOF

# Install cluster-autoscaler
helm install oci-cluster-autoscaler autoscaler/cluster-autoscaler \
  --values cluster-autoscaler-values.yaml \
  --namespace capi-system
```

### Step 4: Configure cluster-autoscaler RBAC

cluster-autoscaler needs additional permissions to work with CAPI resources:

```bash
cat > role-cluster-autoscaler.yaml << EOF
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: oci-cluster-autoscaler-extra
rules:
  - apiGroups:
    - infrastructure.cluster.x-k8s.io
    resources:
    - "*"
    verbs:
    - get
    - list
    - watch
    - update

---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: oci-cluster-autoscaler-extra
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: oci-cluster-autoscaler-extra
subjects:
  - kind: ServiceAccount
    name: oci-cluster-autoscaler
    namespace: capi-system
EOF

oc apply -f role-cluster-autoscaler.yaml
```

### Step 5: Create Bootstrap Ignition Configuration

Create ignition configuration to be used when provisioning new nodes.

```bash
# Get cluster information
cluster_name=$(oc get infrastructure cluster -ojsonpath='{.status.infrastructureName}')
echo "Cluster name is $cluster_name"

secret="${cluster_name}-bootstrap"

# Create temporary directory
bootstrap_dir=bootstrap-ignition-secret
mkdir -p $bootstrap_dir

# Get required cluster certificates and endpoints
MACHINECONFIG_CA=$(oc get secret -n openshift-machine-config-operator machine-config-server-tls -o jsonpath='{.data.tls\.crt}')
API_INT_HOST=$(oc get infrastructure cluster -o jsonpath='{.status.apiServerInternalURI}' | cut -d / -f 3- | cut -d : -f 1)

# Create the bootstrap ignition template
mkdir -p $bootstrap_dir/secret
echo ignition > $bootstrap_dir/secret/format

cat > $bootstrap_dir/bootstrap-ignition.json << EOF
{
  "ignition": {
    "config": {
      "merge": [
        {
          "source": "https://${API_INT_HOST}:22623/config/worker"
        }
      ]
    },
    "security": {
      "tls": {
        "certificateAuthorities": [
          {
            "source": "data:text/plain;charset=utf-8;base64,${MACHINECONFIG_CA}"
          }
        ]
      }
    },
    "version": "3.2.0"
  }
}
EOF

# Create hostname setting ignition for OCI
# Due to a bug, NetworkManager wont set the hostname if the FQDN is longer than 63 characters
# We need this workaround in case the resulting hostname is too long
cat > $bootstrap_dir/set-hostname-oci-ignition.json << EOF
{
  "systemd": {
    "units": [
      {
        "enabled": true,
        "name": "set-hostname-oci.service",
        "contents": "[Unit]\nDescription=Set hostname from OCI metadata\nAfter=network-online.target\nWants=network-online.target\n[Service]\nType=oneshot\nExecStart=/usr/local/bin/set-hostname-oci.sh\n[Install]\nWantedBy=multi-user.target\n"
      }
    ]
  },
  "storage": {
    "files": [
      {
        "path": "/usr/local/bin/set-hostname-oci.sh",
        "mode": 493,
        "contents": {
          "source": "data:text/plain;charset=utf-8;base64,IyEvYmluL2Jhc2gKc2V0IC1ldW8gcGlwZWZhaWwKCiMgR2V0IGhvc3RuYW1lIGZyb20gT0NJIG1ldGFkYXRhCmhvc3RuYW1lPSQoY3VybCAtcyBodHRwOi8vMTY5LjI1NC4xNjkuMjU0L29wYy92Mi9pbnN0YW5jZS8gfCBqcSAtciAuZGlzcGxheU5hbWUpCgppZiBbIC1uICIkaG9zdG5hbWUiIF07IHRoZW4KICAgIGVjaG8gIlNldHRpbmcgaG9zdG5hbWUgdG8gJGhvc3RuYW1lIgogICAgaG9zdG5hbWVjdGwgc2V0LWhvc3RuYW1lICIkaG9zdG5hbWUiCmVsc2UKICAgIGVjaG8gIkZhaWxlZCB0byBnZXQgaG9zdG5hbWUgZnJvbSBPQ0kgbWV0YWRhdGEiCiAgICBleGl0IDEKZmkK"
        }
      }
    ]
  }
}
EOF

# Create final ignition json, merging the workaround with the bootstrap ignition file
jq '.systemd += input.systemd' $bootstrap_dir/bootstrap-ignition.json $bootstrap_dir/set-hostname-oci-ignition.json \
    | jq '.storage += input.storage' - $bootstrap_dir/set-hostname-oci-ignition.json > $bootstrap_dir/secret/value

# Create the secret
oc create -n capi-system secret generic $secret --from-file=$bootstrap_dir/secret --dry-run=client -o yaml | oc apply -f -

echo "Created bootstrap ignition secret: $secret"
```

### Step 6: Create Kubeconfig for CAPI Cluster

CAPI needs a kubeconfig to manage the CAPI cluster (even if it's a self-managed cluster).
Let's create a kubeconfig that grants CAPI full admin permissions into the current cluster:

```bash
# Get cluster information
cluster_name=$(oc get infrastructure cluster -ojsonpath='{.status.infrastructureName}')
secret="${cluster_name}-kubeconfig"

# Create temporary directory
kubeconfig_dir=kubeconfig-secret
mkdir -p $kubeconfig_dir

# Create service account and get token
oc create serviceaccount $cluster_name -n capi-system
oc extract -n kube-system configmap/kube-root-ca.crt --to=- > $kubeconfig_dir/cluster-ca.crt
oc adm policy add-cluster-role-to-user cluster-admin -z $cluster_name -n capi-system
oc create token $cluster_name -n capi-system --duration=8760h > $kubeconfig_dir/token

# Create kubeconfig
cat > $kubeconfig_dir/kubeconfig << EOF
apiVersion: v1
kind: Config
clusters:
- name: $cluster_name
  cluster:
    certificate-authority-data: $(base64 -w0 < $kubeconfig_dir/cluster-ca.crt)
    server: https://kubernetes.default.svc
contexts:
- name: $cluster_name
  context:
    cluster: $cluster_name
    namespace: capi-system
    user: $cluster_name
current-context: $cluster_name
users:
- name: $cluster_name
  user:
    token: $(cat $kubeconfig_dir/token)
EOF

# Create the secret with proper labels
oc create secret generic -n capi-system $secret --from-file=value=$kubeconfig_dir/kubeconfig --type=cluster.x-k8s.io/secret
oc label secret -n capi-system $secret cluster.x-k8s.io/cluster-name=$cluster_name clusterctl.cluster.x-k8s.io/move=""

echo "Created kubeconfig secret: $secret"
```

### Step 7: Create CAPI Manifests

Now we'll create the CAPI resources that define our autoscaling infrastructure

First of all, we'll need to populate environment variables
```bash
# Autoscaling configuration
autoscaling_nodegroup_min=0
autoscaling_nodegroup_max=5
autoscaling_shape=VM.Standard.E4.Flex
autoscaling_shapeconfig_cpu=6
autoscaling_shapeconfig_memory=16

# Running cluster configuration
cluster_name=$(oc get infrastructure cluster -ojsonpath='{.status.infrastructureName}')
oci_cluster_name=$(echo "$cluster_name" | rev | cut -d - -f 2- | rev)
cluster_cidr=$(oc get network.config.openshift.io cluster -o jsonpath='{.spec.clusterNetwork[*].cidr}')
service_cidr=$(oc get network.config.openshift.io cluster -o jsonpath='{.spec.serviceNetwork[*]}')

```
Then, we will set all the needed values to import the running cluster.

Two different methods can be used:

#### Method 1. Auto-discover OCID values
This method auto-discovers the OCID of most of the objects, assuming the cluster was created using the (assisted installer and terraform method)[https://docs.redhat.com/en/documentation/openshift_container_platform/4.19/html/installing_on_oci/installing-oci-assisted-installer].

```bash
# Modify to match your environment
compartment_name="your-compartment-name"
image_name="your-custom-image-name"

# Those names are used by the Openshift OCI terraform stack
nsg_name=cluster-compute-nsg
subnet_name=private

compartment=$(oci iam compartment list --all --compartment-id-in-subtree true --access-level ACCESSIBLE --raw-output --query "data[?name=='$compartment_name'].id | [0]")
image=$(oci compute image list --compartment-id "$compartment" --display-name "$image_name" | jq -r '.data[0].id')
vcn=$(oci network vcn list --compartment-id "$compartment" --display-name "$oci_cluster_name" | jq -r '.data[0].id')
apiserver_lb=$(oci lb load-balancer list --compartment-id $compartment --display-name ${oci_cluster_name}-openshift_api_apps_lb | jq -r '.data[].id')
control_plane_endpoint=$(oci lb load-balancer list --compartment-id $compartment --display-name ${oci_cluster_name}-openshift_api_apps_lb \
                         | jq -r '.data[]."ip-addresses"[] | select(."is-public" == true) | ."ip-address"')
subnet=$(oci network subnet list --compartment-id "$compartment" --vcn-id "$vcn" --display-name "$subnet_name" | jq -r '.data[0].id')
nsg=$(oci network nsg list --compartment-id "$compartment" --vcn-id "$vcn" --display-name "$nsg_name" | jq -r '.data[0].id')

cat << EOF
Auto-discovered values:
cluster_name=$cluster_name
compartment=$compartment
image=$image
vcn=$vcn
control_plane_endpoint=$control_plane_endpoint
apiserver_lb=$apiserver_lb
subnet=$subnet
nsg=$nsg

EOF
```

#### Method 2. Hardcode OCID values
If your cluster is installed differently, or you're not sure, just find the OCID of each object and set the proper
variables
```bash
compartment=ocid1.compartment.oc1..your_compartment_id
image=ocid1.image.oc1.us-sanjose-1.your_image_id
vcn=ocid1.vcn.oc1.us-sanjose-1.your_vcn_id
apiserver_lb=ocid1.loadbalancer.oc1.us-sanjose-1.your_apiserver_lb_id
control_plane_endpoint=192.168.2.3 # Here goes your Control Plane Endpoint IP Address
nsg=ocid1.networksecuritygroup.oc1.us-sanjose-1.your_network_security_group_id_for_worker_nodes
subnet=ocid1.subnet.oc1.us-sanjose-1.your_subnet_id_for_worker_nodes
```

#### Create manifests
```bash
# Create manifests directory
mkdir -p manifests

# 1. OCICluster - defines the OCI infrastructure
cat > manifests/01-ocicluster.yaml << EOF
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta2
kind: OCICluster
metadata:
  name: ${cluster_name}
  namespace: capi-system
  labels:
    cluster.x-k8s.io/cluster-name: ${cluster_name}
  annotations:
    cluster.x-k8s.io/skip-apiserver-lb-management: "true"
spec:
  compartmentId: ${compartment}
  controlPlaneEndpoint:
    host: ${control_plane_endpoint}
    port: 6443
  networkSpec:
    apiServerLoadBalancer:
      loadBalancerId: ${apiserver_lb}
    skipNetworkManagement: true
    vcn:
      id: ${vcn}
      subnets:
      - id: ${subnet}
        name: private
        role: worker
      networkSecurityGroup:
        list:
        - id: ${nsg}
          name: cluster-compute-nsg
          role: worker
EOF

# 2. Cluster - links the infrastructure to CAPI
cat > manifests/02-cluster.yaml << EOF
---
apiVersion: cluster.x-k8s.io/v1beta1
kind: Cluster
metadata:
  name: ${cluster_name}
  namespace: capi-system
  labels:
    cluster.x-k8s.io/cluster-name: ${cluster_name}
spec:
  clusterNetwork:
    pods:
      cidrBlocks:
      - ${cluster_cidr}
    serviceDomain: cluster.local
    services:
      cidrBlocks:
      - ${service_cidr}
  infrastructureRef:
    apiVersion: infrastructure.cluster.x-k8s.io/v1beta2
    kind: OCICluster
    name: ${cluster_name}
    namespace: capi-system
EOF

# 3. OCIMachineTemplate - defines the machine template for autoscaling
cat > manifests/03-ocimachinetemplate.yaml << EOF
---
apiVersion: infrastructure.cluster.x-k8s.io/v1beta2
kind: OCIMachineTemplate
metadata:
  name: ${cluster_name}-autoscaling
  namespace: capi-system
spec:
  template:
    spec:
      imageId: ${image}
      shape: ${autoscaling_shape}
      shapeConfig:
        ocpus: "${autoscaling_shapeconfig_cpu}"
        memoryInGBs: "${autoscaling_shapeconfig_memory}"
      isPvEncryptionInTransitEnabled: false
EOF

# 4. MachineDeployment - defines the autoscaling worker nodes
cat > manifests/04-machinedeployment.yaml << EOF
---
apiVersion: cluster.x-k8s.io/v1beta1
kind: MachineDeployment
metadata:
  name: ${oci_cluster_name}
  namespace: capi-system
  annotations:
    capacity.cluster-autoscaler.kubernetes.io/cpu: "${autoscaling_shapeconfig_cpu}"
    capacity.cluster-autoscaler.kubernetes.io/memory: "${autoscaling_shapeconfig_memory}G"
    cluster.x-k8s.io/cluster-api-autoscaler-node-group-min-size: "${autoscaling_nodegroup_min}"
    cluster.x-k8s.io/cluster-api-autoscaler-node-group-max-size: "${autoscaling_nodegroup_max}"
spec:
  clusterName: ${cluster_name}
  selector:
    matchLabels:
  template:
    spec:
      clusterName: ${cluster_name}
      bootstrap:
        dataSecretName: ${cluster_name}-bootstrap
      infrastructureRef:
        name: ${cluster_name}-autoscaling
        namespace: capi-system
        apiVersion: infrastructure.cluster.x-k8s.io/v1beta2
        kind: OCIMachineTemplate
EOF

echo "CAPI manifests created in manifests/ directory"
```

### Step 8: Apply CAPI Resources

Apply all the CAPI resources to enable autoscaling:

```bash
oc apply -f manifests/
```

Monitor the cluster status:
```bash
# Watch cluster status
oc get cluster -n capi-system -w

# Check machine deployment
oc get machinedeployment -n capi-system

# Monitor cluster-autoscaler logs
oc logs -f deployment/oci-cluster-autoscaler -n capi-system
```

### Certificate Auto-Approval

New nodes cannot be added to the cluster until their certificates are approved.
Until a component is created (maybe as part of a future operator) that does this automatically, new nodes need to be approved manually.
Here's a simple shellscript that watches for pending CSRs, checks if the CSR matches any of the OCIMachines, and if so, approves
the certificate.
You can run this manually for testing purposes, or inside a container in the cluster that runs with enough privileges to:
- list OCIMachines in capi-system
- list, get and update CSRs
```bash
#!/bin/bash

set -euo pipefail

get_csr_hostname() {
    local csr=$1
    signer_name=$(oc get csr $csr -ojsonpath={.spec.signerName})
    case $signer_name in
        kubernetes.io/kube-apiserver-client-kubelet)
            # In this case we need to inspect the CSR to get the host name from the CN
            oc get csr $csr -ojsonpath={.spec.request} \
                | base64 -d \
                | openssl req -in - -subject -noout \
                | xargs -n1 \
                | grep CN=system:node \
                | cut -d : -f 3 \
                | cut -d . -f 1
            ;;
        kubernetes.io/kubelet-serving)
            # In this case the node name can be extracted from the username issuing the CSR
            oc get csr $csr -ojsonpath={.spec.username} \
                | cut -d : -f 3 \
                | cut -d . -f 1
            ;;
    esac
}

check_and_sign_csrs(){
    local csr
    for csr in $(oc get csr -ojson | jq '.items[] | select(.status=={}) | .metadata.name' -r); do
        # Get node name from CSR
        node_name=$(get_csr_hostname $csr)
        # If there is an OCIMachine present with the same name as the node name, sign the certificate
        if oc get ocimachine -oname -n capi-system | cut -d / -f 2 | grep -xq $node_name; then
            echo "Approving certificate $csr for node $node_name"
            oc adm certificate approve $csr
        fi
    done
}

while true; do
    check_and_sign_csrs
    sleep 10
done

```
### Scale Up Test

Create a deployment that will trigger node scaling:

```bash
# Create nginx deployment
oc create deployment nginx --namespace default --image=docker.io/nginx:latest --replicas=0

# Set resource requests to force node creation
oc set resources deployment -n default nginx --requests=memory=2Gi

# Scale up to trigger autoscaling
# Remember to adjust the number of replicas until not all of them will fit the cluster
# That will trigger a scale up
oc scale deployment -n default nginx --replicas=20
```

Monitor the scaling process:
```bash
# Watch machine deployments
oc get machinedeployment -n capi-system -w

# Watch cluster status
oc get cluster -n capi-system -w

# Watch nodes
oc get nodes -w

# Check cluster-autoscaler logs
oc logs -f deployment/oci-cluster-autoscaler -n capi-system
```

### Scale Down Test

Test scale-down functionality:

```bash
# Reduce replica count
oc scale deployment -n default nginx --replicas=5

# Watch for node removal (takes several minutes due to scale-down delay)
oc get nodes -w
```

## Monitoring and Troubleshooting

### Checking Component Health

Monitor the health of all components:

```bash
# Check CAPI
oc get pods -n capi-system

# Check CAPOCI
oc get pods -n cluster-api-provider-oci-system

# Check CAPI cluster status
oc get cluster -n capi-system
oc describe cluster -n capi-system

# Check CAPI MachineDeployment status
oc get machinedeployment -n capi-system
oc describe machinedeployment -n capi-system
```

### Logs and Debugging

```bash
# cluster-autoscaler logs
oc logs deployment/oci-cluster-autoscaler -n capi-system

# CAPI controller logs
oc logs deployment/capi-controller-manager -n capi-system

# CAPOCI controller logs
oc logs deployment/capoci-controller-manager -n cluster-api-provider-oci-system
```

## Current Limitations

1. **Security Context Constraints**: A new SCC needs to be created to deploy CAPI/CAPOCI in OpenShift
2. **Extra RBAC Permissions**: Extra RBAC permissions need to be created so cluster-autoscaler can work with CAPOCI objects
3. **Hostname Setting**: A custom set-hostname-oci script is added to account for cases when the cluster name is too long and it
   triggers a bug in NetworkManager
4. **Certificate Approval**: Manual or automated certificate approval is required for new nodes
5. **Pre-existing Nodes**: cluster-autoscaler complains about nodes not managed by CAPI

## Conclusion

Implementing cluster autoscaling for OpenShift on Oracle Cloud Infrastructure using CAPI provides a robust, cloud-native solution for dynamic resource management. While the setup involves multiple components and careful configuration, the result is a highly scalable infrastructure that can automatically adapt to workload demands.

Remember to monitor your autoscaling behavior closely in production environments and adjust the min/max node counts and scaling policies based on your specific workload patterns and requirements.