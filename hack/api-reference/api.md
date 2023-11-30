<p>Packages:</p>
<ul>
<li>
<a href="#ironcore.provider.extensions.gardener.cloud%2fv1alpha1">ironcore.provider.extensions.gardener.cloud/v1alpha1</a>
</li>
</ul>
<h2 id="ironcore.provider.extensions.gardener.cloud/v1alpha1">ironcore.provider.extensions.gardener.cloud/v1alpha1</h2>
<p>
<p>Package v1alpha1 contains the ironcore provider API resources.</p>
</p>
Resource Types:
<ul><li>
<a href="#ironcore.provider.extensions.gardener.cloud/v1alpha1.CloudProfileConfig">CloudProfileConfig</a>
</li><li>
<a href="#ironcore.provider.extensions.gardener.cloud/v1alpha1.ControlPlaneConfig">ControlPlaneConfig</a>
</li><li>
<a href="#ironcore.provider.extensions.gardener.cloud/v1alpha1.InfrastructureConfig">InfrastructureConfig</a>
</li></ul>
<h3 id="ironcore.provider.extensions.gardener.cloud/v1alpha1.CloudProfileConfig">CloudProfileConfig
</h3>
<p>
<p>CloudProfileConfig contains provider-specific configuration that is embedded into Gardener&rsquo;s <code>CloudProfile</code>
resource.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
ironcore.provider.extensions.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>CloudProfileConfig</code></td>
</tr>
<tr>
<td>
<code>machineImages</code></br>
<em>
<a href="#ironcore.provider.extensions.gardener.cloud/v1alpha1.MachineImages">
[]MachineImages
</a>
</em>
</td>
<td>
<p>MachineImages is the list of machine images that are understood by the controller. It maps
logical names and versions to provider-specific identifiers.</p>
</td>
</tr>
<tr>
<td>
<code>regionConfigs</code></br>
<em>
<a href="#ironcore.provider.extensions.gardener.cloud/v1alpha1.RegionConfig">
[]RegionConfig
</a>
</em>
</td>
<td>
<p>RegionConfigs is the list of supported regions.</p>
</td>
</tr>
<tr>
<td>
<code>storageClasses</code></br>
<em>
<a href="#ironcore.provider.extensions.gardener.cloud/v1alpha1.StorageClasses">
StorageClasses
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>StorageClasses defines the DefaultStrorageClass and AdditionalStoreClasses for the shoot</p>
</td>
</tr>
</tbody>
</table>
<h3 id="ironcore.provider.extensions.gardener.cloud/v1alpha1.ControlPlaneConfig">ControlPlaneConfig
</h3>
<p>
<p>ControlPlaneConfig contains configuration settings for the control plane.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
ironcore.provider.extensions.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>ControlPlaneConfig</code></td>
</tr>
<tr>
<td>
<code>cloudControllerManager</code></br>
<em>
<a href="#ironcore.provider.extensions.gardener.cloud/v1alpha1.CloudControllerManagerConfig">
CloudControllerManagerConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>CloudControllerManager contains configuration settings for the cloud-controller-manager.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="ironcore.provider.extensions.gardener.cloud/v1alpha1.InfrastructureConfig">InfrastructureConfig
</h3>
<p>
<p>InfrastructureConfig infrastructure configuration resource</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
ironcore.provider.extensions.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>InfrastructureConfig</code></td>
</tr>
<tr>
<td>
<code>networkRef</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.26/#localobjectreference-v1-core">
Kubernetes core/v1.LocalObjectReference
</a>
</em>
</td>
<td>
<p>NetworkRef references the network to use for the Shoot creation.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="ironcore.provider.extensions.gardener.cloud/v1alpha1.CloudControllerManagerConfig">CloudControllerManagerConfig
</h3>
<p>
(<em>Appears on:</em>
<a href="#ironcore.provider.extensions.gardener.cloud/v1alpha1.ControlPlaneConfig">ControlPlaneConfig</a>)
</p>
<p>
<p>CloudControllerManagerConfig contains configuration settings for the cloud-controller-manager.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>featureGates</code></br>
<em>
map[string]bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>FeatureGates contains information about enabled feature gates.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="ironcore.provider.extensions.gardener.cloud/v1alpha1.InfrastructureStatus">InfrastructureStatus
</h3>
<p>
<p>InfrastructureStatus contains information about created infrastructure resources.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>networkRef</code></br>
<em>
<a href="https://github.com/ironcore-dev/ironcore/blob/main/docs/api-reference/common.md#localuidreference">
github.com/ironcore-dev/ironcore/api/common/v1alpha1.LocalUIDReference
</a>
</em>
</td>
<td>
<p>NetworkRef is the reference to the networked used</p>
</td>
</tr>
<tr>
<td>
<code>natGatewayRef</code></br>
<em>
<a href="https://github.com/ironcore-dev/ironcore/blob/main/docs/api-reference/common.md#localuidreference">
github.com/ironcore-dev/ironcore/api/common/v1alpha1.LocalUIDReference
</a>
</em>
</td>
<td>
<p>NATGatewayRef is the reference to the NAT gateway used</p>
</td>
</tr>
<tr>
<td>
<code>prefixRef</code></br>
<em>
<a href="https://github.com/ironcore-dev/ironcore/blob/main/docs/api-reference/common.md#localuidreference">
github.com/ironcore-dev/ironcore/api/common/v1alpha1.LocalUIDReference
</a>
</em>
</td>
<td>
<p>PrefixRef is the reference to the Prefix used</p>
</td>
</tr>
</tbody>
</table>
<h3 id="ironcore.provider.extensions.gardener.cloud/v1alpha1.MachineImage">MachineImage
</h3>
<p>
(<em>Appears on:</em>
<a href="#ironcore.provider.extensions.gardener.cloud/v1alpha1.WorkerStatus">WorkerStatus</a>)
</p>
<p>
<p>MachineImage is a mapping from logical names and versions to ironcore-specific identifiers.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the logical name of the machine image.</p>
</td>
</tr>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<p>Version is the logical version of the machine image.</p>
</td>
</tr>
<tr>
<td>
<code>image</code></br>
<em>
string
</em>
</td>
<td>
<p>Image is the path to the image.</p>
</td>
</tr>
<tr>
<td>
<code>architecture</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Architecture is the CPU architecture of the machine image.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="ironcore.provider.extensions.gardener.cloud/v1alpha1.MachineImageVersion">MachineImageVersion
</h3>
<p>
(<em>Appears on:</em>
<a href="#ironcore.provider.extensions.gardener.cloud/v1alpha1.MachineImages">MachineImages</a>)
</p>
<p>
<p>MachineImageVersion contains a version and a provider-specific identifier.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<p>Version is the version of the image.</p>
</td>
</tr>
<tr>
<td>
<code>image</code></br>
<em>
string
</em>
</td>
<td>
<p>Image is the path to the image.</p>
</td>
</tr>
<tr>
<td>
<code>architecture</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Architecture is the CPU architecture of the machine image.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="ironcore.provider.extensions.gardener.cloud/v1alpha1.MachineImages">MachineImages
</h3>
<p>
(<em>Appears on:</em>
<a href="#ironcore.provider.extensions.gardener.cloud/v1alpha1.CloudProfileConfig">CloudProfileConfig</a>)
</p>
<p>
<p>MachineImages is a mapping from logical names and versions to provider-specific identifiers.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the logical name of the machine image.</p>
</td>
</tr>
<tr>
<td>
<code>versions</code></br>
<em>
<a href="#ironcore.provider.extensions.gardener.cloud/v1alpha1.MachineImageVersion">
[]MachineImageVersion
</a>
</em>
</td>
<td>
<p>Versions contains versions and a provider-specific identifier.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="ironcore.provider.extensions.gardener.cloud/v1alpha1.RegionConfig">RegionConfig
</h3>
<p>
(<em>Appears on:</em>
<a href="#ironcore.provider.extensions.gardener.cloud/v1alpha1.CloudProfileConfig">CloudProfileConfig</a>)
</p>
<p>
<p>RegionConfig is the definition of a region.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of a region.</p>
</td>
</tr>
<tr>
<td>
<code>server</code></br>
<em>
string
</em>
</td>
<td>
<p>Server is the server endpoint of this region.</p>
</td>
</tr>
<tr>
<td>
<code>certificateAuthorityData</code></br>
<em>
[]byte
</em>
</td>
<td>
<p>CertificateAuthorityData is the CA data of the region server.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="ironcore.provider.extensions.gardener.cloud/v1alpha1.StorageClass">StorageClass
</h3>
<p>
(<em>Appears on:</em>
<a href="#ironcore.provider.extensions.gardener.cloud/v1alpha1.StorageClasses">StorageClasses</a>)
</p>
<p>
<p>StorageClass is a definition of a storageClass</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the storageclass</p>
</td>
</tr>
<tr>
<td>
<code>type</code></br>
<em>
string
</em>
</td>
<td>
<p>Type is referring to the VolumeClass to use for this StorageClass</p>
</td>
</tr>
</tbody>
</table>
<h3 id="ironcore.provider.extensions.gardener.cloud/v1alpha1.StorageClasses">StorageClasses
</h3>
<p>
(<em>Appears on:</em>
<a href="#ironcore.provider.extensions.gardener.cloud/v1alpha1.CloudProfileConfig">CloudProfileConfig</a>)
</p>
<p>
<p>StorageClasses is a definition of a storageClasses</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>default</code></br>
<em>
<a href="#ironcore.provider.extensions.gardener.cloud/v1alpha1.StorageClass">
StorageClass
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Default defines the default storage class for the shoot</p>
</td>
</tr>
<tr>
<td>
<code>additional</code></br>
<em>
<a href="#ironcore.provider.extensions.gardener.cloud/v1alpha1.StorageClass">
[]StorageClass
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional defines the additional storage classes for the shoot</p>
</td>
</tr>
</tbody>
</table>
<h3 id="ironcore.provider.extensions.gardener.cloud/v1alpha1.WorkerStatus">WorkerStatus
</h3>
<p>
<p>WorkerStatus contains information about created worker resources.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>machineImages</code></br>
<em>
<a href="#ironcore.provider.extensions.gardener.cloud/v1alpha1.MachineImage">
[]MachineImage
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>MachineImages is a list of machine images that have been used in this worker. Usually, the extension controller
gets the mapping from name/version to the provider-specific machine image data in its componentconfig. However, if
a version that is still in use gets removed from this componentconfig it cannot reconcile anymore existing <code>Worker</code>
resources that are still using this version. Hence, it stores the used versions in the provider status to ensure
reconciliation is possible.</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <a href="https://github.com/ahmetb/gen-crd-api-reference-docs">gen-crd-api-reference-docs</a>
</em></p>
