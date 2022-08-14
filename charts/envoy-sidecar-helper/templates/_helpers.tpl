{{- define "envoy-sidecar-helper.image" -}}
image: {{ printf "%s/%s:%s" (default .image.registry .global.imageRegistry) .image.repository .image.tag }}
imagePullPolicy: {{ .image.pullPolicy }}
{{- end -}}