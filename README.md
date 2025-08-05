# OCI CAPI Operator

This operator automates the deployment and management of Cluster API (CAPI) components for Oracle Cloud Infrastructure (OCI) in OpenShift clusters, enabling seamless cluster autoscaling.

## Description

The OCI CAPI Operator streamlines the complex process documented in [operator.md](./operator.md) by automating:

- CAPI and CAPOCI installation management
- SecurityContextConstraints creation for OpenShift compatibility
- Cluster-autoscaler deployment and configuration
- RBAC setup for all components
- Certificate auto-approval for new nodes
- OCI credentials management

## Features

- **Automated CAPI Stack Management**: Monitors and ensures CAPI components are properly installed
- **Cluster Autoscaling**: Deploys and configures cluster-autoscaler for OCI
- **Certificate Management**: Automatically approves certificates for new OCI machines
- **OpenShift Integration**: Creates necessary SecurityContextConstraints
- **Comprehensive RBAC**: Sets up proper permissions for all components

## Getting Started

### Prerequisites

For using the operator:
- OpenShift cluster running on OCI
- Oracle Cloud CLI (oci-cli) configured with proper credentials
- Custom RHCOS image in your OCI tenancy
- CAPI/CAPOCI installed (can be done via `clusterctl init --infrastructure oci`)

For development:
- go version v1.22.0+
- docker version 17.03+
- kubectl version v1.11.3+
- Access to a Kubernetes v1.11.3+ cluster

### Configuration

Create OCI Private Key Secret:

```bash
kubectl create secret generic oci-private-key \
  --from-file=private_key=/path/to/your/oci-private-key.pem \
  -n <operator-namespace>
```

Create OCIClusterAutoscaler Resource:

```yaml
apiVersion: capi.openshift.io/v1beta1
kind: OCIClusterAutoscaler
metadata:
  name: my-cluster-autoscaler
spec:
  oci:
    tenancyId: "ocid1.tenancy.oc1..your-tenancy-id"
    userId: "ocid1.user.oc1..your-user-id"
    region: "us-sanjose-1"
    fingerprint: "your-key-fingerprint"
    privateKeySecretRef:
      name: "oci-private-key"
      key: "private_key"
    compartmentId: "ocid1.compartment.oc1..your-compartment-id"
    imageId: "ocid1.image.oc1.us-sanjose-1.your-custom-rhcos-image"
    network:
      vcnId: "ocid1.vcn.oc1.us-sanjose-1.your-vcn-id"
      subnetId: "ocid1.subnet.oc1.us-sanjose-1.your-subnet-id"
      networkSecurityGroupId: "ocid1.networksecuritygroup.oc1.us-sanjose-1.your-nsg-id"
      apiServerLoadBalancerId: "ocid1.loadbalancer.oc1.us-sanjose-1.your-lb-id"
      controlPlaneEndpoint: "your-control-plane-ip"
  autoscaling:
    minSize: 0
    maxSize: 10
    shape: "VM.Standard.E4.Flex"
    shapeConfig:
      cpus: "4"
      memoryInGBs: "16"
```

### To Deploy on the cluster
**Build and push your image to the location specified by `IMG`:**

```sh
make docker-build docker-push IMG=<some-registry>/oci-capi-operator:tag
```

**NOTE:** This image ought to be published in the personal registry you specified.
And it is required to have access to pull the image from the working environment.
Make sure you have the proper permission to the registry if the above commands don’t work.

**Install the CRDs into the cluster:**

```sh
make install
```

**Deploy the Manager to the cluster with the image specified by `IMG`:**

```sh
make deploy IMG=<some-registry>/oci-capi-operator:tag
```

> **NOTE**: If you encounter RBAC errors, you may need to grant yourself cluster-admin
privileges or be logged in as admin.

**Create instances of your solution**
You can apply the samples (examples) from the config/sample:

```sh
kubectl apply -k config/samples/
```

>**NOTE**: Ensure that the samples has default values to test it out.

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```

## Project Distribution

Following are the steps to build the installer and distribute this project to users.

1. Build the installer for the image built and published in the registry:

```sh
make build-installer IMG=<some-registry>/oci-capi-operator:tag
```

NOTE: The makefile target mentioned above generates an 'install.yaml'
file in the dist directory. This file contains all the resources built
with Kustomize, which are necessary to install this project without
its dependencies.

2. Using the installer

Users can just run kubectl apply -f <URL for YAML BUNDLE> to install the project, i.e.:

```sh
kubectl apply -f https://raw.githubusercontent.com/<org>/oci-capi-operator/<tag or branch>/dist/install.yaml
```

## Contributing
// TODO(user): Add detailed information on how you would like others to contribute to this project

**NOTE:** Run `make help` for more information on all potential `make` targets

More information can be found via the [Kubebuilder Documentation](https://book.kubebuilder.io/introduction.html)

## License

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

