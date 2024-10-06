package store

import (
	"errors"
	"time"
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
	Cmd   string
	Key   string
	Value string
}

type CommandResponse struct {
	Cmd     string
	Message string
	Err     error
}

type QueryResponse struct {
	Value     []byte
	Message   string
	ExpiredIn time.Duration
	Err       error
}

// func unmarshalCommand(b []byte, cmd *Command) error {
// 	err := json.Unmarshal(b, cmd)
// 	if err != nil {
// 		return err
// 	}
// 	return nil
// }

// func UnmarshalCommand(b []byte, cmd *Command) error {
// 	return unmarshalCommand(b, cmd)
// }
