# OCI CAPI Operator

This operator automates the deployment and management of Cluster API (CAPI) and cluster autoscaler components for OpenShift (OCP) clusters deployed with Oracle Cloud Infrastructure (OCI), enabling seamless cluster autoscaling.

## Description

The OCI CAPI Operator deploys all of the necessary components in order to enable autoscaling in an OCI OCP cluster

- CAPI and CAPOCI installation management
- [Cluster-autoscaler](https://github.com/kubernetes/autoscaler/tree/master/cluster-autoscaler) deployment and configuration
- CAPI CRs to manager the OCI cluster
- Certificate approver for new nodes

See @javipolo's [blog](https://github.com/javipolo/openshift-oci-capi-autoscaling/blob/blog_cursor/BLOG.md) for more details.

## Getting Started

### Prerequisites

- OpenShift cluster running on OCI (see instructions for creating an OCI OCP cluster [here](https://docs.redhat.com/en/documentation/openshift_container_platform/4.19/html/installing_on_oci/installing-oci-assisted-installer#installing-oci-about-assisted-installer_installing-oci-assisted-installer))
- The `kubeconfig` file for this cluster downloaded locally
- Oracle Cloud CLI ([oci-cli](https://github.com/oracle/oci-cli)) installed
- [oc](https://docs.redhat.com/en/documentation/openshift_container_platform/4.19/html/cli_tools/openshift-cli-oc) CLI installed


### Configuration

#### Step 1: Create a private Key and upload public key to OCI

The following command creates public and private key files in the `~/.oci` directory
```sh
mkdir ~/.oci
openssl genrsa -out ~/.oci/oci_api_key.pem 2048
chmod go-rwx ~/.oci/oci_api_key.pem
openssl rsa -pubout -in ~/.oci/oci_api_key.pem -out ~/.oci/oci_api_key_public.pem
```

Public Key file: ` ~/.oci/oci_api_key_public.pem`

Private Key file: ` ~/.oci/oci_api_key.pem`
    
Upload API key to [OCI Console](https://cloud.oracle.com/identity/domains/my-profile/auth-tokens) > choose public key file

Additional information: 
  - [Full instructions](https://docs.oracle.com/en-us/iaas/Content/API/Concepts/apisigningkey.htm#Required_Keys_and_OCIDs)
  - Script above comes from [Generate API Signing Key](https://docs.oracle.com/en-us/iaas/Content/API/Concepts/apisigningkey.htm#apisigningkey_topic_How_to_Generate_an_API_Signing_Key_Mac_Linux)


#### Step 2: Configure OCI CLI

**Requires**: 
- User ID https://cloud.oracle.com/identity/domains/my-profile
- Tenancy ID https://cloud.oracle.com/tenancy 
- Region (e.g. `us-sanjose-1`)
- Path to Private Key file created in step 1

Run the following to configure the `oci` CLI:
```sh
oci setup config
```

#### Step 3: Upload an RHCOS Image [src](https://github.com/javipolo/openshift-oci-capi-autoscaling/blob/blog_cursor/BLOG.md#step-1-create-custom-rhcos-image)

This RHCOS image is used during autoscaling when installing new worker nodes to this cluster.

You need a custom Red Hat CoreOS image in your OCI tenancy for the autoscaling nodes:

  1. Download RHCOS Image (OpenStack flavor) that matches your OpenShift version: In this example, our version is 4.19.0
    
      ```sh
      curl -LO https://mirror.openshift.com/pub/openshift-v4/x86_64/dependencies/rhcos/4.19/4.19.0/rhcos-4.19.0-x86_64-openstack.x86_64.qcow2.gz
      gzip -d rhcos-4.19.0-x86_64-openstack.x86_64.qcow2
      ```

  2. Upload the qcow2 file to OCI and create a custom image
      - Follow Oracle's documentation: [Create a Custom Linux Image](https://docs.public.oneportal.content.oci.oraclecloud.com/en-us/iaas/compute-cloud-at-customer/topics/images/importing-custom-linux-imges.htm)
      - Note the image name 

#### Step 4: Clone this repo and fill out config.sh file

```sh
git clone https://github.com/CrystalChun/oci-capi-operator-test
cd oci-capi-operator-test
```

In `config.sh` fill out these mandatory variables: 
```sh
compartment_name= 
image_name= # this is the name of the image you uploaded in step 3
```


Ensure you're authenticated into your OCI OCP Cluster:
```sh
export KUBECONFIG= # Path to OCI OCP Cluster Kubeconfig file
```

Source the config file:
```sh
source config.sh
```

This should auto-fill/retrieve all the environment variables needed to deploy the operator.

### Deploy the operator

Optional: Build the operator image from this repository
```sh
export IMG= # set to image for the oci capi operator
make build-image

# Optionally, push the built image to a remote image store
make push-image
```

Deploy the OCI CAPI operator
```sh
export IMG= # set to image for the oci capi operator
make deploy
```

By default, the operator will be deployed in the `oci-capi-operator` namespace.

### Use the operator

Apply the `ociclusterautoscalers` CR to deploy all of the workloads needed for autoscaling:

```sh
make apply
```

#### Verify Operator Deployments

The operator will deploy:
- CAPI
- CAPOCI
- Cluster Autoscaler 
- CAPI CRs for autoscaling this cluster: `OCICluster`, `Cluster`, `OCIMachineTemplate`, `MachineDeployment`

Run the following to verify:
```sh
# Should have both CAPI and Autoscaler deployments
oc get po -n capi-system 
oc get po -n cluster-api-provider-oci-system
oc get cluster -n capi-system
oc get ocicluster -n capi-system
oc get machinedeployment -n capi-system 
oc get ocimachinetemplate -n capi-system
```


**NOTE**: The `ociclusterautoscalers` CR has spec fields that are not required, but can be used to override the default values of the deployments.
See the [ociclusterautoscaler.md doc](./docs/ociclusterautoscaler.md) for more details.

### Testing autoscaling

To test if autoscaling works, a deployment (for example) needs to exceed the amount of resources currently available in this cluster.
```sh
oc create deployment nginx --namespace default --image=docker.io/nginx:latest --replicas=0
oc set resources deployment -n default nginx --requests=memory=2Gi
oc scale deployment -n default nginx --replicas=20
```

Watch the status of the following resources:
```sh
oc get md -n capi-system
oc get machineset -n capi-system
oc get ocicluster -n capi-system
oc get ocimachine -n capi-system
```

New worker node(s) should be installed
```sh
oc get no
```

To test scale down:
```sh
oc scale deployment -n default nginx --replicas=0
```

### Cleanup

#### Scale down completely

To ensure no instances are left dangling, completely scale down CAPI-installed instances by:
1. Scaling down the nginx deployment 
    ```sh
    oc scale deployment -n default nginx --replicas=0
    ```
2. Scale down the MachineDeployment, if necessary
    ```sh
    oc scale md -n capi-system <md name> --replicas=0
    ```
3. Ensure all the `OCIMachine` CRs are removed
    ```sh
    oc get ocimachine -n capi-system
    ```

#### Remove the CAPI and Autoscaler deployments

Remove the OCIClusterAutoscaler CR:
```sh
make remove-cr
```

This deletes the `ociclusterautoscaler` instance, which triggers the operator to cleanup all of the resources it created.

Check the pod logs of the oci-capi-operator:
```sh
oc logs -n oci-capi-operator deploy/oci-capi-operator-controller-manager
```

Verify all resources are removed by re-running [the verification step](#verify-operator-deployments) and ensuring no resources appear.

#### Remove the operator

To uninstall this operator and all of its components, run the following:
```sh
make undeploy
```

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
