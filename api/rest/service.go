package rest

import (
	"context"
	"crypto/tls"
	"errors"
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

	return router

}

// run will running blocked http server.
func (h *Service) Run(addr string, conf *tls.Config) error {
	//node http server
	h.srv = &http.Server{Addr: addr, Handler: h.mux, ErrorLog: slog.NewLogLogger(h.logger.Handler(), slog.LevelInfo), TLSConfig: conf}
	h.errCh <- h.srv.ListenAndServe()
	close(h.errCh)

	return nil
}

func (h *Service) Serve(l net.Listener, conf *tls.Config) error {
	//node http server
	h.srv = &http.Server{Handler: h.mux, ErrorLog: slog.NewLogLogger(h.logger.Handler(), slog.LevelInfo), TLSConfig: conf}
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
