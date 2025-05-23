package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
)

var (
	version              = os.Getenv("VERSION")
	OcelotAuthCookieName = "ocelot-auth"
)

func main() {
	http.HandleFunc("/api", handleCounter)
	port := "3000"
	fmt.Printf("Server running on port %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil)) // #nosec G114 (CWE-676): Use of net/http serve function that has no support for setting timeouts; sample app is only relevant for testing
}

func handleCounter(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(OcelotAuthCookieName) != "" {
		message := fmt.Sprintf("error, the '%s' header should never be set", OcelotAuthCookieName)
		http.Error(w, message, http.StatusBadRequest)
		return
	}
	if len(r.Cookies()) > 0 {
		http.Error(w, "error, the request should not contain any cookies", http.StatusBadRequest)
		return
	}
	if len(r.URL.Query()) > 0 {
		http.Error(w, "error, the request should not contain any query parameters", http.StatusBadRequest)
		return
	}

	_, err := w.Write([]byte("this is version " + version))
	if err != nil {
		log.Printf("Error writing response: %v", err)
		return
	}
}
