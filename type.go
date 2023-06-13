package gqlwss

import (
	"net/http"

	"github.com/graphql-go/graphql"
)

type Options struct {
	Schema      graphql.Schema
	AuthPath    string
	GraphQLPath string
}

type Server struct {
	mux *http.ServeMux
}

type GraphQL struct {
	schema graphql.Schema
}

type Auth struct {
	// ...
}
