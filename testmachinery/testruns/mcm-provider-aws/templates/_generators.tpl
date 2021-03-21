{{- define "generator" }}
{{ if eq .Values.shoot.cloudprovider "aws" }}
  - name: generate-provider
    annotations:
      testmachinery.sapcloud.io/system-step: "true"
    definition:
      name: gen-provider-aws
      config:
      - name: CONTROLPLANE_PROVIDER_CONFIG_FILEPATH
        type: env
        value: /tmp/tm/shared/generators/controlplane.yaml
      - name: INFRASTRUCTURE_PROVIDER_CONFIG_FILEPATH
        type: env
        value: /tmp/tm/shared/generators/infra.yaml
      - name: ZONE
        type: env
        value: {{ required "a zone is required for aws shoots" .Values.shoot.zone }}
{{ end }}
{{- end }}

{{- define "config-overwrites" }}
      {{ if .Values.shoot.infrastructureConfig }}
      - name: INFRASTRUCTURE_PROVIDER_CONFIG_FILEPATH
        type: file
        path: /tmp/tm/shared/generators/infra.yaml
        value: {{ .Values.shoot.infrastructureConfig }}
      {{ end }}
      {{ if .Values.shoot.controlplaneConfig }}
      - name: CONTROLPLANE_PROVIDER_CONFIG_FILEPATH
        type: file
        path: /tmp/tm/shared/generators/controlplane.yaml
        value: {{ .Values.shoot.controlplaneConfig }}
      {{ end }}
{{- end }}