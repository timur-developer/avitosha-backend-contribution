package api

import _ "embed"

//go:embed openapi.yaml
var openapiYAML []byte

func OpenAPIYAML() []byte {
	return append([]byte(nil), openapiYAML...)
}
