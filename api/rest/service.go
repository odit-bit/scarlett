package rest

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/odit-bit/scarlett/store"
)

type ClusterClient interface {
	//it should return the node remote address in cluster
	GetNodeAPI(ctx context.Context, nodeAddr string) (string, error)
}

type Service struct {
	logger *slog.Logger

	mux *http.ServeMux
	srv *http.Server

	cluster ClusterClient
	store   *store.Store
	routes  map[string]string

	errCh chan error

	mutex *sync.Mutex
}

func NewServer(db *store.Store, l *slog.Logger, clstr ClusterClient) Service {
	var err error
	// if tc != nil {
	// 	l.Info("store use tls connection")
	// 	clstr, err = cluster.NewTLSClient(tc)
	// } else {
	// 	l.Info("store use non-tls connection")
	// 	clstr, err = cluster.NewClient()
	// }

	if err != nil {
		l.Error(err.Error())
		return Service{}
	}

	router := Service{
		logger:  l,
		mux:     http.NewServeMux(),
		srv:     &http.Server{},
		cluster: clstr,
		store:   db,
		routes:  map[string]string{},
		errCh:   make(chan error),
		mutex:   &sync.Mutex{},
	}

	for k, v := range APIRoute {
		m, p := v.Route()
		switch k {

		// case JOIN_API:
		// 	router.handleJoin(m, p)
		// 	router.routes["join"] = fmt.Sprintf("%s %s", m, p)

		case COMMAND_API:
			router.handleCommand(m, p)
			router.routes["command"] = fmt.Sprintf("%s %s", m, p)

		case QUERY_API:
			router.handleQuery(m, p)
			router.routes["query"] = fmt.Sprintf("%s %s", m, p)

		}
	}

	return router

}

func (h *Service) Mux() *http.ServeMux {
	return h.mux
}

// run will running blocked http server.
func (h *Service) Run(addr string, conf *tls.Config) error {
	//node http server
	h.srv = &http.Server{Addr: addr, Handler: h.Mux(), ErrorLog: slog.NewLogLogger(h.logger.Handler(), slog.LevelInfo), TLSConfig: conf}
	h.errCh <- h.srv.ListenAndServe()
	close(h.errCh)

	return nil
}

func (h *Service) Serve(l net.Listener, conf *tls.Config) error {
	//node http server
	h.srv = &http.Server{Handler: h.Mux(), ErrorLog: slog.NewLogLogger(h.logger.Handler(), slog.LevelInfo), TLSConfig: conf}
	h.errCh <- h.srv.Serve(l)
	close(h.errCh)

	return nil
}

func (h *Service) Stop() error {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err = errors.Join(err, h.srv.Shutdown(ctx))

	select {
	case <-ctx.Done():
		err = errors.Join(err, ctx.Err(), h.srv.Close())
	case e := <-h.errCh:
		err = errors.Join(err, e)
	}

	h.logger.Info("http-server-shutdown", "error", err)
	return err
}

func (h *Service) handleCommand(method, path string) {
	endpoint := fmt.Sprintf("%s %s", method, path)
	h.mux.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {

		// is leader ?
		if ok := h.isRedirect(w, r); ok {
			return
		}

		defer r.Body.Close()
		cmd := struct {
			Cmd   string
			Key   string
			Value string
		}{}
		if err := json.NewDecoder(r.Body).Decode(&cmd); err != nil {
			h.writeErrResponse(w, err)
			h.logger.Error(err.Error())
			return
		}

		res := store.CommandResponse{}
		if err := h.store.Command(store.CMDType(cmd.Cmd), []byte(cmd.Key), []byte(cmd.Value), &res); err != nil {
			h.writeErrResponse(w, err)
			h.logger.Error(err.Error())
			return
		}
		h.writeResponse(w, map[string]any{"msg": res.Message, "err": res.Err})
	})
}

func (h *Service) handleQuery(method, path string) {
	endpoint := fmt.Sprintf("%s %s", method, path)
	h.mux.HandleFunc(endpoint, func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		args, err := io.ReadAll(r.Body)
		if err != nil {
			h.writeErrResponse(w, err)
			return
		}

		cmd := map[string]string{}
		if err := json.Unmarshal(args, &cmd); err != nil {
			h.writeErrResponse(w, err)
			return
		}

		res, err := h.store.Query(r.Context(), store.QueryType(cmd["cmd"]), []byte(cmd["key"]))
		if err != nil {
			h.writeErrResponse(w, err)
			return
		}
		if res.Err != nil {
			h.writeErrResponse(w, res.Err)
			return
		}
		h.writeResponse(w, map[string]any{"value": string(res.Value)})
	})
}

// if return true is indicate current node is not a leader of the cluster,
// either it redirect to actual leader or response error.
// value is true only if current node is leader
func (h *Service) isRedirect(w http.ResponseWriter, r *http.Request) bool {
	raftLeader, _ := h.store.GetLeader()
	current := h.store.Addr()
	if raftLeader != current {
		httpLeader, err := h.cluster.GetNodeAPI(r.Context(), raftLeader)
		if err != nil {
			h.logger.Error("http-server", "error", err.Error(), "type", fmt.Sprintf("%T", err))
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

func (rest *Service) writeResponse(w http.ResponseWriter, arg any) {
	b, err := json.Marshal(arg)
	if err != nil {
		http.Error(w, "", 500)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	w.Write(b)
}

func (router *Service) writeErrResponse(w http.ResponseWriter, err error) {
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
