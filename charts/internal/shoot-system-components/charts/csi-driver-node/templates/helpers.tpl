{{- define "csi-driver-node.extensionsGroup" -}}
extensions.gardener.cloud
{{- end -}}

{{- define "csi-driver-node.name" -}}
provider-ironcore
{{- end -}}

{{- define "csi-driver-node.provisioner" -}}
csi.ironcore.dev
{{- end -}}

{{- define "csi-driver-node.storageversion" -}}
storage.k8s.io/v1
{{- end -}}
