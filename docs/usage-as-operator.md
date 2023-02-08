# Using the Onmetal provider extension with Gardener as operator

The [`core.gardener.cloud/v1beta1.CloudProfile` resource](https://github.com/gardener/gardener/blob/master/example/30-cloudprofile.yaml) declares a `providerConfig` field that is meant to contain provider-specific configuration.
The [`core.gardener.cloud/v1beta1.Seed` resource](https://github.com/gardener/gardener/blob/master/example/50-seed.yaml) is structured similarly.
Additionally, it allows configuring settings for the backups of the main etcds' data of shoot clusters control planes running in this seed cluster.

This document explains the necessary configuration for this provider extension.

## `CloudProfile` resource

This section describes, how the configuration for `CloudProfile`s looks like for onmetal by providing an example `CloudProfile` manifest with minimal configuration that can be used to allow the creation of onmetal shoot clusters.

### `CloudProfileConfig`

The cloud profile configuration contains information about the real machine image IDs in the onmetal environment (image URLs).
You have to map every version that you specify in `.spec.machineImages[].versions` here such that the onmetal extension knows the image URL for every version you want to offer.
For each machine image version an `architecture` field can be specified which specifies the CPU architecture of the machine on which given machine image can be used.

An example `CloudProfileConfig` for the onmetal extension looks as follows:

```yaml
apiVersion: onmetal.provider.extensions.gardener.cloud/v1alpha1
kind: CloudProfileConfig
machineImages:
- name: coreos
  versions:
  - version: 2135.6.0
    image: projects/coreos-cloud/global/images/coreos-stable-2135-6-0-v20190801
    # architecture: amd64 # optional
```

### Example `CloudProfile` manifest

Please find below an example `CloudProfile` manifest:

```yaml
apiVersion: core.gardener.cloud/v1beta1
kind: CloudProfile
metadata:
  name: onmetal
spec:
  type: onmetal
  kubernetes:
    versions:
    - version: 1.25.3
    - version: 1.24.3
  machineImages:
  - name: coreos
    versions:
    - version: 2135.6.0
  regions:
  - region: europe-west1
    names:
    - europe-west1-b
    - europe-west1-c
    - europe-west1-d
  providerConfig:
    apiVersion: onmetal.provider.extensions.gardener.cloud/v1alpha1
    kind: CloudProfileConfig
    machineImages:
    - name: coreos
      versions:
      - version: 2135.6.0
        image: projects/coreos-cloud/global/images/coreos-stable-2135-6-0-v20190801
        # architecture: amd64 # optional
```

## `Seed` resource

This provider extension supports configuration for the `Seed`'s `.spec.provider.type` field.

Please find below an example `Seed` manifest that configures Seed cluster. 

```yaml
---
apiVersion: core.gardener.cloud/v1beta1
kind: Seed
metadata:
  name: my-seed
spec:
  provider:
    type: onmetal
  ...
```

## `Shoot` resource

This provider extension supports configuration for the `Shoot` cluster resource. 
`.spec.provider.workers` field is a list of worker groups.
`.spec.provider.networking.nodes` field is the CIDR of the entire node network. 


```yaml
---
apiVersion: core.gardener.cloud/v1beta1
kind: Shoot
metadata:
  name: my-shoot
spec:
  provider:
    workers: 
    - name: foo
    - name: bar
    networking:
      nodes: "10.0.0.0/24"
  ...
```