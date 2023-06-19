package gqlwss

import (
	"fmt"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func TestServer(t *testing.T) {

	// ..

}

func TestAuth(t *testing.T) {

	// ..

}

func TestGraphQL(t *testing.T) {
	schema, err := newTestSchema()
	if err != nil {
		t.Fatal(err)
	}

	handler := NewGraphQLSocket(*schema)

	c, s, err := newTestClientServer(handler)
	if err != nil {
		t.Fatal(err)
	} else {
		defer c.Close()
		defer s.Close()
	}

	in1 := `{"id": 1, "type": "ConnectionInit"}`
	c.WriteMessage(websocket.TextMessage, []byte(in1))

	in2 := `{"id": 2, "type": "Subscribe", "payload": "query { getUser { id, name } }"}`
	c.WriteMessage(websocket.TextMessage, []byte(in2))

	in3 := `{"id": 3, "type": "Subscribe", "payload": "query { getUser { name } }"}`
	c.WriteMessage(websocket.TextMessage, []byte(in3))

	count := 0

	for count < 5 {
		c.SetReadDeadline(time.Now().Add(time.Second * 5))
		mt, out, err := c.ReadMessage()
		if err != nil {
			t.Fatal(err)
		}

		if mt != 1 {
			t.Error("Expected message type to be 1, got: ", mt)
		}

		fmt.Println(string(out))

		count += 1
	}
}
