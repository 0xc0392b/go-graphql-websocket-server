package gqlwss

import (
	"context"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/graphql-go/graphql/language/source"
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

func connectionLoop(ctx context.Context) {
	// connectionLoop can cancel the connection context
	ctx, cancel := context.WithCancelCause(ctx)

	// create IO + error channels
	ctx = context.WithValue(ctx, inputsKey, make(chan message))
	ctx = context.WithValue(ctx, outputsKey, make(chan message))
	ctx = context.WithValue(ctx, errorsKey, make(chan error))

	// the connection loop needs to know about inputs and errors
	inputs := ctx.Value(inputsKey).(chan message)
	errors := ctx.Value(errorsKey).(chan error)

	// create connection's initial state
	current := state{}

	// one concurrent reader and one concurrent writer
	go readLoop(ctx)
	go writeLoop(ctx)

	for {
		select {
		// read error, cancel connection context, stop loop
		case error := <-errors:
			// debugging
			log.Println(error)

			// stops read and write and write loops as well as
			// all active subscription routines
			cancel(error)
			break

		// read next input, do state transition
		case input := <-inputs:
			// debugging
			log.Println(input.String())

			// each input gets its own context
			// attach current state and input to context
			ctx = context.WithValue(ctx, currentStateKey, current)
			ctx = context.WithValue(ctx, messageKey, input)

			// do the state transition
			next, action := stateTransition(ctx)

			// replce current state and do the given action
			current = next
			go action(ctx)
		}
	}
}

func writeLoop(ctx context.Context) {
	// retrieve the socket and outputs channel from context
	socket := ctx.Value(socketKey).(*websocket.Conn)
	outputs := ctx.Value(outputsKey).(chan message)
	errors := ctx.Value(errorsKey).(chan error)

	for {
		select {
		// connection context has been cancelled
		case <-ctx.Done():
			break

		// wait for output message
		case output := <-outputs:
			if err := socket.WriteJSON(output); err != nil {
				// signal to connection loop that it should
				// cancel the context
				errors <- err
			}
		}
	}
}

func readLoop(ctx context.Context) {
	// retrieve the socket and inputs channel from context
	socket := ctx.Value(socketKey).(*websocket.Conn)
	inputs := ctx.Value(inputsKey).(chan message)
	errors := ctx.Value(errorsKey).(chan error)

	// TODO: set read size limit
	// ... socket.SetReadLimit(maxMessageSize)

	// TODO: set read deadline
	// ... socket.SetReadDeadline(time.Now().Add(pongWait))

	// TODO: set pong handler
	// ... socket.SetPongHandler(func(string) { ... })

	for {
		select {
		// connection context has been cancelled
		case <-ctx.Done():
			break

		// wait for input message
		default:
			input := message{}

			if err := socket.ReadJSON(&input); err != nil {
				// signal to connection loop that it should
				// cancel the context
				errors <- err
			} else {
				inputs <- input
			}
		}
	}
}

func stateTransition(ctx context.Context) (state, action) {
	// ...
	current := ctx.Value(currentStateKey).(state)
	input := ctx.Value(inputsKey).(message)

	// see the following link for graphql-ws protocol spec
	// https://github.com/enisdenjo/graphql-ws/blob/master/PROTOCOL.md
	switch input.Type {
	// Useful for detecting failed connections, displaying latency
	// metrics or other types of network probing. A Pong must be
	// sent in response from the receiving party as soon as
	// possible.
	case "Ping":
		return current, pong

	// The response to the Ping message. Must be sent as soon as
	// the Ping message is received. The Pong message can be sent
	// at any time within the established socket. Furthermore, the
	// Pong message may even be sent unsolicited as an
	// unidirectional heartbeat.
	case "Pong":
		return current, nothing

	// Indicates that the client wants to establish a connection
	// within the existing socket. This connection is not the
	// actual WebSocket communication channel, but is rather a
	// frame within it asking the server to allow future operation
	// requests.
	case "ConnectionInit":
		if current.isInitialised {
			return current, errorAlreadyInitialised
		} else {
			next := current
			next.isInitialised = true
			next.operations = make(map[int64]operation)
			return next, connectionAck
		}

	// Requests an operation specified in the message
	// payload. This message provides a unique ID field to connect
	// published messages to the operation requested by this
	// message. If there is already an active subscriber for an
	// operation matching the provided ID, regardless of the
	// operation type, the server must close the socket
	// immediately.
	case "Subscribe":
		if !current.isInitialised {
			return current, errorNotInitialised
		} else if _, exists := current.operations[input.Id]; exists {
			return current, errorIdAlreadyExists
		} else {
			ctx, cancel := context.WithCancel(ctx)
			next := current
			next.operations[input.Id] = operation{ctx, cancel}
			return next, subscribe
		}

	// Indicates that the client has stopped listening and wants
	// to complete the subscription. No further events, relevant
	// to the original subscription, should be sent through. Even
	// if the client sent a Complete message for a
	// single-result-operation before it resolved, the result
	// should not be sent through once it does.
	case "Complete":
		if !current.isInitialised {
			return current, errorNotInitialised
		} else if _, exists := current.operations[input.Id]; !exists {
			return current, errorIdDoesNotExist
		} else {
			next := current
			next.operations[input.Id].cancel()
			delete(next.operations, input.Id)
			return next, nothing
		}

	// Receiving a message of a type or format which is not
	// specified in this document will result in an immediate
	// socket closure.
	default:
		return current, errorUnknownMessageType
	}
}

func nothing(ctx context.Context) {
	// noop
}

func pong(ctx context.Context) {
	// ...
	outputs := ctx.Value(outputsKey).(chan message)
	outputs <- message{Type: "Pong"}
}

func connectionAck(ctx context.Context) {
	// ...
	outputs := ctx.Value(outputsKey).(chan message)
	outputs <- message{Type: "ConnectionAck"}
}

func errorAlreadyInitialised(ctx context.Context) {
	// ...
	outputs := ctx.Value(outputsKey).(chan message)
	outputs <- message{Type: "Error"}
}

func errorNotInitialised(ctx context.Context) {
	// ...
	outputs := ctx.Value(outputsKey).(chan message)
	outputs <- message{Type: "Error"}
}

func errorIdAlreadyExists(ctx context.Context) {
	// ...
	outputs := ctx.Value(outputsKey).(chan message)
	outputs <- message{Type: "Error"}
}

func errorIdDoesNotExist(ctx context.Context) {
	// ...
	outputs := ctx.Value(outputsKey).(chan message)
	outputs <- message{Type: "Error"}
}

func errorUnknownMessageType(ctx context.Context) {
	// ...
	outputs := ctx.Value(outputsKey).(chan message)
	outputs <- message{Type: "Error"}
}
