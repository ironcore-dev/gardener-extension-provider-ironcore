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
{{- if semverCompare "<= 1.18.x" .Values.kubernetesVersion -}}
storage.k8s.io/v1beta1
{{- else -}}
storage.k8s.io/v1
{{- end -}}
{{- end -}}
