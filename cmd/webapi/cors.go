package main

import (
	"github.com/gorilla/handlers"
	"net/http"
	"strings"
)

// applyCORSHandler applies a CORS policy to the router. CORS stands for Cross-Origin Resource Sharing: it's a security
// feature present in web browsers that blocks JavaScript requests going across different domains if not specified in a
// policy. This function sends the policy of this API server.

func applyCORSHandler(h http.Handler, configuredOrigins string) http.Handler {
	origins := make([]string, 0)
	for _, origin := range strings.Split(configuredOrigins, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			origins = append(origins, origin)
		}
	}
	if len(origins) == 0 {
		origins = []string{"http://localhost:8080"}
	}
	return handlers.CORS(
		handlers.AllowedHeaders([]string{
			"Content-Type",
			"Authorization",
		}),
		handlers.AllowedMethods([]string{"GET", "POST", "OPTIONS", "DELETE", "PUT"}),
		handlers.AllowedOrigins(origins),
		handlers.AllowCredentials(),
		handlers.MaxAge(600),
	)(h)
}
