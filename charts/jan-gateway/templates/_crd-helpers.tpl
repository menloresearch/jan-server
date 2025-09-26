{{/*
Helper template to check if required CRDs exist.
This will be used in other templates to conditionally render resources.
*/}}
{{- define "jan-gateway.crdsExist" -}}
{{- $crdsReady := true }}

{{/* Check GPU Operator CRDs if enabled */}}
{{- if and .Values.inference.enabled .Values.inference.dependencies.gpuOperator.enabled }}
{{- if not (lookup "apiextensions.k8s.io/v1" "CustomResourceDefinition" "" "clusterpolicies.nvidia.com") }}
{{- $crdsReady = false }}
{{- end }}
{{- end }}

{{/* Check KubeRay CRDs if enabled */}}
{{- if and .Values.inference.enabled .Values.inference.dependencies.kuberayOperator.enabled }}
{{- if not (lookup "apiextensions.k8s.io/v1" "CustomResourceDefinition" "" "rayclusters.ray.io") }}
{{- $crdsReady = false }}
{{- end }}
{{- end }}

{{/* Check Envoy Gateway CRDs if enabled */}}
{{- if and .Values.inference.enabled .Values.inference.dependencies.envoyGateway.enabled }}
{{- if not (lookup "apiextensions.k8s.io/v1" "CustomResourceDefinition" "" "gatewayclasses.gateway.networking.k8s.io") }}
{{- $crdsReady = false }}
{{- end }}
{{- end }}

{{/* Check Aibrix CRDs if enabled */}}
{{- if and .Values.inference.enabled .Values.inference.dependencies.aibrix.enabled }}
{{- if not (lookup "apiextensions.k8s.io/v1" "CustomResourceDefinition" "" "podautoscalers.autoscaling.aibrix.ai") }}
{{- $crdsReady = false }}
{{- end }}
{{- end }}

{{- $crdsReady }}
{{- end }}

{{/*
Helper template to generate CRD validation warning
*/}}
{{- define "jan-gateway.crdValidationWarning" -}}
{{- $missingCrds := list }}

{{- if and .Values.inference.enabled .Values.inference.dependencies.gpuOperator.enabled }}
{{- if not (lookup "apiextensions.k8s.io/v1" "CustomResourceDefinition" "" "clusterpolicies.nvidia.com") }}
{{- $missingCrds = append $missingCrds "clusterpolicies.nvidia.com (GPU Operator)" }}
{{- end }}
{{- end }}

{{- if and .Values.inference.enabled .Values.inference.dependencies.kuberayOperator.enabled }}
{{- if not (lookup "apiextensions.k8s.io/v1" "CustomResourceDefinition" "" "rayclusters.ray.io") }}
{{- $missingCrds = append $missingCrds "rayclusters.ray.io (KubeRay)" }}
{{- end }}
{{- end }}

{{- if and .Values.inference.enabled .Values.inference.dependencies.envoyGateway.enabled }}
{{- if not (lookup "apiextensions.k8s.io/v1" "CustomResourceDefinition" "" "gatewayclasses.gateway.networking.k8s.io") }}
{{- $missingCrds = append $missingCrds "gatewayclasses.gateway.networking.k8s.io (Envoy Gateway)" }}
{{- end }}
{{- end }}

{{- if and .Values.inference.enabled .Values.inference.dependencies.aibrix.enabled }}
{{- if not (lookup "apiextensions.k8s.io/v1" "CustomResourceDefinition" "" "podautoscalers.autoscaling.aibrix.ai") }}
{{- $missingCrds = append $missingCrds "podautoscalers.autoscaling.aibrix.ai (Aibrix)" }}
{{- end }}
{{- end }}

{{- if $missingCrds }}
WARNING: The following CRDs are missing and will be installed by pre-install hook:
{{- range $missingCrds }}
  - {{ . }}
{{- end }}

If the hook fails, you can install dependencies manually:
  helm upgrade --install gpu-operator nvidia/gpu-operator --namespace gpu-operator-resources --create-namespace
  helm upgrade --install kuberay-operator kuberay/kuberay-operator --namespace kuberay-system --create-namespace
  helm upgrade --install envoy-gateway oci://docker.io/envoyproxy/gateway-helm --namespace envoy-gateway-system --create-namespace
  helm upgrade --install aibrix aibrix/aibrix --namespace aibrix-system --create-namespace
{{- end }}
{{- end }}