package gqlwss

import (
	"net/http"
)

func New(opts Options) Server {
	mux := http.NewServeMux()
	mux.Handle(opts.AuthPath, Auth{})
	mux.Handle(opts.GraphQLPath, GraphQL{opts.Schema})
	return Server{mux}
}
