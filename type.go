package gqlwss

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
)

var (
	authKey           = contextKey("auth key")
	inputsKey         = contextKey("inputs")
	outputsKey        = contextKey("outputs")
	errorsKey         = contextKey("errors")
	schemaKey         = contextKey("schema")
	socketKey         = contextKey("socket")
	messageKey        = contextKey("message")
	currentStateKey   = contextKey("current state")
	executeResultsKey = contextKey("graphql execute results")
	executeParamsKey  = contextKey("graphql execute params")
)

type contextKey string

type opId int64

type opMap map[opId]op

type action func(context.Context)

type testUser struct {
	Id    int64
	Name  string
	Email string
}

type op struct {
	ctx    context.Context
	cancel context.CancelFunc
}

type state struct {
	isInitialised bool
	ops           opMap
}

type message struct {
	Id      opId   `json:"id"`
	Type    string `json:"type"`
	Payload string `json:"payload,omitempty"`
}

type Options struct {
	Schema      graphql.Schema
	AuthPath    string
	GraphQLPath string
}

type Server struct {
	mux *http.ServeMux
}

type GraphQLSocket struct {
	schema   graphql.Schema
	upgrader websocket.Upgrader
}

type Auth struct {
	// ...
}
