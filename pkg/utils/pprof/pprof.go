package pprof

import (
	"context"
	"expvar"
	"net"
	"net/http"
	"net/http/pprof"
	"os"

	"github.com/sunweiwe/kuber/pkg/log"
)

func newHandler() http.Handler {
	m := http.NewServeMux()
	m.Handle("/debug/vars", expvar.Handler())
	m.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	m.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	m.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	m.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	m.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	return m
}

func Run(ctx context.Context) error {
	var port string
	port = os.Getenv("KUBER_PPROF_PORT")
	if port == "" {
		port = ":6060"
	}
	server := http.Server{
		Addr:    port,
		Handler: newHandler(),
		BaseContext: func(l net.Listener) context.Context {
			return ctx
		},
	}
	log := log.FromContextOrDiscard(ctx)
	go func() {
		<-ctx.Done()
		_ = server.Shutdown(ctx)
		log.Info("pprof stopped")
	}()

	log.Info("debug pprof listen", "addr", server.Addr)
	return server.ListenAndServe()
}
