{{/*
=============================================================================
_helpers.tpl — deploy/helm/fabric-network (DEP-1)
=============================================================================
Named templates shared across every node type. The two load-bearing ones are
"fabric-network.peer" and "fabric-network.orderer": each renders one
ConfigMap + one headless Service + one StatefulSet for a SINGLE named node,
and is invoked once per list entry (via `range`) from
templates/peers-org1.yaml, templates/orderers-org1.yaml,
templates/peer-org3.yaml, and templates/peer-tenants.yaml. This is the
mechanism DEP-1's DoD requires for the per-tenant peer ("a Helm sub-template/
named template instantiated once per tenant via values.yaml's tenant list")
— reused for org1/org3 too, for DRYness, not because their node counts scale
the same way (see values.yaml's own comment on org1.peers/org1.orderers).

Every `define` below takes a single dict argument (Helm's `include`
mechanism only passes one value) shaped as:
  root  — the root chart context (has .Values/.Release/.Chart/.Files)
  node  — the one values.yaml list entry for this specific node
  org   — a small dict of org-level constants: mspID, kind ("peer"|"orderer")

Cross-org TLS CA union: network-docker-compose.yaml's own header states the
INTENT plainly — "each PEER's clientRootCAs is set to the UNION of all three
peer-org TLS CAs" — with no distinction drawn between Org1/Org3/tenant
peers. `fabric-network.peer` therefore mounts the identical union (org1 +
org3 + every tenant in `.Values.tenants`) for every peer it renders,
regardless of which org list called it; `fabric-network.orderer` mounts that
same union plus OrdererMSP's own TLS CA on top (needed for orderer-to-orderer
Raft dialing, which reuses the General TLS cert as a client credential — the
compose file's own "RAFT CLUSTER TLS" note). This is computed directly
inside each define via `range $root.Values.tenants`, not passed in by the
caller, so the set can never drift out of sync between org1/org3/tenant
call sites the way network-docker-compose.yaml's own hand-maintained,
per-node mount lists have (that file's peer0.org1/peer1.org1 mount only
{org1, tenant01, org3} — tenant02 is missing, added after that file was last
touched; see values.yaml's own note on this).
=============================================================================
*/}}

{{- define "fabric-network.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "fabric-network.fullname" -}}
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

{{/* Chart-wide labels. Argument: root context. */}}
{{- define "fabric-network.labels" -}}
helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | quote }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: fabric-network
{{- end -}}

{{/*
Tenant MSP ID — the ONE place ADR-0013's "OrgClient-<tenantID>MSP" (hyphen,
2026-08-06 empirical correction) string is assembled, so every caller stays
correct if that rule is ever revisited again. Argument: the tenant id string.
*/}}
{{- define "fabric-network.tenantMspID" -}}
{{- printf "OrgClient-%sMSP" . -}}
{{- end -}}

{{/*
Resolve the effective cross-org TLS CA bundle ConfigMap name. Argument: root
context. Empty override in values.yaml falls back to "<fullname>-cross-tlsca".
*/}}
{{- define "fabric-network.crossTLSCAConfigMapName" -}}
{{- if .Values.crossTLSCAConfigMapName -}}
{{- .Values.crossTLSCAConfigMapName -}}
{{- else -}}
{{- printf "%s-cross-tlsca" (include "fabric-network.fullname" .) -}}
{{- end -}}
{{- end -}}

{{/*
A secret-or-CSI volume source, keyed off .Values.secrets.mode. Argument: dict
{root, secretName, items} where `items` is the list of {key, path} pairs the
caller wants projected into the mounted directory.
*/}}
{{- define "fabric-network.credentialVolumeSource" -}}
{{- if eq .root.Values.secrets.mode "csiExternalSecret" }}
csi:
  driver: secrets-store.csi.k8s.io
  readOnly: true
  volumeAttributes:
    secretProviderClass: {{ .secretName | quote }}
{{- else }}
secret:
  secretName: {{ .secretName | quote }}
  items:
    {{- range .items }}
    - key: {{ .key | quote }}
      path: {{ .path | quote }}
    {{- end }}
{{- end }}
{{- end -}}

{{/*
=============================================================================
fabric-network.peer — ConfigMap + headless Service + StatefulSet for ONE
peer node. Argument: dict {root, node, mspID}.
  node fields used: name, hostname, gossipBootstrap, useLeaderElection,
                    orgLeader, mspSecretName, tlsSecretName
=============================================================================
*/}}
{{- define "fabric-network.peer" -}}
{{- $root := .root -}}
{{- $node := .node -}}
{{- $fullname := include "fabric-network.fullname" $root -}}
{{- $storageClass := $root.Values.persistence.storageClassName -}}
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ $node.name }}-env
  labels:
    {{- include "fabric-network.labels" $root | nindent 4 }}
    app.kubernetes.io/name: {{ $node.name }}
    app.kubernetes.io/component: peer
data:
  # Direct translation of network-docker-compose.yaml's per-peer
  # `environment:` block. Every key/value below is copied from that file
  # (peer0.org1's block, the shape shared by every peer service in it) with
  # only the per-node identity fields substituted from values.yaml. All
  # values are explicitly quoted: a ConfigMap's `data` map is
  # map[string]string at the Kubernetes API level, and an unquoted YAML
  # bool/number here (e.g. `true` for USELEADERELECTION) renders as a JSON
  # non-string and fails API validation — quoting every value sidesteps
  # that class of bug uniformly rather than tracking it field-by-field.
  FABRIC_CFG_PATH: "/etc/hyperledger/fabric"
  FABRIC_LOGGING_SPEC: {{ $root.Values.logging.spec | quote }}
  CORE_PEER_ID: {{ $node.hostname | quote }}
  CORE_PEER_NETWORKID: {{ $root.Values.networkId | quote }}
  CORE_PEER_ADDRESS: {{ printf "%s:7051" $node.hostname | quote }}
  CORE_PEER_LISTENADDRESS: "0.0.0.0:7051"
  CORE_PEER_CHAINCODEADDRESS: {{ printf "%s:7052" $node.hostname | quote }}
  CORE_PEER_CHAINCODELISTENADDRESS: "0.0.0.0:7052"
  CORE_PEER_LOCALMSPID: {{ .mspID | quote }}
  CORE_PEER_MSPCONFIGPATH: "/etc/hyperledger/fabric/msp"
  CORE_PEER_GOSSIP_BOOTSTRAP: {{ $node.gossipBootstrap | quote }}
  CORE_PEER_GOSSIP_EXTERNALENDPOINT: {{ printf "%s:7051" $node.hostname | quote }}
  CORE_PEER_GOSSIP_USELEADERELECTION: {{ $node.useLeaderElection | quote }}
  CORE_PEER_GOSSIP_ORGLEADER: {{ $node.orgLeader | quote }}
  CORE_PEER_TLS_ENABLED: {{ $root.Values.mtls.enabled | quote }}
  CORE_PEER_TLS_CERT_FILE: "/etc/hyperledger/fabric/tls/server.crt"
  CORE_PEER_TLS_KEY_FILE: "/etc/hyperledger/fabric/tls/server.key"
  CORE_PEER_TLS_ROOTCERT_FILE: "/etc/hyperledger/fabric/tls/ca.crt"
  CORE_PEER_TLS_CLIENTAUTHREQUIRED: {{ $root.Values.mtls.enabled | quote }}
  CORE_LEDGER_STATE_STATEDATABASE: "goleveldb" # ADR-0007
  CORE_LEDGER_HISTORY_ENABLEHISTORYDATABASE: "true" # data-model.md §5
  CORE_PEER_FILESYSTEMPATH: "/var/hyperledger/production"
  CORE_OPERATIONS_LISTENADDRESS: "0.0.0.0:9443"
  CORE_METRICS_PROVIDER: {{ $root.Values.metrics.provider | quote }}
---
apiVersion: v1
kind: Service
metadata:
  name: {{ $node.name }}
  labels:
    {{- include "fabric-network.labels" $root | nindent 4 }}
    app.kubernetes.io/name: {{ $node.name }}
    app.kubernetes.io/component: peer
spec:
  # Headless (clusterIP: None) — still type ClusterIP (the Kubernetes
  # default; "headless" only removes the virtual/load-balanced IP), chosen
  # because every peer StatefulSet here has exactly 1 replica: a single,
  # individually-addressed identity, never a horizontally-scaled pool a
  # normal load-balancing ClusterIP would be appropriate for. Gossip's
  # externalEndpoint / bootstrap addressing needs a stable, specific-pod DNS
  # name — a load-balanced ClusterIP could route to "a peer" but not
  # necessarily "this peer." This is also the governing Service a
  # StatefulSet's `serviceName` requires for stable network identity.
  clusterIP: None
  selector:
    app.kubernetes.io/name: {{ $node.name }}
  ports:
    - name: grpc
      port: 7051
      targetPort: 7051
    - name: operations
      port: 9443
      targetPort: 9443
    # Note: 7052 (chaincode-listen) is deliberately NOT exposed as a Service
    # port here — same judgment network-docker-compose.yaml's own "NOT IN
    # SCOPE HERE" section makes explicit for its host-port publication: no
    # chaincode-as-a-service workload exists yet in this build phase
    # (CC-5's scope), so exposing it now would be an unused surface.
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{ $node.name }}
  labels:
    {{- include "fabric-network.labels" $root | nindent 4 }}
    app.kubernetes.io/name: {{ $node.name }}
    app.kubernetes.io/component: peer
spec:
  serviceName: {{ $node.name }}
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: {{ $node.name }}
  template:
    metadata:
      labels:
        app.kubernetes.io/name: {{ $node.name }}
        app.kubernetes.io/component: peer
    spec:
      automountServiceAccountToken: false
      containers:
        - name: peer
          image: "{{ $root.Values.images.peer.repository }}:{{ $root.Values.images.peer.tag }}"
          imagePullPolicy: {{ $root.Values.images.peer.pullPolicy }}
          command: ["peer"]
          args: ["node", "start"]
          envFrom:
            - configMapRef:
                name: {{ $node.name }}-env
          ports:
            - name: grpc
              containerPort: 7051
            - name: chaincode
              containerPort: 7052
            - name: operations
              containerPort: 9443
          readinessProbe:
            # httpGet probes are executed by the kubelet from OUTSIDE the
            # container, over the pod's network namespace — unlike a Docker
            # Compose `healthcheck:`, which needs an HTTP client binary
            # INSIDE the container. network-docker-compose.yaml's own "NOT
            # IN SCOPE HERE" note explains it added no healthcheck for
            # exactly that missing-client reason; that limitation does not
            # apply here, so a probe against the operations endpoint is a
            # safe, K8s-native addition, not a re-litigation of that note.
            httpGet:
              path: /healthz
              port: 9443
            initialDelaySeconds: 15
            periodSeconds: 15
          livenessProbe:
            httpGet:
              path: /healthz
              port: 9443
            initialDelaySeconds: 30
            periodSeconds: 20
          resources:
            {{- toYaml $root.Values.resources.peer | nindent 12 }}
          volumeMounts:
            - name: msp
              mountPath: /etc/hyperledger/fabric/msp
              readOnly: true
            - name: tls
              mountPath: /etc/hyperledger/fabric/tls
              readOnly: true
            - name: tls-cross
              mountPath: /etc/hyperledger/fabric/tls-cross
              readOnly: true
            - name: ledger
              mountPath: /var/hyperledger/production
      volumes:
        - name: msp
          {{- include "fabric-network.credentialVolumeSource" (dict "root" $root "secretName" $node.mspSecretName "items" (list
              (dict "key" "signcert" "path" "signcerts/cert.pem")
              (dict "key" "keystorekey" "path" "keystore/key.pem")
              (dict "key" "cacert" "path" "cacerts/ca-cert.pem")
              (dict "key" "tlscacert" "path" "tlscacerts/tlsca-cert.pem")
              (dict "key" "admincert" "path" "admincerts/admin-cert.pem")
              (dict "key" "configyaml" "path" "config.yaml")
            )) | nindent 10 }}
        - name: tls
          {{- include "fabric-network.credentialVolumeSource" (dict "root" $root "secretName" $node.tlsSecretName "items" (list
              (dict "key" "tlscert" "path" "server.crt")
              (dict "key" "tlskey" "path" "server.key")
              (dict "key" "tlscacert" "path" "ca.crt")
            )) | nindent 10 }}
        - name: tls-cross
          configMap:
            name: {{ include "fabric-network.crossTLSCAConfigMapName" $root }}
            items:
              - key: org1
                path: tlsca.org1-cert.pem
              - key: org3
                path: tlsca.org3-cert.pem
              {{- range $root.Values.tenants }}
              - key: {{ .id }}
                path: {{ printf "tlsca.%s-cert.pem" .id | quote }}
              {{- end }}
  volumeClaimTemplates:
    - metadata:
        name: ledger
      spec:
        accessModes: {{ $root.Values.persistence.accessModes | toYaml | nindent 10 }}
        {{- if $storageClass }}
        storageClassName: {{ $storageClass | quote }}
        {{- end }}
        resources:
          requests:
            storage: {{ $root.Values.persistence.peerLedgerSize | quote }}
{{- end -}}

{{/*
=============================================================================
fabric-network.orderer — ConfigMap + headless Service + StatefulSet for ONE
Raft consenter. Argument: dict {root, node}.
  node fields used: name, hostname, mspSecretName, tlsSecretName
=============================================================================
*/}}
{{- define "fabric-network.orderer" -}}
{{- $root := .root -}}
{{- $node := .node -}}
{{- $storageClass := $root.Values.persistence.storageClassName -}}
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ $node.name }}-env
  labels:
    {{- include "fabric-network.labels" $root | nindent 4 }}
    app.kubernetes.io/name: {{ $node.name }}
    app.kubernetes.io/component: orderer
data:
  # Direct translation of network-docker-compose.yaml's per-orderer
  # `environment:` block (identical across all 3 Org1 orderers except
  # hostname/identity, matching the source file exactly).
  FABRIC_CFG_PATH: "/etc/hyperledger/fabric"
  FABRIC_LOGGING_SPEC: {{ $root.Values.logging.spec | quote }}
  ORDERER_GENERAL_LISTENADDRESS: "0.0.0.0"
  ORDERER_GENERAL_LISTENPORT: "7050"
  ORDERER_GENERAL_LOCALMSPID: "OrdererMSP"
  ORDERER_GENERAL_LOCALMSPDIR: "/var/hyperledger/orderer/msp"
  ORDERER_GENERAL_BOOTSTRAPMETHOD: "none"
  ORDERER_GENERAL_TLS_ENABLED: {{ $root.Values.mtls.enabled | quote }}
  ORDERER_GENERAL_TLS_PRIVATEKEY: "/var/hyperledger/orderer/tls/server.key"
  ORDERER_GENERAL_TLS_CERTIFICATE: "/var/hyperledger/orderer/tls/server.crt"
  ORDERER_GENERAL_TLS_ROOTCAS: "/var/hyperledger/orderer/tls/ca.crt"
  ORDERER_GENERAL_TLS_CLIENTAUTHREQUIRED: {{ $root.Values.mtls.enabled | quote }}
  ORDERER_CHANNELPARTICIPATION_ENABLED: "true"
  ORDERER_FILELEDGER_LOCATION: "/var/hyperledger/production/orderer"
  ORDERER_CONSENSUS_WALDIR: "/var/hyperledger/production/orderer/etcdraft/wal"
  ORDERER_CONSENSUS_SNAPDIR: "/var/hyperledger/production/orderer/etcdraft/snapshot"
  ORDERER_ADMIN_TLS_ENABLED: {{ $root.Values.mtls.enabled | quote }}
  ORDERER_ADMIN_TLS_CERTIFICATE: "/var/hyperledger/orderer/tls/server.crt"
  ORDERER_ADMIN_TLS_PRIVATEKEY: "/var/hyperledger/orderer/tls/server.key"
  ORDERER_ADMIN_TLS_CLIENTAUTHREQUIRED: {{ $root.Values.mtls.enabled | quote }}
  ORDERER_ADMIN_TLS_CLIENTROOTCAS: "/var/hyperledger/orderer/tls/ca.crt"
  ORDERER_ADMIN_LISTENADDRESS: "0.0.0.0:9443"
  ORDERER_OPERATIONS_LISTENADDRESS: "0.0.0.0:8443"
  ORDERER_METRICS_PROVIDER: {{ $root.Values.metrics.provider | quote }}
---
apiVersion: v1
kind: Service
metadata:
  name: {{ $node.name }}
  labels:
    {{- include "fabric-network.labels" $root | nindent 4 }}
    app.kubernetes.io/name: {{ $node.name }}
    app.kubernetes.io/component: orderer
spec:
  clusterIP: None # see fabric-network.peer's identical note — 1 replica, needs a stable specific-pod identity, not load-balancing.
  selector:
    app.kubernetes.io/name: {{ $node.name }}
  ports:
    - name: grpc
      port: 7050
      targetPort: 7050
    - name: admin
      port: 9443
      targetPort: 9443
    - name: operations
      port: 8443
      targetPort: 8443
---
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: {{ $node.name }}
  labels:
    {{- include "fabric-network.labels" $root | nindent 4 }}
    app.kubernetes.io/name: {{ $node.name }}
    app.kubernetes.io/component: orderer
spec:
  serviceName: {{ $node.name }}
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: {{ $node.name }}
  template:
    metadata:
      labels:
        app.kubernetes.io/name: {{ $node.name }}
        app.kubernetes.io/component: orderer
    spec:
      automountServiceAccountToken: false
      containers:
        - name: orderer
          image: "{{ $root.Values.images.orderer.repository }}:{{ $root.Values.images.orderer.tag }}"
          imagePullPolicy: {{ $root.Values.images.orderer.pullPolicy }}
          command: ["orderer"]
          envFrom:
            - configMapRef:
                name: {{ $node.name }}-env
          ports:
            - name: grpc
              containerPort: 7050
            - name: admin
              containerPort: 9443
            - name: operations
              containerPort: 8443
          readinessProbe:
            httpGet:
              path: /healthz
              port: 8443
            initialDelaySeconds: 15
            periodSeconds: 15
          livenessProbe:
            httpGet:
              path: /healthz
              port: 8443
            initialDelaySeconds: 30
            periodSeconds: 20
          resources:
            {{- toYaml $root.Values.resources.orderer | nindent 12 }}
          volumeMounts:
            - name: msp
              mountPath: /var/hyperledger/orderer/msp
              readOnly: true
            - name: tls
              mountPath: /var/hyperledger/orderer/tls
              readOnly: true
            - name: tls-cross
              mountPath: /var/hyperledger/orderer/tls-cross
              readOnly: true
            - name: ledger
              mountPath: /var/hyperledger/production/orderer
      volumes:
        - name: msp
          {{- include "fabric-network.credentialVolumeSource" (dict "root" $root "secretName" $node.mspSecretName "items" (list
              (dict "key" "signcert" "path" "signcerts/cert.pem")
              (dict "key" "keystorekey" "path" "keystore/key.pem")
              (dict "key" "cacert" "path" "cacerts/ca-cert.pem")
              (dict "key" "tlscacert" "path" "tlscacerts/tlsca-cert.pem")
              (dict "key" "admincert" "path" "admincerts/admin-cert.pem")
              (dict "key" "configyaml" "path" "config.yaml")
            )) | nindent 10 }}
        - name: tls
          {{- include "fabric-network.credentialVolumeSource" (dict "root" $root "secretName" $node.tlsSecretName "items" (list
              (dict "key" "tlscert" "path" "server.crt")
              (dict "key" "tlskey" "path" "server.key")
              (dict "key" "tlscacert" "path" "ca.crt")
            )) | nindent 10 }}
        - name: tls-cross
          configMap:
            name: {{ include "fabric-network.crossTLSCAConfigMapName" $root }}
            items:
              - key: ordererorg
                path: tlsca.ordererorg-cert.pem
              - key: org1
                path: tlsca.org1-cert.pem
              - key: org3
                path: tlsca.org3-cert.pem
              {{- range $root.Values.tenants }}
              - key: {{ .id }}
                path: {{ printf "tlsca.%s-cert.pem" .id | quote }}
              {{- end }}
  volumeClaimTemplates:
    - metadata:
        name: ledger
      spec:
        accessModes: {{ $root.Values.persistence.accessModes | toYaml | nindent 10 }}
        {{- if $storageClass }}
        storageClassName: {{ $storageClass | quote }}
        {{- end }}
        resources:
          requests:
            storage: {{ $root.Values.persistence.ordererLedgerSize | quote }}
{{- end -}}
