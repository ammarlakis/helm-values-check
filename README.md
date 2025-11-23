# helm-values-check

A small Go CLI that checks mismatches between values defined in a Helm chart's `values.yaml` and the values referenced in its templates.

## Features

- Detects values **defined in `values.yaml` but never used** in templates.
- Detects values **used in templates but not defined** in `values.yaml`.
- Understands both `{{ .Values.foo.bar }}` and `{{ index .Values "foo" "bar" }}` forms.
- Optionally includes subchart templates from the `charts/` directory.

## Usage

From the project root:

```sh
just tidy   # optional, to resolve Go module dependencies
just build  # builds the binary

# Run against a chart directory
just run path/to/chart
```

Or directly with Go:

```sh
go run ./cmd/helm-values-check path/to/chart
```

Exit code is non-zero when mismatches are found, which makes this suitable for CI checks.
