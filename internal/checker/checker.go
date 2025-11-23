package checker

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"helm.sh/helm/v3/pkg/chart"
)

type Config struct {
	IncludeSubcharts bool
}

type Result struct {
	DefinedNotUsed []string
	UsedNotDefined []string
}

var (
	dotAccessRe = regexp.MustCompile(`(?:^|[^A-Za-z0-9_])(?:\.|\$\.)Values\.([A-Za-z0-9_]+(?:\.[A-Za-z0-9_]+)*)`)
	indexAccessRe = regexp.MustCompile(`index\s+\.Values\s+("([^"]+)"(?:\s+"([^"]+)")*)`)
	rootValuesRe = regexp.MustCompile(`(?:^|[^A-Za-z0-9_])(?:\.|\$\.)Values([^A-Za-z0-9_\.]|$)`)
)

func CheckChart(ch *chart.Chart, cfg Config) (Result, error) {
	if ch == nil {
		return Result{}, fmt.Errorf("chart is nil")
	}

	defined := collectDefinedValues(ch)
	used, usesRootValues := collectUsedValues(ch, cfg.IncludeSubcharts)

	result := Result{}

	if usesRootValues {
		result.DefinedNotUsed = nil
	} else {
		result.DefinedNotUsed = definedButNotUsed(defined, used)
	}

	result.UsedNotDefined = usedButNotDefined(used, defined)

	return result, nil
}

func collectDefinedValues(ch *chart.Chart) map[string]struct{} {
	defined := make(map[string]struct{})
	flattenValues("", ch.Values, defined)
	return defined
}

func flattenValues(prefix string, v interface{}, out map[string]struct{}) {
	switch val := v.(type) {
	case map[string]interface{}:
		if prefix != "" {
			out[prefix] = struct{}{}
		}
		for k, child := range val {
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			flattenValues(key, child, out)
		}
	case map[interface{}]interface{}:
		if prefix != "" {
			out[prefix] = struct{}{}
		}
		for rawK, child := range val {
			k, ok := rawK.(string)
			if !ok {
				continue
			}
			key := k
			if prefix != "" {
				key = prefix + "." + k
			}
			flattenValues(key, child, out)
		}
	default:
		if prefix != "" {
			out[prefix] = struct{}{}
		}
	}
}

func collectUsedValues(ch *chart.Chart, includeSubcharts bool) (map[string]struct{}, bool) {
	used := make(map[string]struct{})
	usesRootValues := false

	for _, tmpl := range ch.Templates {
		if !includeSubcharts && isSubchartTemplate(tmpl.Name) {
			continue
		}

		content := string(tmpl.Data)

		if rootValuesRe.MatchString(content) {
			usesRootValues = true
		}

		for _, m := range dotAccessRe.FindAllStringSubmatch(content, -1) {
			path := m[1]
			if path == "" {
				continue
			}
			used[path] = struct{}{}
		}

		for _, m := range indexAccessRe.FindAllStringSubmatch(content, -1) {
			full := strings.TrimSpace(m[1])
			if full == "" {
				continue
			}
			parts := strings.Fields(full)
			var keys []string
			for _, p := range parts {
				p = strings.Trim(p, "\"")
				if p != "" {
					keys = append(keys, p)
				}
			}
			if len(keys) > 0 {
				used[strings.Join(keys, ".")] = struct{}{}
			}
		}
	}

	return used, usesRootValues
}

func isSubchartTemplate(name string) bool {
	clean := filepath.ToSlash(name)
	return strings.Contains(clean, "/charts/")
}

func definedButNotUsed(defined, used map[string]struct{}) []string {
	if len(defined) == 0 {
		return nil
	}

	definedKeys := make([]string, 0, len(defined))
	for k := range defined {
		definedKeys = append(definedKeys, k)
	}

	usedKeys := make([]string, 0, len(used))
	for k := range used {
		usedKeys = append(usedKeys, k)
	}

	var out []string
	for _, d := range definedKeys {
		// Skip container keys that have defined children (e.g. "zigbee2mqtt", "zigbee2mqtt.ingress").
		isContainer := false
		for _, other := range definedKeys {
			if other != d && strings.HasPrefix(other, d+".") {
				isContainer = true
				break
			}
		}
		if isContainer {
			continue
		}

		// If this exact key is used, it's not "defined but not used".
		if _, ok := used[d]; ok {
			continue
		}

		// If a parent of this key is used (e.g. used "a.b" while d is "a.b.c"),
		// treat this key as effectively used as well.
		skip := false
		for _, u := range usedKeys {
			if strings.HasPrefix(d, u+".") {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		out = append(out, d)
	}

	sort.Strings(out)
	return out
}

func usedButNotDefined(used, defined map[string]struct{}) []string {
	if len(used) == 0 {
		return nil
	}

	definedKeys := make([]string, 0, len(defined))
	for k := range defined {
		definedKeys = append(definedKeys, k)
	}

	var out []string

UsedLoop:
	for u := range used {
		if _, ok := defined[u]; ok {
			continue
		}
		for _, d := range definedKeys {
			if strings.HasPrefix(d, u+".") {
				continue UsedLoop
			}
		}
		out = append(out, u)
	}

	sort.Strings(out)
	return out
}
