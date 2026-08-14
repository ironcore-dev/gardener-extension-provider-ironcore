<p>Packages:</p>
<ul>
<li>
<a href="#ironcore.provider.extensions.config.gardener.cloud%2fv1alpha1">ironcore.provider.extensions.config.gardener.cloud/v1alpha1</a>
</li>
</ul>

<h2 id="ironcore.provider.extensions.config.gardener.cloud/v1alpha1">ironcore.provider.extensions.config.gardener.cloud/v1alpha1</h2>
<p>

</p>

<h3 id="backupbucketconfig">BackupBucketConfig
</h3>


<p>
(<em>Appears on:</em><a href="#controllerconfiguration">ControllerConfiguration</a>)
</p>

<p>
BackupBucketConfig is config for Backup Bucket
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
<code>bucketClassName</code></br>
<em>
string
</em>
</td>
<td>
<p>BucketClassName is the name of the ironcore BucketClass to use for the BackupBucket</p>
</td>
</tr>

</tbody>
</table>


<h3 id="bastionconfig">BastionConfig
</h3>


<p>
(<em>Appears on:</em><a href="#controllerconfiguration">ControllerConfiguration</a>)
</p>

<p>
BastionConfig is the config for the Bastion
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
<code>image</code></br>
<em>
string
</em>
</td>
<td>
<p>Image is the URL pointing to an OCI registry containing the operating system image which should be used to boot the Bastion host</p>
</td>
</tr>
<tr>
<td>
<code>machineClassName</code></br>
<em>
string
</em>
</td>
<td>
<p>MachineClassName is the name of the ironcore MachineClass to use for the Bastion host</p>
</td>
</tr>
<tr>
<td>
<code>volumeClassName</code></br>
<em>
string
</em>
</td>
<td>
<p>VolumeClassName is the name of the ironcore VolumeClass to use for the Bastion host root disk volume</p>
</td>
</tr>

</tbody>
</table>


<h3 id="controllerconfiguration">ControllerConfiguration
</h3>


<p>
ControllerConfiguration defines the configuration for the ironcore provider.
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
<code>clientConnection</code></br>
<em>
<a href="https://pkg.go.dev/k8s.io/component-base/config/v1alpha1#ClientConnectionConfiguration">ClientConnectionConfiguration</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>ClientConnection specifies the kubeconfig file and client connection<br />settings for the proxy server to use when communicating with the apiserver.</p>
</td>
</tr>
<tr>
<td>
<code>etcd</code></br>
<em>
<a href="#etcd">ETCD</a>
</em>
</td>
<td>
<p>ETCD is the etcd configuration.</p>
</td>
</tr>
<tr>
<td>
<code>healthCheckConfig</code></br>
<em>
<a href="https://pkg.go.dev/github.com/gardener/gardener/extensions/pkg/apis/config/v1alpha1#HealthCheckConfig">HealthCheckConfig</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>HealthCheckConfig is the config for the health check controller</p>
</td>
</tr>
<tr>
<td>
<code>featureGates</code></br>
<em>
object (keys:string, values:boolean)
</em>
</td>
<td>
<em>(Optional)</em>
<p>FeatureGates is a map of feature names to bools that enable<br />or disable alpha/experimental features.<br />Default: nil</p>
</td>
</tr>
<tr>
<td>
<code>bastionConfig</code></br>
<em>
<a href="#bastionconfig">BastionConfig</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>BastionConfig is the config for the Bastion</p>
</td>
</tr>
<tr>
<td>
<code>backupBucketConfig</code></br>
<em>
<a href="#backupbucketconfig">BackupBucketConfig</a>
</em>
</td>
<td>
<p>BackupBucketConfig is config for Backup Bucket</p>
</td>
</tr>

</tbody>
</table>


<h3 id="etcd">ETCD
</h3>


<p>
(<em>Appears on:</em><a href="#controllerconfiguration">ControllerConfiguration</a>)
</p>

<p>
ETCD is an etcd configuration.
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
<code>storage</code></br>
<em>
<a href="#etcdstorage">ETCDStorage</a>
</em>
</td>
<td>
<p>ETCDStorage is the etcd storage configuration.</p>
</td>
</tr>
<tr>
<td>
<code>backup</code></br>
<em>
<a href="#etcdbackup">ETCDBackup</a>
</em>
</td>
<td>
<p>ETCDBackup is the etcd backup configuration.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="etcdbackup">ETCDBackup
</h3>


<p>
(<em>Appears on:</em><a href="#etcd">ETCD</a>)
</p>

<p>
ETCDBackup is an etcd backup configuration.
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
<code>schedule</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Schedule is the etcd backup schedule.</p>
</td>
</tr>

</tbody>
</table>


<h3 id="etcdstorage">ETCDStorage
</h3>


<p>
(<em>Appears on:</em><a href="#etcd">ETCD</a>)
</p>

<p>
ETCDStorage is an etcd storage configuration.
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
<code>className</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ClassName is the name of the storage class used in etcd-main volume claims.</p>
</td>
</tr>
<tr>
<td>
<code>capacity</code></br>
<em>
<a href="https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.36/#quantity-resource-api">Quantity</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Capacity is the storage capacity used in etcd-main volume claims.</p>
</td>
</tr>

</tbody>
</table>


