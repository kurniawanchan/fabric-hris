{{/*
Standard name/fullname helpers (the conventional chart-scaffold pattern —
not a bespoke invention). Kept intentionally minimal: this chart has no
subchart dependencies to disambiguate against (see README.md "Why
standalone").
*/}}

{{- define "ipfs-private-cluster.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "ipfs-private-cluster.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}
{{- end -}}

{{- define "ipfs-private-cluster.labels" -}}
app.kubernetes.io/name: {{ include "ipfs-private-cluster.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" }}
{{- end -}}

{{/* Resolve the swarm-key Secret name: explicit override, else a chart-owned default. Never generated from values — the key material itself never passes through this chart's templates. */}}
{{- define "ipfs-private-cluster.swarmKeySecretName" -}}
{{- if .Values.swarmKey.existingSecretName -}}
{{- .Values.swarmKey.existingSecretName -}}
{{- else -}}
{{- printf "%s-ipfs-swarm-key" (include "ipfs-private-cluster.fullname" .) -}}
{{- end -}}
{{- end -}}

{{- define "ipfs-private-cluster.clusterSecretName" -}}
{{- if .Values.clusterSecret.existingSecretName -}}
{{- .Values.clusterSecret.existingSecretName -}}
{{- else -}}
{{- printf "%s-ipfs-cluster-secret" (include "ipfs-private-cluster.fullname" .) -}}
{{- end -}}
{{- end -}}

{{/* Stable per-ordinal kubo pod DNS name, used both by the kubo StatefulSet's own hostname and by the paired ipfs-cluster peer's CLUSTER_IPFSHTTP_NODEMULTIADDRESS. Centralized here so the naming convention is defined exactly once. */}}
{{- define "ipfs-private-cluster.kuboPodFQDN" -}}
{{- $ctx := index . 0 -}}
{{- $ordinal := index . 1 -}}
{{- printf "%s-kubo-%d.%s-kubo-headless.%s.svc.cluster.local" (include "ipfs-private-cluster.fullname" $ctx) $ordinal (include "ipfs-private-cluster.fullname" $ctx) $ctx.Release.Namespace -}}
{{- end -}}
