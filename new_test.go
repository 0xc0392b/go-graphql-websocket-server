package gqlwss

import (
	"fmt"
	"testing"

	"github.com/graphql-go/graphql"
)

func TestNew(t *testing.T) {
	// ...
	userResolver := func(p graphql.ResolveParams) (interface{}, error) {
		return nil, nil
	}

	// ...
	userType := graphql.NewObject(
		graphql.ObjectConfig{
			Name: "User",
			Fields: graphql.Fields{
				"id":           &graphql.Field{Type: graphql.Int},
				"firstName":    &graphql.Field{Type: graphql.String},
				"emailAddress": &graphql.Field{Type: graphql.String},
			},
		},
	)

	// ...
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

	schema, err := graphql.NewSchema(
		graphql.SchemaConfig{
			Query: queryType,
		},
	)
	if err != nil {
		t.Error(err)
	}

	server := New(Options{
		Schema:      schema,
		AuthPath:    "/auth",
		GraphQLPath: "/graphql",
	})

	fmt.Println(server)

}
