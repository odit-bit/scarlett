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

	// cluster ClusterClient
	store *store.Store
	// routes  map[string]string

	errCh chan error

	mutex *sync.Mutex
}

func NewServer(db *store.Store, l *slog.Logger) Service {
	router := Service{
		logger: l,
		mux:    http.NewServeMux(),
		srv:    &http.Server{},
		// cluster: clstr,
		store: db,
		// routes:  map[string]string{},
		errCh: make(chan error),
		mutex: &sync.Mutex{},
	}

	return router

}

func (h *Service) Serve(l net.Listener, conf *tls.Config) error {
	//node http server
	h.srv = &http.Server{
		Addr:                         l.Addr().String(),
		Handler:                      h.mux,
		DisableGeneralOptionsHandler: false,
		TLSConfig:                    conf,
		ReadTimeout:                  5 * time.Second,
		ReadHeaderTimeout:            0,
		WriteTimeout:                 5 * time.Second,
		IdleTimeout:                  20 * time.Second,
		MaxHeaderBytes:               1024,
		ErrorLog:                     slog.NewLogLogger(h.logger.Handler(), slog.LevelError),
	}
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

	return err
}
