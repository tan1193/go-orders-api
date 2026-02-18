package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	_ "github.com/lib/pq"

	"go-order-service/internal/handler"
	"go-order-service/internal/repository"
	"go-order-service/internal/service"
	"go-order-service/internal/worker"
)

func main() {
	logger := log.New(os.Stdout, "", log.LstdFlags)

	port := getenv("HTTP_PORT", "8080")
	dsn := getenv("POSTGRES_DSN", "postgres://postgres:postgres@localhost:5432/orders_db?sslmode=disable")

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		logger.Fatalf("msg=db_open_failed err=%q", err)
	}
	defer db.Close()

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := db.PingContext(pingCtx); err != nil {
		logger.Fatalf("msg=db_ping_failed err=%q", err)
	}

	repo := repository.NewPostgresOrderRepository(db, 2*time.Second)
	queue := make(chan string, 100)
	svc := service.NewOrderService(repo, worker.ChannelEnqueuer{Ch: queue})

	ordWorker := worker.NewOrderWorker(queue, svc, logger)
	workerCtx, cancelWorker := context.WithCancel(context.Background())
	var workerWG sync.WaitGroup
	workerWG.Add(1)
	go func() {
		defer workerWG.Done()
		ordWorker.Run(workerCtx)
	}()

	h := handler.NewOrderHandler(svc, logger)
	mux := http.NewServeMux()
	mux.HandleFunc("/orders", h.Orders)
	mux.HandleFunc("/orders/", h.OrderByID)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", port),
		Handler:           requestLogger(logger, mux),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Printf("msg=http_server_start addr=%s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		logger.Printf("msg=shutdown_signal signal=%s", sig.String())
	case err := <-errCh:
		logger.Printf("msg=http_server_failed err=%q", err)
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Printf("msg=http_shutdown_failed err=%q", err)
	}

	close(queue)
	workerDone := make(chan struct{})
	go func() {
		defer close(workerDone)
		workerWG.Wait()
	}()

	select {
	case <-workerDone:
		logger.Printf("msg=worker_drained")
	case <-time.After(5 * time.Second):
		logger.Printf("msg=worker_drain_timeout action=cancel")
		cancelWorker()
		<-workerDone
	}

	cancelWorker()
	logger.Printf("msg=shutdown_complete")
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func requestLogger(logger *log.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		logger.Printf("msg=http_request method=%s path=%s duration_ms=%d", r.Method, r.URL.Path, time.Since(start).Milliseconds())
	})
}
