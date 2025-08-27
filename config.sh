#!/bin/bash

# Required Config Variables
compartment_name=
image_name=


# Auto-detected 
export OCI_TENANCY_ID="\"$(grep -E "^tenancy=" ~/.oci/config | cut -d = -f 2)\""
export OCI_USER_ID="\"$(grep -E "^user=" ~/.oci/config | cut -d = -f 2)\""
export OCI_REGION="\"$(grep -E "^region=" ~/.oci/config | cut -d = -f 2)\""
export OCI_CREDENTIALS_FINGERPRINT="$(grep -E "^fingerprint=" ~/.oci/config | cut -d = -f 2)"
export OCI_CREDENTIALS_KEY="$(grep -E "^key_file=" ~/.oci/config | cut -d = -f 2)"
export privateKey="\"$(base64 < $OCI_CREDENTIALS_KEY | tr -d '\n')\""
export fingerprint="\"$(echo $OCI_CREDENTIALS_FINGERPRINT | tr -d '\n' | base64)\""
export passphrase="" # passphrase for api key, leave empty if did not set during generation

compartment_id="$(oci iam compartment list --all --compartment-id-in-subtree true --access-level ACCESSIBLE --raw-output --query "data[?name=='$compartment_name'].id | [0]")"
nsg_name=cluster-compute-nsg
subnet_name=private
cluster_name=$(oc get infrastructure cluster -ojsonpath='{.status.infrastructureName}')
oci_cluster_name=$(echo "$cluster_name" | rev | cut -d - -f 2- | rev)
vcn_id="$(oci network vcn list --compartment-id "$compartment_id" --display-name "$oci_cluster_name" | jq -r '.data[0].id')"

# IMAGE_ID is the OCID of the RHCOS image to use for new worker nodes added to the cluster
export IMAGE_ID="\"$(oci compute image list --compartment-id "$compartment_id" --display-name "$image_name" | jq -r '.data[0].id')\""

export SUBNET_ID="\"$(oci network subnet list --compartment-id "$compartment_id" --vcn-id "$vcn_id" --display-name "$subnet_name" | jq -r '.data[0].id')\""
export NSG_ID="\"$(oci network nsg list --compartment-id "$compartment_id" --vcn-id "$vcn_id" --display-name "$nsg_name" | jq -r '.data[0].id')\""
export API_LB_ID="\"$(oci lb load-balancer list --compartment-id $compartment_id --display-name ${oci_cluster_name}-openshift_api_apps_lb | jq -r '.data[].id')\""
export API_LB_IP="\"$(oci lb load-balancer list --compartment-id $compartment_id --display-name ${oci_cluster_name}-openshift_api_apps_lb | jq -r '.data[]."ip-addresses"[] | select(."is-public" == true) | ."ip-address"')\""
export SVC_CIDR="\"$(oc get network.config.openshift.io cluster -o jsonpath='{.spec.serviceNetwork[*]}')\""
export CLUSTER_CIDR="\"$(oc get network.config.openshift.io cluster -o jsonpath='{.spec.clusterNetwork[*].cidr}')\""
export COMPARTMENT_ID="\"$compartment_id\""
export VCN_ID="\"$vcn_id\""

# Default Config Variables
export AUTOSCALER_MIN_NODES="\"0\""
export AUTOSCALER_MAX_NODES="\"5\""
export AUTOSCALER_CPUS="\"6\""
export AUTOSCALER_MEMORY="\"16\""
export AUTOSCALER_SHAPE="\"VM.Standard.E4.Flex\""
export OCI_USE_INSTANCE_PRINCIPAL="\"false\""

echo "All Variables set"
