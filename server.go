package gqlwss

import (
	"context"
	"net/http"

	"github.com/gorilla/websocket"
)

func upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	// setup upgrader
	upgrader := &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	// do the upgrade and return pointer to socket
	if socket, err := upgrader.Upgrade(w, r, nil); err != nil {
		return nil, err
	} else {
		return socket, nil
	}
}

func auth(r *http.Request) (string, error) {

	// ...

	return "", nil
}

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
		return
	} else {
		// success - add key to context
		ctx = context.WithValue(ctx, authKey, key)
	}

	// do connection upgrade
	socket, err := upgrade(w, r)
	if err != nil {
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
