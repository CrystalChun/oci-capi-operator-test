# OCI CAPI Operator

The OCI CAPI Operator automates the setup and management of OpenShift cluster autoscaling on Oracle Cloud Infrastructure (OCI) using the Cluster API (CAPI) framework.

## Features

- Automated installation and configuration of CAPI and CAPOCI controllers
- Automated setup of cluster-autoscaler
- Management of RHCOS worker node autoscaling
- Automatic certificate approval for new nodes
- Integrated OCI infrastructure management

## Prerequisites

Before using this operator, ensure you have:

1. An OpenShift cluster running on OCI
2. OCI credentials with appropriate permissions
3. A custom RHCOS image in your OCI tenancy
4. `oc` CLI tool installed and configured
5. Administrative access to the OpenShift cluster

## Installation

1. Install the operator CRDs:
   ```bash
   make install
   ```

2. Deploy the operator:
   ```bash
   make deploy IMG=quay.io/openshift/oci-capi-operator:latest
   ```

## Usage

1. Create a secret containing your OCI credentials:
   ```yaml
   apiVersion: v1
   kind: Secret
   metadata:
     name: oci-credentials
     namespace: capi-system
   type: Opaque
   data:
     private-key: <base64-encoded-private-key>
   ```

2. Create an OCICAPICluster resource:
   ```yaml
   apiVersion: ocicapi.openshift.io/v1alpha1
   kind: OCICAPICluster
   metadata:
     name: my-cluster
     namespace: capi-system
   spec:
     oci:
       compartmentId: "ocid1.compartment.oc1..example"
       region: "us-ashburn-1"
       credentials:
         tenancyId: "ocid1.tenancy.oc1..example"
         userId: "ocid1.user.oc1..example"
         privateKeySecretRef:
           name: oci-credentials
           namespace: capi-system
           key: private-key
         fingerprint: "aa:bb:cc:dd:ee:ff:00:11:22:33:44:55:66:77:88:99"
     autoscaling:
       minNodes: 0
       maxNodes: 5
       nodeShape: "VM.Standard.E4.Flex"
       shapeConfig:
         ocpus: 6
         memoryInGBs: 16
     network:
       vcnId: "ocid1.vcn.oc1.iad.example"
       subnetId: "ocid1.subnet.oc1.iad.example"
       networkSecurityGroupId: "ocid1.networksecuritygroup.oc1.iad.example"
       apiServerLoadBalancerId: "ocid1.loadbalancer.oc1.iad.example"
       controlPlaneEndpoint: "192.0.2.1"
     image:
       imageId: "ocid1.image.oc1.iad.example"
   ```

## Development

### Prerequisites

- Go 1.21+
- Docker
- make
- kubectl

### Building

1. Build the operator:
   ```bash
   make build
   ```

2. Build and push the Docker image:
   ```bash
   make docker-build docker-push IMG=<your-registry>/oci-capi-operator:tag
   ```

### Testing

Run the test suite:
```bash
make test
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests
5. Submit a pull request

## License

Apache License 2.0