package gqlwss

import (
	"context"
	"log"
	"net/http"
)

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s Auth) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// TODO auth logic
	// ...

}

func (s GraphQLSocket) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// each connection has a context
	ctx := r.Context()

	// do authentication
	identifier := "something"

	// do the upgrade connection upgrade
	socket, err := s.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	} else {
		defer socket.Close()
	}

	// ...
	ctx = context.WithValue(ctx, schemaKey, s.schema)
	ctx = context.WithValue(ctx, authKey, identifier)
	ctx = context.WithValue(ctx, socketKey, socket)

	// enter connection loop - this will block until
	// client disconnects or until there is an error that
	// forcefully closes the connection
	connectionLoop(ctx)
}
