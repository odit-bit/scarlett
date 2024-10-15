package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/odit-bit/scarlett/store"
)

type storeService struct {
	// logger *slog.Logger
	mux *http.ServeMux
}

func AddStore(svc *Service) {
	router := storeService{
		mux: svc.mux,
	}

	router.handleCommand(http.MethodPost, "/command", svc.store)
	router.handleQuery(http.MethodGet, "/query", svc.store)
}

func (s *storeService) handleCommand(method, path string, db *store.Store) {
	endpoint := fmt.Sprintf("%s %s", method, path)

	s.mux.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {

		// // is leader ?
		// if ok := s.isRedirect(w, r, db, clstr); ok {
		// 	return
		// }

		defer r.Body.Close()
		cmd := struct {
			Cmd   string
			Key   string
			Value string
		}{}
		if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
			writeErrResponse(w, err)
			// s.logger.Error(err.Error())
			return
		}

		if res, err := db.Command(r.Context(), store.CMDType(cmd.Cmd), []byte(cmd.Key), []byte(cmd.Value)); err != nil {
			writeErrResponse(w, err)
			// s.logger.Error(err.Error())
			return
		} else {
			writeResponse(w, map[string]any{"msg": res.Message})
		}
	})
}

func (h *storeService) handleQuery(method, path string, db *store.Store) {
	endpoint := fmt.Sprintf("%s %s", method, path)
	h.mux.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		args, err := io.ReadAll(r.Body)
		if err != nil {
			writeErrResponse(w, err)
			return
		}

		cmd := map[string]string{}
		if err := json.Unmarshal(args, &cmd); err != nil {
			writeErrResponse(w, err)
			return
		}

		res, _, err := db.Get(r.Context(), []byte(cmd["key"]))
		if err != nil {
			writeErrResponse(w, err)
			return
		} else {
			writeResponse(w, map[string]any{"value": string(res)})
		}
	})
}

// if return true is indicate current node is not a leader of the cluster,
// either it redirect to actual leader or response error.
// value is true only if current node is leader
func (h *storeService) isRedirect(w http.ResponseWriter, r *http.Request, db *store.Store, clstr ClusterClient) bool {
	raftLeader, _ := db.Leader()
	current := db.Addr()
	if raftLeader != current {
		httpLeader, err := clstr.GetNodeAPI(r.Context(), raftLeader)
		if err != nil {
			// h.logger.Error("http-server", "error", err.Error(), "type", fmt.Sprintf("%T", err))
			http.Error(w, "leader not found", 500)
			return true
		}
		scheme := "http"
		if r.TLS != nil {
			scheme = "https"
		}
		redirectUrl := fmt.Sprintf("%s://%s%s", scheme, httpLeader, r.URL.Path)

		http.Redirect(w, r, redirectUrl, http.StatusPermanentRedirect)
		return true
	}

	return false
}

func writeResponse(w http.ResponseWriter, arg any) {
	b, err := json.Marshal(arg)
	if err != nil {
		http.Error(w, "", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(b)
}

func writeErrResponse(w http.ResponseWriter, err error) {
	code := 400
	if err != nil {
		if err == store.ErrNotLeader {
			err = fmt.Errorf("cluster leader is not chosen")
			code = 500
		}
	}

	b, err := json.Marshal(map[string]string{"error": err.Error()})
	if err != nil {
		http.Error(w, "", code)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(b)

}
