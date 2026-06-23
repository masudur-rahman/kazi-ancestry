package cmd

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/masudur-rahman/kazi-ancestry/api/web"
	"github.com/masudur-rahman/kazi-ancestry/configs"
	"github.com/masudur-rahman/kazi-ancestry/infra/logr"
	"github.com/masudur-rahman/kazi-ancestry/services/all"

	"github.com/spf13/cobra"
)

// serveCmd runs the HTTP server.
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the Kazi Ancestry web server",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		if err := configs.InitiateDatabaseConnection(ctx); err != nil {
			log.Fatalln(err)
		}
		// Ensure the tree exists on first boot (idempotent).
		if n, err := all.GetServices().Person.Seed(configs.KaziConfig.SeedPath); err != nil {
			log.Fatalln(err)
		} else {
			logr.DefaultLogger.Infof("person table ready: %d people", n)
		}

		cfg := configs.KaziConfig.Server
		addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
		router := web.NewRouter(cfg.WebDir)
		srv := &http.Server{Addr: addr, Handler: router, ReadHeaderTimeout: 10 * time.Second}

		stopCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()

		go func() {
			logr.DefaultLogger.Infof("server listening on %s", addr)
			if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
				log.Fatalln(err)
			}
		}()

		<-stopCtx.Done()
		logr.DefaultLogger.Infof("shutting down")
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutCtx)
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
