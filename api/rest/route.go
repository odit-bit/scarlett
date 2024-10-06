package rest

import "net/http"

const (
	JOIN_API    = "JOIN"
	COMMAND_API = "COMMAND"
	QUERY_API   = "QUERY"
)

type API interface {
	Route() (method, path string)
}

var APIRoute = map[string]API{
	JOIN_API:    &JoinRoute{},
	COMMAND_API: &CommandRoute{},
	QUERY_API:   &QueryRoute{},
}

type JoinRoute struct{}

func (jr *JoinRoute) Route() (string, string) {
	return http.MethodGet, "/join"
}

type CommandRoute struct{}

func (cr *CommandRoute) Route() (string, string) {
	return http.MethodPost, "/command"
}

type QueryRoute struct{}

func (qr *QueryRoute) Route() (string, string) {
	return http.MethodGet, "/query"
}

type RaftJoinRequest struct {
	NodeID  string
	Address string
}

type RaftJoinResponse struct{}

type RaftRestartCluster struct{}

func (rrc *RaftRestartCluster) Route() (string, string) {
	return http.MethodGet, "/raft/restart"
}
