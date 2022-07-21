# Sidecar helper for service-meshes

## Motivation

A service mesh is a dedicated infrastructure layer that you can add to your applications. This additional layer is based on adding a proxy "sidecar" along with every application deployed.

Sometime it's hard to handle this "sidecar" with job or daemons:

### Problem #1

Jobs or daemons need that "proxy" sidecar was ready before executing application

### Problem #2

After executing job (success or failure), "proxy" sidecar must be stoped

## How it works

`envoy-sidecar-helper` is additional sidecar container that will monitor Termination of main application container (with Kubernetes API), and will shutdown envoy "proxy" sidecar. Also it can share via `emptyDir` volume information about ready envoy container

```yaml
...
serviceAccount: envoy-sidecar-helper
volumes:
- name: envoy-sidecar-helper
  emptyDir: {}
containers:
- name: main
  image: alpine:latest
  imagePullPolicy: Always
  command:
  - sh
  - -c
  - |
    set -ex

    while [ ! -f /envoy-sidecar-helper/envoy.ready ]; do sleep 1s; done

    # start your application
    echo envoy ready
  volumeMounts:
  - mountPath: /envoy-sidecar-helper
    name: envoy-sidecar-helper
- name: envoy
  image: envoyproxy/envoy-dev
  imagePullPolicy: Always
###########################################
# envoy helper
###########################################
- name: envoy-sidecar-helper
  image: paskalmaksim/envoy-sidecar-helper:latest
  imagePullPolicy: Always
  args:
  - -envoy.ready.check=true
  - -envoy.endpoint.ready=/ready
  - -envoy.port=9901
  env:
  - name: POD_NAME
    valueFrom:
      fieldRef:
        fieldPath: metadata.name
  - name: POD_NAMESPACE
    valueFrom:
      fieldRef:
        fieldPath: metadata.namespace
  volumeMounts:
  - mountPath: /envoy-sidecar-helper
    name: envoy-sidecar-helper
...
```

`envoy-sidecar-helper` need service account with permissions to get pod information

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: envoy-sidecar-helper
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: envoy-sidecar-helper-role
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: envoy-sidecar-helper
roleRef:
  kind: Role
  name: envoy-sidecar-helper-role
  apiGroup: rbac.authorization.k8s.io
subjects:
- kind: ServiceAccount
  name: envoy-sidecar-helper
  namespace: default
```
