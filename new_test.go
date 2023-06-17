package gqlwss

import (
	"fmt"
	"testing"
)

func TestNewServerWithAuth(t *testing.T) {
	schema, err := newTestSchema()
	if err != nil {
		t.Fatal(err)
	}

	server := NewServerWithAuth(Options{
		Schema:      *schema,
		AuthPath:    "/auth",
		GraphQLPath: "/graphql",
	})

	fmt.Println(server)

}

func TestNewServerWithoutAuth(t *testing.T) {
	schema, err := newTestSchema()
	if err != nil {
		t.Fatal(err)
	}

	server := NewServerWithoutAuth(Options{
		Schema:      *schema,
		GraphQLPath: "/graphql",
	})

	fmt.Println(server)

}
