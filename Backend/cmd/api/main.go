package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"project-management-system/internal/config"
	"project-management-system/internal/domain"
	"project-management-system/internal/handler"
	"project-management-system/internal/repository"
	"project-management-system/internal/service"
	"project-management-system/pkg/middleware"
	"project-management-system/pkg/response"
)

func main() {
	// Initialize structured JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load and strictly validate environment configuration
	cfg, err := config.Load()
	if err != nil {
		slog.Error("CRITICAL: Failed to load configuration", "error", err)
		os.Exit(1)
	}

	slog.Info("Configuration loaded successfully",
		"port", cfg.Port,
		"env", cfg.Environment,
		"work_hours_per_day", cfg.WorkHoursPerDay,
	)

	// Repositories
	userRepo := repository.NewInMemoryUserRepository()
	tokenStore := repository.NewInMemoryTokenStore()
	projectRepo := repository.NewInMemoryProjectRepository()

	// Services
	calendarServ := service.NewCalendarService()
	authService := service.NewAuthService(userRepo, tokenStore, cfg.JWTSecret)
	userService := service.NewUserService(userRepo, authService)
	projectService := service.NewProjectService(projectRepo, userRepo, calendarServ)

	// Seed initial data (Admins, Team Members, and Initial Project)
	seedInitialData(context.Background(), userService, projectService)

	// Handlers
	authHandler := handler.NewAuthHandler(authService, userService)
	userHandler := handler.NewUserHandler(userService)
	projectHandler := handler.NewProjectHandler(projectService)

	mux := http.NewServeMux()

	// Public Health check & version
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		response.JSON(w, http.StatusOK, map[string]interface{}{
			"status":      "HEALTHY",
			"environment": cfg.Environment,
			"timestamp":   time.Now().UTC(),
		}, reqID)
	})

	mux.HandleFunc("/api/v1/version", func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		response.JSON(w, http.StatusOK, map[string]string{
			"version": "1.0.0",
			"name":    "PulseFlow Project Management & Feasibility Engine",
		}, reqID)
	})

	// Public Auth endpoints
	mux.HandleFunc("/api/v1/auth/login", authHandler.Login)
	mux.HandleFunc("/api/v1/auth/refresh", authHandler.Refresh)
	mux.HandleFunc("/api/v1/auth/logout", authHandler.Logout)

	// Protected routes (require Bearer JWT authentication)
	authMiddleware := middleware.Authenticate(authService)

	// /me profile
	mux.Handle("/api/v1/auth/me", authMiddleware(http.HandlerFunc(authHandler.Me)))

	// /users collection & member management
	mux.Handle("/api/v1/users", authMiddleware(http.HandlerFunc(userHandler.HandleUsers)))
	mux.Handle("/api/v1/users/", authMiddleware(http.HandlerFunc(userHandler.HandleUserByID)))

	// /projects collection, timeline & capacity
	mux.Handle("/api/v1/projects", authMiddleware(http.HandlerFunc(projectHandler.HandleProjects)))
	mux.Handle("/api/v1/projects/", authMiddleware(http.HandlerFunc(projectHandler.HandleProjectByID)))

	// Assemble global middleware: CORS -> RequestID -> Logger -> Recovery
	rootHandler := middleware.CORS(
		middleware.RequestID(
			middleware.Logger(
				middleware.Recovery(mux),
			),
		),
	)

	server := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      rootHandler,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Server context for graceful shutdown
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig
		shutdownCtx, shutdownCancel := context.WithTimeout(serverCtx, 30*time.Second)
		defer shutdownCancel()

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				slog.Error("Graceful shutdown timed out... forcing exit.")
				os.Exit(1)
			}
		}()

		slog.Info("Shutting down HTTP server gracefully...")
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("Server shutdown error", "error", err)
		}
		serverStopCtx()
	}()

	slog.Info(fmt.Sprintf("🚀 PulseFlow API Server started on http://localhost:%s", cfg.Port))
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Server closed with error", "error", err)
	}

	<-serverCtx.Done()
	slog.Info("Server stopped cleanly.")
}

func seedInitialData(ctx context.Context, userService *service.UserService, projectService *service.ProjectService) {
	seeds := []service.CreateUserDTO{
		{
			Email:             "alex.pm@pulseflow.io",
			Password:          "SecurePass123!",
			FullName:          "Alex Morgan",
			PrimaryRole:       domain.RoleProjectManager,
			DefaultDailyHours: 8.0,
		},
		{
			Email:             "jordan.dev@pulseflow.io",
			Password:          "SecurePass123!",
			FullName:          "Jordan Lee",
			PrimaryRole:       domain.RoleDeveloper,
			DefaultDailyHours: 8.0,
		},
		{
			Email:             "maya.qa@pulseflow.io",
			Password:          "SecurePass123!",
			FullName:          "Maya Patel",
			PrimaryRole:       domain.RoleQAEngineer,
			DefaultDailyHours: 8.0,
		},
		{
			Email:             "taylor.ba@pulseflow.io",
			Password:          "SecurePass123!",
			FullName:          "Taylor Kim",
			PrimaryRole:       domain.RoleBusinessAnalyst,
			DefaultDailyHours: 8.0,
		},
	}

	createdUsers := make(map[string]*domain.User)
	for _, s := range seeds {
		u, err := userService.Create(ctx, s)
		if err == nil {
			createdUsers[s.Email] = u
		}
	}

	// Seed Initial Project
	now := time.Now().UTC()
	start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 2, 0) // 2 months timeline

	proj, err := projectService.Create(ctx, service.CreateProjectDTO{
		Code:          "CORE",
		Name:          "SaaS Core Platform",
		Description:   "High-performance distributed project management & feasibility engine",
		StartDate:     start,
		TargetEndDate: end,
		BudgetHours:   600.0,
		CreatedBy:     createdUsers["alex.pm@pulseflow.io"].ID,
	})

	if err == nil && proj != nil {
		// Assign team members to project
		if dev, ok := createdUsers["jordan.dev@pulseflow.io"]; ok {
			_, _ = projectService.AddMember(ctx, service.AssignMemberDTO{
				ProjectID:           proj.ID,
				UserID:              dev.ID,
				ProjectRole:         domain.ProjectRoleDeveloper,
				AllocationPct:       100,
				EffectiveDailyHours: 8.0,
				JoinedDate:          start,
			})
		}
		if qa, ok := createdUsers["maya.qa@pulseflow.io"]; ok {
			_, _ = projectService.AddMember(ctx, service.AssignMemberDTO{
				ProjectID:           proj.ID,
				UserID:              qa.ID,
				ProjectRole:         domain.ProjectRoleQA,
				AllocationPct:       100,
				EffectiveDailyHours: 8.0,
				JoinedDate:          start,
			})
		}
		if ba, ok := createdUsers["taylor.ba@pulseflow.io"]; ok {
			_, _ = projectService.AddMember(ctx, service.AssignMemberDTO{
				ProjectID:           proj.ID,
				UserID:              ba.ID,
				ProjectRole:         domain.ProjectRoleBA,
				AllocationPct:       50, // Part-time 50% allocation
				EffectiveDailyHours: 4.0,
				JoinedDate:          start,
			})
		}
		if pm, ok := createdUsers["alex.pm@pulseflow.io"]; ok {
			_, _ = projectService.AddMember(ctx, service.AssignMemberDTO{
				ProjectID:           proj.ID,
				UserID:              pm.ID,
				ProjectRole:         domain.ProjectRolePM,
				AllocationPct:       100,
				EffectiveDailyHours: 8.0,
				JoinedDate:          start,
			})
		}
	}
}
