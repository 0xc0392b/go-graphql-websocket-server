package gqlwss

import (
	"net/http"
)

func (s Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.mux.ServeHTTP(w, r)
}

func (s Auth) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// TODO auth logic
	// ...

}

func (s GraphQL) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// setup connection context
	ctx := context.WithValue(r.Context(), schemaKey, s.schema)

	// do authentication
	key, err := auth(r)
	if err != nil {
		log.Println(err)
		return
	} else {
		// success - add key to context
		ctx = context.WithValue(ctx, authKey, key)
	}

	// do connection upgrade
	socket, err := upgrade(w, r)
	if err != nil {
		log.Println(err)
		return
	} else {
		// success - add socket to context
		ctx = context.WithValue(ctx, socketKey, socket)
		defer socket.Close()
	}

	// enter connection loop - this will block until
	// client disconnects or until there is an error that
	// forcefully closes the connection
	connectionLoop(ctx)
}
