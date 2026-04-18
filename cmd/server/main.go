package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Modificator/readlater-wip/internal/api"
	"github.com/Modificator/readlater-wip/internal/service"
	"github.com/Modificator/readlater-wip/internal/store"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	storageRoot := envOrDefault("ARCHIVE_STORAGE_ROOT", "./archive")
	addr := envOrDefault("LISTEN_ADDR", ":8080")

	repo := store.NewMemoryRepository()
	ruleSvc := service.NewRuleService(repo)
	giteaSvc := service.NewGiteaService(repo)
	archiveSvc := service.NewArchiveService(repo, ruleSvc, giteaSvc, storageRoot)
	archiveSvc.StartWorkers(2)

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.Logger())

	h := api.NewHandler(archiveSvc, ruleSvc, giteaSvc)
	h.RegisterRoutes(e)

	go func() {
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("start server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = e.Shutdown(ctx)
	archiveSvc.StopWorkers()
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
