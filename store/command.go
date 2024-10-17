package store

import (
	"errors"
	// clusterpb "github.com/odit-bit/scarlett/store/api/cluster/proto"
)

const (
	SET_CMD_TYPE    CMDType = "set"
	DELETE_CMD_TYPE CMDType = "delete"

	GET_QUERY_TYPE QueryType = "get"
)

type CMDType string
type QueryType string

var ErrNotLeader = errors.New("only leader could change the state")

type Command struct {
	Cmd  string
	Args [][]byte
}

type CommandResponse struct {
	Message string
}

type QueryResponse struct {
	Value []byte
}
