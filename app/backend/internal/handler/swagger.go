package handler

import (
	"html/template"
	"net/http"
	"path"

	backendapi "github.com/guitaramust-sudo/Avitosha/app/backend/api"
	swaggerFiles "github.com/swaggo/files"
)

const openAPIContentType = "application/yaml; charset=utf-8"

var swaggerUITemplate = template.Must(template.New("swagger-ui").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Avitosha Auth API</title>
  <link rel="stylesheet" href="{{ .BasePath }}/swagger-ui.css">
  <link rel="stylesheet" href="{{ .BasePath }}/index.css">
  <style>
    body {
      margin: 0;
      background: #faf7f2;
    }
    .topbar {
      display: none;
    }
  </style>
</head>
<body>
  <div id="swagger-ui"></div>
  <script src="{{ .BasePath }}/swagger-ui-bundle.js"></script>
  <script src="{{ .BasePath }}/swagger-ui-standalone-preset.js"></script>
  <script src="{{ .BasePath }}/swagger-initializer.js"></script>
</body>
</html>
`))

var swaggerInitializerTemplate = template.Must(template.New("swagger-initializer").Parse(`window.onload = function () {
  window.ui = SwaggerUIBundle({
    url: "{{ .SpecPath }}",
    dom_id: "#swagger-ui",
    deepLinking: true,
    displayRequestDuration: true,
    docExpansion: "list",
    defaultModelsExpandDepth: 1,
    presets: [
      SwaggerUIBundle.presets.apis,
      SwaggerUIStandalonePreset
    ],
    layout: "StandaloneLayout"
  });
};
`))

type SwaggerUIHandler struct {
	specPath string
	basePath string
	assets   http.Handler
}

func NewSwaggerUIHandler(specPath string) SwaggerUIHandler {
	return SwaggerUIHandler{
		specPath: specPath,
		basePath: "/swagger",
		assets:   http.FileServer(swaggerFiles.HTTP),
	}
}

func (h SwaggerUIHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	switch path.Clean(req.URL.Path) {
	case "/", "/index.html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := swaggerUITemplate.Execute(w, struct {
			BasePath string
		}{
			BasePath: h.basePath,
		}); err != nil {
			http.Error(w, "failed to render swagger ui", http.StatusInternalServerError)
		}
	case "/swagger-initializer.js":
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
		if err := swaggerInitializerTemplate.Execute(w, struct {
			SpecPath string
		}{
			SpecPath: h.specPath,
		}); err != nil {
			http.Error(w, "failed to render swagger initializer", http.StatusInternalServerError)
		}
	default:
		h.assets.ServeHTTP(w, req)
	}
}

func OpenAPISpec(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", openAPIContentType)
	_, _ = w.Write(backendapi.OpenAPIYAML())
}
