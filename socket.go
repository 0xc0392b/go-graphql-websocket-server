package gqlwss

import (
	"context"
	"encoding/json"
	"log"
	"strconv"

	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
	"github.com/graphql-go/graphql/language/parser"
	"github.com/graphql-go/graphql/language/source"
)

func (i opId) String() string {
	return strconv.FormatInt(int64(i), 10)
}

func (m message) String() string {
	return "[" + m.Id.String() + "] " + m.Type + ": " + m.Payload
}

func (m opMap) Exists(id opId) bool {
	if _, ok := m[id]; ok {
		return true
	} else {
		return false
	}
}

func (m opMap) Add(id opId, ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	m[id] = op{ctx, cancel}
}

func (m opMap) Remove(id opId) {
	m[id].cancel()
	delete(m, id)
}

func connectionLoop(ctx context.Context) {
	// connectionLoop can cancel the connection context
	ctx, cancel := context.WithCancelCause(ctx)

	// create IO + error channels
	ctx = context.WithValue(ctx, inputsKey, make(chan message))
	ctx = context.WithValue(ctx, outputsKey, make(chan message))
	ctx = context.WithValue(ctx, errorsKey, make(chan error))

	// the connection loop only needs to know about inputs and errors
	inputs := ctx.Value(inputsKey).(chan message)
	errors := ctx.Value(errorsKey).(chan error)

	// create connection's initial state
	current := state{}

	// one concurrent reader and one concurrent writer
	go readLoop(ctx)
	go writeLoop(ctx)

	// stop flag will be set to true when an error is received
	stop := false

	for !stop {
		select {
		// read error, cancel connection context, stop loops
		case error := <-errors:
			log.Println(error)
			cancel(error)
			stop = true
			break

		// read next input, do state transition
		case input := <-inputs:
			log.Println(input.String())

			// each input gets its own context
			ctx = context.WithValue(ctx, currentStateKey, current)
			ctx = context.WithValue(ctx, messageKey, input)

			next, action := stateTransition(ctx)
			current = next
			go action(ctx)
		}
	}
}

func readLoop(ctx context.Context) {
	// read loop needs to know about the socket, inputs, and errors
	socket := ctx.Value(socketKey).(*websocket.Conn)
	inputs := ctx.Value(inputsKey).(chan message)
	errors := ctx.Value(errorsKey).(chan error)

	// TODO: set read size limit
	// ... socket.SetReadLimit(maxMessageSize)

	// TODO: set read deadline
	// ... socket.SetReadDeadline(time.Now().Add(pongWait))

	// TODO: set pong handler
	// ... socket.SetPongHandler(func(string) { ... })

	// stop flag will be set to true when an error is received
	stop := false

	for !stop {
		select {
		// connection context has been cancelled
		case <-ctx.Done():
			stop = true
			break

		// wait for input message
		default:
			input := message{}

			if err := socket.ReadJSON(&input); err != nil {
				// signal to connection loop that it should
				// cancel the context
				errors <- err
			} else {
				// send input to connectop loop
				inputs <- input
			}
		}
	}
}

func writeLoop(ctx context.Context) {
	// write loop needs to know about the socket, ouputs, and errors
	socket := ctx.Value(socketKey).(*websocket.Conn)
	outputs := ctx.Value(outputsKey).(chan message)
	errors := ctx.Value(errorsKey).(chan error)

	// stop flag will be set to true when an error is received
	stop := false

	for !stop {
		select {
		// connection context has been cancelled
		case <-ctx.Done():
			stop = true
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

func stateTransition(ctx context.Context) (state, action) {
	// state transition needs to know about the current state and inputs
	current := ctx.Value(currentStateKey).(state)
	input := ctx.Value(messageKey).(message)

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
			next.ops = make(opMap)
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
		} else if current.ops.Exists(input.Id) {
			return current, errorIdAlreadyExists
		} else {
			next := current
			next.ops.Add(input.Id, ctx)
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
		} else if !current.ops.Exists(input.Id) {
			return current, errorIdDoesNotExist
		} else {
			next := current
			next.ops.Remove(input.Id)
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

func subscribe(ctx context.Context) {
	// subscribe needs to know about the schema, inputs, and outputs
	schema := ctx.Value(schemaKey).(graphql.Schema)
	input := ctx.Value(messageKey).(message)
	outputs := ctx.Value(outputsKey).(chan message)

	// ...
	params := graphql.Params{
		Schema:        schema,
		RequestString: input.Payload,
	}

	// ...
	source := source.NewSource(&source.Source{
		Body: []byte(params.RequestString),
		Name: "GraphQL request",
	})

	// parse the graphql input
	ast, err := parser.Parse(parser.ParseParams{Source: source})
	if err != nil {
		return
	}

	// validate ast against the schema
	validation := graphql.ValidateDocument(&schema, ast, nil)
	if !validation.IsValid {
		return
	}

	// ...
	execute := graphql.ExecuteParams{
		AST:           ast,
		Schema:        params.Schema,
		Root:          params.RootObject,
		Args:          params.VariableValues,
		OperationName: params.OperationName,
	}

	// ...
	results := make(chan *graphql.Result)
	stop := false

	// ...
	go func() {
		result := graphql.Execute(execute)
		results <- result
		close(results)
	}()

	for !stop {
		select {
		// connection context has been cancelled
		case <-ctx.Done():
			stop = true
			break

		// ...
		case result, ok := <-results:
			if ok {
				if payload, err := json.Marshal(result); err != nil {
					outputs <- message{
						Id:   input.Id,
						Type: "Error",
					}
				} else {
					outputs <- message{
						Id:      input.Id,
						Type:    "Next",
						Payload: string(payload),
					}
				}
			} else {
				stop = true
				break
			}
		}
	}

	// need to remove the operation from opmap
	// ...

}

func pong(ctx context.Context) {
	input := ctx.Value(messageKey).(message)
	outputs := ctx.Value(outputsKey).(chan message)
	outputs <- message{Id: input.Id, Type: "Pong"}
}

func connectionAck(ctx context.Context) {
	input := ctx.Value(messageKey).(message)
	outputs := ctx.Value(outputsKey).(chan message)
	outputs <- message{Id: input.Id, Type: "ConnectionAck"}
}

func errorAlreadyInitialised(ctx context.Context) {
	input := ctx.Value(messageKey).(message)
	outputs := ctx.Value(outputsKey).(chan message)
	outputs <- message{Id: input.Id, Type: "Error"}
}

func errorNotInitialised(ctx context.Context) {
	input := ctx.Value(messageKey).(message)
	outputs := ctx.Value(outputsKey).(chan message)
	outputs <- message{Id: input.Id, Type: "Error"}
}

func errorIdAlreadyExists(ctx context.Context) {
	input := ctx.Value(messageKey).(message)
	outputs := ctx.Value(outputsKey).(chan message)
	outputs <- message{Id: input.Id, Type: "Error"}
}

func errorIdDoesNotExist(ctx context.Context) {
	input := ctx.Value(messageKey).(message)
	outputs := ctx.Value(outputsKey).(chan message)
	outputs <- message{Id: input.Id, Type: "Error"}
}

func errorUnknownMessageType(ctx context.Context) {
	input := ctx.Value(messageKey).(message)
	outputs := ctx.Value(outputsKey).(chan message)
	outputs <- message{Id: input.Id, Type: "Error"}
}
