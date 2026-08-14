<p>Packages:</p>
<ul>
<li>
<a href="#ironcore.provider.extensions.gardener.cloud%2fv1alpha1">ironcore.provider.extensions.gardener.cloud/v1alpha1</a>
</li>
</ul>

<h2 id="ironcore.provider.extensions.gardener.cloud/v1alpha1">ironcore.provider.extensions.gardener.cloud/v1alpha1</h2>
<p>

</p>

<h3 id="cloudcontrollermanagerconfig">CloudControllerManagerConfig
</h3>


<p>
(<em>Appears on:</em><a href="#controlplaneconfig">ControlPlaneConfig</a>)
</p>

<p>
CloudControllerManagerConfig contains configuration settings for the cloud-controller-manager.
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
object (keys:string, values:boolean)
</em>
</td>
<td>
<em>(Optional)</em>
<p>FeatureGates contains information about enabled feature gates.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="cloudprofileconfig">CloudProfileConfig
</h3>


<p>
CloudProfileConfig contains provider-specific configuration that is embedded into Gardener's `CloudProfile`
resource.
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
<a href="#machineimages">MachineImages</a> array
</em>
</td>
<td>
<p>MachineImages is the list of machine images that are understood by the controller. It maps<br />logical names and versions to provider-specific identifiers.</p>
</td>
</tr>
<tr>
<td>
<code>regionConfigs</code></br>
<em>
<a href="#regionconfig">RegionConfig</a> array
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
<a href="#storageclasses">StorageClasses</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>StorageClasses defines the DefaultStrorageClass and AdditionalStoreClasses for the shoot</p>
</td>
</tr>

</tbody>
</table>


<h3 id="controlplaneconfig">ControlPlaneConfig
</h3>


<p>
ControlPlaneConfig contains configuration settings for the control plane.
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
<code>cloudControllerManager</code></br>
<em>
<a href="#cloudcontrollermanagerconfig">CloudControllerManagerConfig</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>CloudControllerManager contains configuration settings for the cloud-controller-manager.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="infrastructureconfig">InfrastructureConfig
</h3>


<p>
InfrastructureConfig infrastructure configuration resource
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
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.36/#localobjectreference-v1-core">LocalObjectReference</a>
</em>
</td>
<td>
<p>NetworkRef references the network to use for the Shoot creation.</p>
</td>
</tr>
<tr>
<td>
<code>natPortsPerNetworkInterface</code></br>
<em>
integer
</em>
</td>
<td>
<p>NATPortsPerNetworkInterface defines the minimum number of ports per network interface the NAT gateway should use.<br />Has to be a power of 2. If empty, 2048 is the default.</p>
</td>
</tr>
<tr>
<td>
<code>networkPolicyRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/ironcore-dev/ironcore/api/common/v1alpha1#LocalUIDReference">LocalUIDReference</a>
</em>
</td>
<td>
<p>NetworkPolicy is reference to the NetworkPolicy to use for the Shoot creation.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="infrastructurestatus">InfrastructureStatus
</h3>


<p>
InfrastructureStatus contains information about created infrastructure resources.
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
<a href="https://pkg.go.dev/github.com/ironcore-dev/ironcore/api/common/v1alpha1#LocalUIDReference">LocalUIDReference</a>
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
<a href="https://pkg.go.dev/github.com/ironcore-dev/ironcore/api/common/v1alpha1#LocalUIDReference">LocalUIDReference</a>
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
<a href="https://pkg.go.dev/github.com/ironcore-dev/ironcore/api/common/v1alpha1#LocalUIDReference">LocalUIDReference</a>
</em>
</td>
<td>
<p>PrefixRef is the reference to the Prefix used</p>
</td>
</tr>
<tr>
<td>
<code>networkPolicyRef</code></br>
<em>
<a href="https://pkg.go.dev/github.com/ironcore-dev/ironcore/api/common/v1alpha1#LocalUIDReference">LocalUIDReference</a>
</em>
</td>
<td>
<p>NetworkPolicy is reference to the NetworkPolicy defined</p>
</td>
</tr>

</tbody>
</table>


<h3 id="machineimage">MachineImage
</h3>


<p>
(<em>Appears on:</em><a href="#workerstatus">WorkerStatus</a>)
</p>

<p>
MachineImage is a mapping from logical names and versions to ironcore-specific identifiers.
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


<h3 id="machineimageversion">MachineImageVersion
</h3>


<p>
(<em>Appears on:</em><a href="#machineimages">MachineImages</a>)
</p>

<p>
MachineImageVersion contains a version and a provider-specific identifier.
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


<h3 id="machineimages">MachineImages
</h3>


<p>
(<em>Appears on:</em><a href="#cloudprofileconfig">CloudProfileConfig</a>)
</p>

<p>
MachineImages is a mapping from logical names and versions to provider-specific identifiers.
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
<a href="#machineimageversion">MachineImageVersion</a> array
</em>
</td>
<td>
<p>Versions contains versions and a provider-specific identifier.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="regionconfig">RegionConfig
</h3>


<p>
(<em>Appears on:</em><a href="#cloudprofileconfig">CloudProfileConfig</a>)
</p>

<p>
RegionConfig is the definition of a region.
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
integer array
</em>
</td>
<td>
<p>CertificateAuthorityData is the base64-encoded CA data of the region server.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="storageclass">StorageClass
</h3>


<p>
(<em>Appears on:</em><a href="#storageclasses">StorageClasses</a>)
</p>

<p>
StorageClass is a definition of a storageClass
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


<h3 id="storageclasses">StorageClasses
</h3>


<p>
(<em>Appears on:</em><a href="#cloudprofileconfig">CloudProfileConfig</a>)
</p>

<p>
StorageClasses is a definition of a storageClasses
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
<a href="#storageclass">StorageClass</a>
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
<a href="#storageclass">StorageClass</a> array
</em>
</td>
<td>
<em>(Optional)</em>
<p>Additional defines the additional storage classes for the shoot</p>
</td>
</tr>

</tbody>
</table>


<h3 id="workerstatus">WorkerStatus
</h3>


<p>
WorkerStatus contains information about created worker resources.
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
<a href="#machineimage">MachineImage</a> array
</em>
</td>
<td>
<em>(Optional)</em>
<p>MachineImages is a list of machine images that have been used in this worker. Usually, the extension controller<br />gets the mapping from name/version to the provider-specific machine image data in its componentconfig. However, if<br />a version that is still in use gets removed from this componentconfig it cannot reconcile anymore existing `Worker`<br />resources that are still using this version. Hence, it stores the used versions in the provider status to ensure<br />reconciliation is possible.</p>
</td>
</tr>

</tbody>
</table>


