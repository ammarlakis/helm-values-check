package checker

import (
	"testing"

	"helm.sh/helm/v3/pkg/chart"
)

func TestCheckChart_NoMismatches(t *testing.T) {
	ch := &chart.Chart{
		Values: map[string]interface{}{
			"replicaCount": 1,
		},
		Templates: []*chart.File{
			{
				Name: "templates/configmap.yaml",
				Data: []byte(`{{ .Values.replicaCount }}`),
			},
		},
	}

	res, err := CheckChart(ch, Config{})
	if err != nil {
		t.Fatalf("CheckChart returned error: %v", err)
	}

	if len(res.DefinedNotUsed) != 0 {
		t.Fatalf("expected no defined-not-used values, got %v", res.DefinedNotUsed)
	}
	if len(res.UsedNotDefined) != 0 {
		t.Fatalf("expected no used-not-defined values, got %v", res.UsedNotDefined)
	}
}

func TestCheckChart_DefinedNotUsed(t *testing.T) {
	ch := &chart.Chart{
		Values: map[string]interface{}{
			"foo": "bar",
		},
		Templates: []*chart.File{
			{
				Name: "templates/empty.yaml",
				Data: []byte(`# no values used`),
			},
		},
	}

	res, err := CheckChart(ch, Config{})
	if err != nil {
		t.Fatalf("CheckChart returned error: %v", err)
	}

	if len(res.DefinedNotUsed) != 1 || res.DefinedNotUsed[0] != "foo" {
		t.Fatalf("expected defined-not-used [foo], got %v", res.DefinedNotUsed)
	}
}

func TestCheckChart_UsedNotDefined(t *testing.T) {
	ch := &chart.Chart{
		Values: map[string]interface{}{},
		Templates: []*chart.File{
			{
				Name: "templates/deploy.yaml",
				Data: []byte(`{{ .Values.image.tag }}`),
			},
		},
	}

	res, err := CheckChart(ch, Config{})
	if err != nil {
		t.Fatalf("CheckChart returned error: %v", err)
	}

	if len(res.UsedNotDefined) != 1 || res.UsedNotDefined[0] != "image.tag" {
		t.Fatalf("expected used-not-defined [image.tag], got %v", res.UsedNotDefined)
	}
}

func TestCheckChart_IndexAccess(t *testing.T) {
	ch := &chart.Chart{
		Values: map[string]interface{}{
			"image": map[string]interface{}{
				"tag": "latest",
			},
		},
		Templates: []*chart.File{
			{
				Name: "templates/deploy.yaml",
				Data: []byte(`{{ index .Values "image" "tag" }}`),
			},
		},
	}

	res, err := CheckChart(ch, Config{})
	if err != nil {
		t.Fatalf("CheckChart returned error: %v", err)
	}

	if len(res.DefinedNotUsed) != 0 {
		t.Fatalf("expected no defined-not-used values, got %v", res.DefinedNotUsed)
	}
	if len(res.UsedNotDefined) != 0 {
		t.Fatalf("expected no used-not-defined values, got %v", res.UsedNotDefined)
	}
}

func TestCheckChart_RootValuesUsageSkipsDefinedNotUsed(t *testing.T) {
	ch := &chart.Chart{
		Values: map[string]interface{}{
			"foo": map[string]interface{}{
				"bar": "baz",
			},
		},
		Templates: []*chart.File{
			{
				Name: "templates/config.yaml",
				Data: []byte(`{{ toYaml .Values }}`),
			},
		},
	}

	res, err := CheckChart(ch, Config{})
	if err != nil {
		t.Fatalf("CheckChart returned error: %v", err)
	}

	if len(res.DefinedNotUsed) != 0 {
		t.Fatalf("expected no defined-not-used values when .Values used as a whole, got %v", res.DefinedNotUsed)
	}
}

func TestCheckChart_HelpersAndOverrides(t *testing.T) {
	ch := &chart.Chart{
		Values: map[string]interface{}{
			"nameOverride":       "",
			"fullnameOverride":   "",
		},
		Templates: []*chart.File{
			{
				Name: "templates/_helpers.tpl",
				Data: []byte(`
{{- define "zigbee2mqtt.fullname" -}}
{{- if .Values.fullnameOverride -}}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- .Chart.Name -}}
{{- end -}}
{{- end -}}

{{- define "zigbee2mqtt.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}
`),
			},
		},
	}

	res, err := CheckChart(ch, Config{})
	if err != nil {
		t.Fatalf("CheckChart returned error: %v", err)
	}

	if len(res.DefinedNotUsed) != 0 {
		t.Fatalf("expected no defined-not-used values, got %v", res.DefinedNotUsed)
	}
	if len(res.UsedNotDefined) != 0 {
		t.Fatalf("expected no used-not-defined values, got %v", res.UsedNotDefined)
	}
}

func TestCheckChart_ZigbeeNestedInitAndIngressAnnotations(t *testing.T) {
	ch := &chart.Chart{
		Values: map[string]interface{}{
			"zigbee2mqtt": map[string]interface{}{
				"initConfig": map[string]interface{}{
					"mqtt": map[string]interface{}{
						"host": "localhost",
						"port": 1883,
					},
				},
				"ingress": map[string]interface{}{
					"enabled": true,
					"annotations": map[string]interface{}{
						"kubernetes.io/ingress.class": "nginx",
					},
				},
			},
		},
		Templates: []*chart.File{
			{
				Name: "templates/configmap.yaml",
				Data: []byte(`
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "zigbee2mqtt.fullname" . }}
data:
  config.yaml: |
    {{ .Values.zigbee2mqtt.initConfig | toYaml | nindent 4 }}
`),
			},
			{
				Name: "templates/ingress.yaml",
				Data: []byte(`
{{- if .Values.zigbee2mqtt.ingress.enabled }}
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: {{ include "zigbee2mqtt.fullname" . }}
  annotations:
{{- with .Values.zigbee2mqtt.ingress.annotations }}
{{ toYaml . | indent 4 }}
{{- end }}
{{- end }}
`),
			},
		},
	}

	res, err := CheckChart(ch, Config{})
	if err != nil {
		t.Fatalf("CheckChart returned error: %v", err)
	}

	if len(res.DefinedNotUsed) != 0 {
		t.Fatalf("expected no defined-not-used values, got %v", res.DefinedNotUsed)
	}
	if len(res.UsedNotDefined) != 0 {
		t.Fatalf("expected no used-not-defined values, got %v", res.UsedNotDefined)
	}
}
