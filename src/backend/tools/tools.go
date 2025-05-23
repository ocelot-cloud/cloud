package tools

import (
	"github.com/ocelot-cloud/shared/utils"
	"net/http"
	"os"
	"strings"
)

func FrontendHandlerFunc(w http.ResponseWriter, r *http.Request) {
	// Attempt to open the requested file within the ./dist directory.
	staticFileServer := http.FileServer(http.Dir("./dist"))

	// If the requested file does not exist (err is not nil) and the path does not seem to refer to
	// a static file (i.e. no dot extension like ".css"), then serve index.html. This caters to SPA routing needs,
	// allowing frontend routes to be handled by index.html.
	// This means that users can directly access pages with paths such as "example.com/some/path".
	file, err := http.Dir("./dist").Open(r.URL.Path)
	if err != nil {
		if !strings.Contains(r.URL.Path, ".") {
			Logger.Debug("Serving index.html for SPA route: %s", r.URL.Path)
			http.ServeFile(w, r, "./dist/index.html")
			return
		}

		// If the file does not exist and the path refers to a static file, return 404.
		Logger.Warn("Static file not found: %s", r.URL.Path)
		http.NotFound(w, r)
		return
	}
	defer func(file http.File) {
		err := file.Close()
		if err != nil {
			Logger.Error("Failed to close file: %v", err)
		}
	}(file)

	// If the file exists, ensure the correct Content-Type is set based on the file extension.
	switch {
	case strings.HasSuffix(r.URL.Path, ".css"):
		w.Header().Set("Content-Type", "text/css")
	case strings.HasSuffix(r.URL.Path, ".js"):
		w.Header().Set("Content-Type", "application/javascript")
	case strings.HasSuffix(r.URL.Path, ".html"):
		w.Header().Set("Content-Type", "text/html")
	case strings.HasSuffix(r.URL.Path, ".json"):
		w.Header().Set("Content-Type", "application/json")
	case strings.HasSuffix(r.URL.Path, ".png"):
		w.Header().Set("Content-Type", "image/png")
	case strings.HasSuffix(r.URL.Path, ".jpg"), strings.HasSuffix(r.URL.Path, ".jpeg"):
		w.Header().Set("Content-Type", "image/jpeg")
	case strings.HasSuffix(r.URL.Path, ".svg"):
		w.Header().Set("Content-Type", "image/svg+xml")
	case strings.HasSuffix(r.URL.Path, ".woff"):
		w.Header().Set("Content-Type", "font/woff")
	case strings.HasSuffix(r.URL.Path, ".woff2"):
		w.Header().Set("Content-Type", "font/woff2")
	default:
		// Use the default Content-Type determined by the file server.
	}

	// If the request is for a static file, serve it directly.
	// This handles requests for JS, CSS, images, etc.
	Logger.Debug("Serving static content at '%s'", r.URL.Path)
	staticFileServer.ServeHTTP(w, r)
}

func FindDirWithIdeSupport(dirName string) string {
	currentDir, err := os.Getwd()
	if err != nil {
		Logger.Fatal("Error getting current dir: %v", err)
	}
	// when running the backend via JetBrains IDE's it uses the project dir as current dir, so this case must be handled explicitly.
	if strings.HasSuffix(currentDir, "/ocelot-cloud-pe") {
		return currentDir + "/src/backend/" + dirName
	}

	return utils.FindDir(dirName)
}

func AreMocksUsed() bool {
	return os.Getenv("USE_MOCKS") == "true"
}

func TryLockAndRespondForError(w http.ResponseWriter, operation string) error {
	err := AppOperationMutex.TryLock(operation)
	if err != nil {
		message := "could not execute this operation since there is another operation in progress: " + operation
		Logger.Error(message)
		http.Error(w, message, http.StatusBadRequest)
	}
	return err
}

func FindUniqueMaintainerAndAppNamePairs(apps []MaintainerAndApp) []MaintainerAndApp {
	seen := make(map[string]struct{})
	var result []MaintainerAndApp

	for _, b := range apps {
		key := b.Maintainer + "|" + b.AppName
		if _, exists := seen[key]; !exists {
			seen[key] = struct{}{}
			result = append(result, MaintainerAndApp{
				Maintainer: b.Maintainer,
				AppName:    b.AppName,
			})
		}
	}
	return result
}

func WriteResponse(w http.ResponseWriter, message string) {
	_, err := w.Write([]byte(message))
	if err != nil {
		Logger.Error("Failed to write response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}

func CreateOcelotTempDir() {
	err := os.MkdirAll(OcelotCloudTempDir, 0700)
	if err != nil {
		Logger.Fatal("Error creating temp dir: %v", err)
	}
}
