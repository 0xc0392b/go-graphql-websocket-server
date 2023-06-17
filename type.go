package gqlwss

import (
	"context"
	"net/http"

	"github.com/graphql-go/graphql"
)

var (
	authKey         = contextKey("auth key")
	inputsKey       = contextKey("inputs")
	outputsKey      = contextKey("outputs")
	errorsKey       = contextKey("errors")
	schemaKey       = contextKey("schema")
	socketKey       = contextKey("socket")
	messageKey      = contextKey("message")
	currentStateKey = contextKey("current state")
)

type contextKey string

type action func(context.Context)

type operation struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type state struct {
	isInitialised bool
	operations    map[int64]operation
}

type message struct {
	Id      int64  `json:"id"`
	Type    string `json:"type"`
	Payload string `json:"payload"`
}

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
