package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/Modificator/readlater-wip/internal/model"
	"github.com/Modificator/readlater-wip/internal/service"
	"github.com/Modificator/readlater-wip/internal/store"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	archiveSvc *service.ArchiveService
	ruleSvc    *service.RuleService
	giteaSvc   *service.GiteaService
}

func NewHandler(archiveSvc *service.ArchiveService, ruleSvc *service.RuleService, giteaSvc *service.GiteaService) *Handler {
	return &Handler{archiveSvc: archiveSvc, ruleSvc: ruleSvc, giteaSvc: giteaSvc}
}

func (h *Handler) RegisterRoutes(e *echo.Echo) {
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]any{"status": "ok", "time": time.Now().UTC()})
	})

	v1 := e.Group("/api/v1")
	v1.POST("/tasks", h.createTask)
	v1.GET("/tasks/:id", h.getTask)
	v1.POST("/tasks/:id/retry", h.retryTask)

	v1.POST("/rules", h.createRule)
	v1.GET("/rules", h.listRules)
	v1.POST("/rules/test-match", h.testMatchRule)

	v1.POST("/gitea-targets", h.createGiteaTarget)
	v1.GET("/gitea-targets", h.listGiteaTargets)
}

func (h *Handler) createTask(c echo.Context) error {
	var in service.CreateTaskInput
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid request body"})
	}
	if strings.TrimSpace(in.URL) == "" {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "url is required"})
	}
	if !in.SaveWebContent && !in.CloneToGitea {
		in.SaveWebContent = true
	}
	task, err := h.archiveSvc.CreateTask(in)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusAccepted, map[string]any{"task_id": task.ID, "status": task.Status, "task": task})
}

func (h *Handler) getTask(c echo.Context) error {
	task, err := h.archiveSvc.GetTask(c.Param("id"))
	if err != nil {
		if err == store.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]any{"error": "task not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, task)
}

func (h *Handler) retryTask(c echo.Context) error {
	task, err := h.archiveSvc.RetryTask(c.Param("id"))
	if err != nil {
		if err == store.ErrNotFound {
			return c.JSON(http.StatusNotFound, map[string]any{"error": "task not found"})
		}
		return c.JSON(http.StatusInternalServerError, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusAccepted, task)
}

func (h *Handler) createRule(c echo.Context) error {
	var in model.ExtractionRule
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid request body"})
	}
	if in.ID == "" {
		in.ID = "rule_" + time.Now().UTC().Format("20060102150405.000000000")
	}
	if err := h.ruleSvc.CreateRule(&in); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, in)
}

func (h *Handler) listRules(c echo.Context) error {
	return c.JSON(http.StatusOK, h.ruleSvc.ListRules())
}

func (h *Handler) testMatchRule(c echo.Context) error {
	var in struct {
		URL    string `json:"url"`
		RuleID string `json:"rule_id"`
	}
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid request body"})
	}
	rule, err := h.ruleSvc.MatchRule(in.URL, in.RuleID)
	if err != nil {
		if err == store.ErrNotFound {
			return c.JSON(http.StatusOK, map[string]any{"matched": false})
		}
		return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusOK, map[string]any{"matched": true, "rule": rule})
}

func (h *Handler) createGiteaTarget(c echo.Context) error {
	var in model.GiteaTarget
	if err := c.Bind(&in); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": "invalid request body"})
	}
	if in.ID == "" {
		in.ID = "gitea_" + time.Now().UTC().Format("20060102150405.000000000")
	}
	if err := h.giteaSvc.CreateTarget(&in); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]any{"error": err.Error()})
	}
	return c.JSON(http.StatusCreated, in)
}

func (h *Handler) listGiteaTargets(c echo.Context) error {
	return c.JSON(http.StatusOK, h.giteaSvc.ListTargets())
}
