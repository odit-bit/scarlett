package rest

import (
	"testing"
)

type clusterMock struct{}

func (clusterMock) GetNodeApiAddr(addr string) string {
	return "node_api_address"
}

func Test_route(t *testing.T) {
	// tt := map[string]string{
	// 	"join":    "GET /join",
	// 	"command": "POST /command",
	// 	"query":   "GET /query",
	// }
	// r := NewServer(nil, slog.Default(), nil)
	// for tk, tv := range tt {
	// 	v, ok := r.routes[tk]
	// 	if !ok {
	// 		t.Fatal("endpoint should exist", tv)
	// 	}
	// 	assert.Equal(t, tv, v)
	// }

	//
	//
	//
	// for k, v := range r.routes {
	// 	switch k {
	// 	case "set":
	// 		assert.Equal(t, "POST /set", v)
	// 	case "get":
	// 		assert.Equal(t, "GET /get", v)
	// 	case "join":
	// 		assert.Equal(t, "GET /join", v)
	// 	case "delete":
	// 		assert.Equal(t, "DELETE /delete", v)
	// 	default:
	// 		t.Fatal("unknown route", k)
	// 	}
	// }
}
