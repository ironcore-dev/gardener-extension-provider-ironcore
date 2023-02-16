# Using the onmetal provider extension with Gardener as end-user

The [`core.gardener.cloud/v1beta1.Shoot` resource](https://github.com/gardener/gardener/blob/master/example/90-shoot.yaml) declares a few fields that are meant to contain provider-specific configuration.

This document describes the configurable options for onmetal and provides an example `Shoot` manifest with minimal configuration that can be used to create a onmetal cluster (modulo the landscape-specific information like cloud profile names, secret binding names, etc.).

## onmetal Provider Credentials

In order for Gardener to create a Kubernetes cluster using onmetal infrastructure components, a Shoot has to provide credentials with sufficient permissions to the desired onmetal project.

In Onmetal provider extension, `kubeconfig` is generated from user data. And a secret named `cloudProvider` is created with `kubeconfig`, `token` and `namespace` as data keys of the secret. A sample file below

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: cloudProvider
  namespace: garden-dev
type: Opaque
data:
  namespace: "garden-dev"
  token: "foo"
  kubeconfig: SDer!45skdjfrtYHTL#^GY            // kubeconfig content base64 encoded
```

## `InfrastructureConfig`

The infrastructure configuration mainly describes how the network layout looks like in order to create the shoot worker nodes in a later step, thus, prepares everything relevant to create VMs, load balancers, volumes, etc.

An example `InfrastructureConfig` for the Onmetal extension looks as follows:

```yaml
apiVersion: onmetal.provider.extensions.gardener.cloud/v1alpha1
kind: InfrastructureConfig
networkRef:
  name: "my-network"
```

`networkRef` NetworkRef references the network to use for the Shoot creation.


## `ControlPlaneConfig`

The control plane configuration mainly contains values for the Onmetal-specific control plane components.
Today, the only component deployed by the Onmetal extension is the `cloud-controller-manager`.

An example `ControlPlaneConfig` for the Onmetal extension looks as follows:

```yaml
apiVersion: onmetal.provider.extensions.gardener.cloud/v1alpha1
kind: ControlPlaneConfig
zone: europe-west1-b
cloudControllerManager:
  featureGates:
    CustomResourceValidation: true
```

The `zone` field tells the cloud-controller-manager in which zone it should mainly operate.
You can still create clusters in multiple availability zones, however, the cloud-controller-manager requires one "main" zone.
:warning: You always have to specify this field!

The `cloudControllerManager.featureGates` contains a map of explicitly enabled or disabled feature gates.
For production usage it's not recommend to use this field at all as you can enable alpha features or disable beta/stable features, potentially impacting the cluster stability.
If you don't want to configure anything for the `cloudControllerManager` simply omit the key in the YAML specification.

## WorkerConfig

The worker configuration contains:

* Local SSD interface for the additional volumes attached to GCP worker machines.

  If you attach the disk with `SCRATCH` type, either an `NVMe` interface or a `SCSI` interface must be specified.
  It is only meaningful to provide this volume interface if only `SCRATCH` data volumes are used.
* Service Account with their specified scopes, authorized for this worker.

  Service accounts created in advance that generate access tokens that can be accessed through the metadata server and used to authenticate applications on the instance.

* GPU with its type and count per node. This will attach that GPU to all the machines in the worker grp

  **Note**: 
  * A rolling upgrade of the worker group would be triggered in case the `acceleratorType` or `count` is updated.
  * Some machineTypes like [a2 family](https://cloud.google.com/blog/products/compute/announcing-google-cloud-a2-vm-family-based-on-nvidia-a100-gpu) come with already attached gpu of `a100` type and pre-defined count. If your workerPool consists of those machineTypes, please **do not** specify any GPU configuration.
  * Sufficient quota of gpu is needed in the GCP project. This includes quota to support autoscaling if enabled.
  * GPU-attached machines can't be live migrated during host maintenance events. Find out how to handle that in your application [here](https://cloud.google.com/compute/docs/gpus/gpu-host-maintenance)
  * GPU count specified here is considered for forming node template during scale-from-zero in Cluster Autoscaler

  An example `WorkerConfig` for the GCP looks as follows:

```yaml
apiVersion: gcp.provider.extensions.gardener.cloud/v1alpha1
kind: WorkerConfig
volume:
  interface: NVME
serviceAccount:
  email: foo@bar.com
  scopes:
  - https://www.googleapis.com/auth/cloud-platform
gpu:
  acceleratorType: nvidia-tesla-t4
  count: 1
```
## Example `Shoot` manifest

 An example to a `Shoot` manifest [here](https://github.com/onmetal/gardener-extension-provider-onmetal/blob/doc/usage-as-operator/docs/usage-as-operator.md):

## CSI volume provisioners

Every Onmetal shoot cluster that has at least Kubernetes v1.24 will be deployed with the `onmetal-csi-driver`.

End-users might want to update their custom `StorageClass`es to the new `onmetal-csi-driver` provisioner.


