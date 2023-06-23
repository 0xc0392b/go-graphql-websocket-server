package gqlwss

import (
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/graphql-go/graphql"
)

func newTestClientServer(h http.Handler) (*websocket.Conn, *httptest.Server, error) {
	s := httptest.NewServer(h)
	url := "ws" + strings.TrimPrefix(s.URL, "http")
	c, _, err := websocket.DefaultDialer.Dial(url, nil)
	return c, s, err
}

func newTestSchema() (*graphql.Schema, error) {
	userResolver := func(p graphql.ResolveParams) (interface{}, error) {
		return testUser{1, "test user", "test@user.com"}, nil
	}

	tickResolver := func(p graphql.ResolveParams) (interface{}, error) {
		return p.Source, nil
	}

	tickSubscriber := func(p graphql.ResolveParams) (interface{}, error) {
		ticks := make(chan interface{})

		go func() {
			i := int64(1)
			stop := false

			for !stop {
				select {
				case <-p.Context.Done():
					stop = true
					break
				default:
					if i > 100 {
						stop = true
					} else {
						ticks <- testTick{i}
						i++
					}
				}
			}

			close(ticks)
		}()

		return ticks, nil
	}

	userType := graphql.NewObject(
		graphql.ObjectConfig{
			Name: "User",
			Fields: graphql.Fields{
				"id":    &graphql.Field{Type: graphql.Int},
				"name":  &graphql.Field{Type: graphql.String},
				"email": &graphql.Field{Type: graphql.String},
			},
		},
	)

	tickType := graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Tick",
			Fields: graphql.Fields{
				"value": &graphql.Field{Type: graphql.Int},
			},
		},
	)

	queryType := graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Query",
			Fields: graphql.Fields{
				"getUser": &graphql.Field{
					Type:        userType,
					Resolve:     userResolver,
					Description: "Get test user",
				},
			},
		},
	)

	subscriptionType := graphql.NewObject(
		graphql.ObjectConfig{
			Name: "Subscription",
			Fields: graphql.Fields{
				"tick": &graphql.Field{
					Type:        tickType,
					Resolve:     tickResolver,
					Subscribe:   tickSubscriber,
					Description: "Tick tock tick tock...",
				},
			},
		},
	)

	if schema, err := graphql.NewSchema(
		graphql.SchemaConfig{
			Query:        queryType,
			Subscription: subscriptionType,
		},
	); err != nil {
		return nil, err
	} else {
		return &schema, nil
	}
}

func NewAuth() Auth {
	return Auth{}
}

func NewGraphQLSocket(schema graphql.Schema) GraphQLSocket {
	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	return GraphQLSocket{schema, upgrader}
}

func NewServerWithAuth(opts Options) Server {
	auth := NewAuth()
	graphql := NewGraphQLSocket(opts.Schema)

	mux := http.NewServeMux()
	mux.Handle(opts.AuthPath, auth)
	mux.Handle(opts.GraphQLPath, graphql)

	return Server{mux}
}

func NewServerWithoutAuth(opts Options) Server {
	graphql := NewGraphQLSocket(opts.Schema)

	mux := http.NewServeMux()
	mux.Handle(opts.GraphQLPath, graphql)

	return Server{mux}
}
