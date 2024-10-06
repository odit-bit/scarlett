package redis

import (
	"github.com/odit-bit/scarlett/store"
	"github.com/tidwall/redcon"
)

type StoreHandler struct {
	s *store.Store
}

func NewStoreHandler(s *store.Store) (StoreHandler, error) {
	return StoreHandler{
		s: s,
	}, nil
}

func (rh *StoreHandler) Accept(conn redcon.Conn) bool {
	return true
}

func (rh *StoreHandler) Close(conn redcon.Conn, err error) {
	// return errors.Join(err, conn.Close())
}

func (rh *StoreHandler) Handle(conn redcon.Conn, cmd redcon.Command) {
	// switch strings.ToLower(string(cmd.Args[0])) {
	// case "set":
	// 	if len(cmd.Args) <= 3 {
	// 		msg := fmt.Sprintf("ERR wrong number of arguments for [%s] ", string(cmd.Args[0]))
	// 		conn.WriteError(msg)
	// 		return
	// 	}
	// 	if err := rh.s.Set(cmd.Args[1], cmd.Args[2], 0); err != nil {
	// 		conn.WriteError(fmt.Sprintf("Err %s", err))
	// 		return
	// 	}
	// 	conn.WriteString("OK")

	// case "get":
	// 	if len(cmd.Args) != 2 {
	// 		msg := fmt.Sprintf("ERR wrong number of arguments for [%s] ", string(cmd.Args[0]))
	// 		conn.WriteError(msg)
	// 		return
	// 	}

	// 	v, ok, err := rh.s.Get(string(cmd.Args[1]))
	// 	if err != nil {
	// 		conn.WriteError(err.Error())
	// 		return
	// 	}
	// 	if !ok {
	// 		conn.WriteNull()
	// 		return
	// 	} else {
	// 		conn.WriteBulk(v)
	// 	}
	// case "del":
	// 	if len(cmd.Args) != 2 {
	// 		msg := fmt.Sprintf("ERR wrong number of arguments for [%s] ", string(cmd.Args[0]))
	// 		conn.WriteError(msg)
	// 		return
	// 	}
	// 	if err := rh.s.Delete(cmd.Args[1]); err != nil {
	// 		msg := fmt.Sprintf("ERR %s ", err)
	// 		conn.WriteError(msg)
	// 		return
	// 	}
	// default:
	// 	err := fmt.Errorf("ERR unknown command [%s]", string(cmd.Args[0]))
	// 	conn.WriteError(err.Error())
	// 	return
	// }

}
