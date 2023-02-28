{{- define "csi-driver-node.extensionsGroup" -}}
extensions.gardener.cloud
{{- end -}}

{{- define "csi-driver-node.name" -}}
provider-onmetal
{{- end -}}

{{- define "csi-driver-node.provisioner" -}}
csi.onmetal.de
{{- end -}}

{{- define "csi-driver-node.storageversion" -}}
storage.k8s.io/v1
{{- end -}}
