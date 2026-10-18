package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"project-management-system/internal/config"
	"project-management-system/internal/domain"
	"project-management-system/internal/handler"
	"project-management-system/internal/repository"
	"project-management-system/internal/service"
	"project-management-system/pkg/errcode"
	"project-management-system/pkg/middleware"
	"project-management-system/pkg/response"
)

func main() {
	// Initialize structured slog logger
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
	taskRepo := repository.NewInMemoryTaskRepository(projectRepo)

	// Services
	calendarServ := service.NewCalendarService()
	authService := service.NewAuthService(userRepo, tokenStore, cfg.JWTSecret)
	userService := service.NewUserService(userRepo, authService)
	projectService := service.NewProjectService(projectRepo, userRepo, calendarServ)
	taskService := service.NewTaskService(taskRepo, projectRepo, userRepo, calendarServ)

	// Seed initial data (Admins, Team Members, Initial Project, and Phase 4 Tasks)
	seedInitialData(context.Background(), userService, projectService, taskService)

	// Handlers
	authHandler := handler.NewAuthHandler(authService, userService)
	userHandler := handler.NewUserHandler(userService)
	projectHandler := handler.NewProjectHandler(projectService)
	taskHandler := handler.NewTaskHandler(taskService)
	projectHandler.SetTaskHandler(taskHandler)

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

	// /projects collection, timeline, capacity, tasks, dependencies & workload
	mux.Handle("/api/v1/projects", authMiddleware(http.HandlerFunc(projectHandler.HandleProjects)))
	mux.Handle("/api/v1/projects/", authMiddleware(http.HandlerFunc(projectHandler.HandleProjectByID)))

	// /tasks collection, updates, state machine & dependencies
	mux.Handle("/api/v1/tasks/", authMiddleware(http.HandlerFunc(taskHandler.HandleTaskByID)))

	// /dependencies deletion endpoint
	mux.Handle("/api/v1/dependencies/", authMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := middleware.GetRequestID(r.Context())
		parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
		if len(parts) >= 3 && r.Method == http.MethodDelete {
			depID := parts[2]
			if err := taskService.RemoveDependency(r.Context(), depID); err != nil {
				response.Error(w, err, reqID)
				return
			}
			response.JSON(w, http.StatusOK, map[string]string{"message": "Dependency removed successfully"}, reqID)
			return
		}
		response.Error(w, errcode.New(errcode.ErrBadRequest, "Method not allowed", http.StatusMethodNotAllowed, nil), reqID)
	})))

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

func seedInitialData(
	ctx context.Context,
	userService *service.UserService,
	projectService *service.ProjectService,
	taskService *service.TaskService,
) {
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
		var devID, qaID, baID string

		// Assign team members to project
		if dev, ok := createdUsers["jordan.dev@pulseflow.io"]; ok {
			devID = dev.ID
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
			qaID = qa.ID
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
			baID = ba.ID
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

		// Seed Initial Tasks for Project (Phase 4 Tasks)
		t1Start := start.AddDate(0, 0, 1)
		t1Due := start.AddDate(0, 0, 5)
		task1, _ := taskService.CreateTask(ctx, &domain.Task{
			ID:             "task-101",
			ProjectID:      proj.ID,
			Title:          "Define OpenAPI 3.0 & Domain Models",
			Description:    "Complete Swagger specification for Auth, Projects, Tasks and Feasibility Engine",
			RequiredRole:   domain.RoleRequiredBA,
			Priority:       domain.PriorityHigh,
			Status:         domain.TaskStatusDone,
			EstimatedHours: 16.0,
			LoggedHours:    16.0,
			StartDate:      t1Start,
			DueDate:        t1Due,
			AssignedTo:     &baID,
		})

		t2Start := start.AddDate(0, 0, 6)
		t2Due := start.AddDate(0, 0, 12)
		task2, _ := taskService.CreateTask(ctx, &domain.Task{
			ID:             "task-102",
			ProjectID:      proj.ID,
			Title:          "Implement Clean Architecture & DB Migrations",
			Description:    "PostgreSQL versioned schema migrations, strict config loader and repository layers",
			RequiredRole:   domain.RoleRequiredDev,
			Priority:       domain.PriorityHigh,
			Status:         domain.TaskStatusDone,
			EstimatedHours: 24.0,
			LoggedHours:    24.0,
			StartDate:      t2Start,
			DueDate:        t2Due,
			AssignedTo:     &devID,
		})

		t3Start := start.AddDate(0, 0, 13)
		t3Due := start.AddDate(0, 0, 18)
		task3, _ := taskService.CreateTask(ctx, &domain.Task{
			ID:             "task-103",
			ProjectID:      proj.ID,
			Title:          "Auth Engine & JWT Token Security",
			Description:    "Bcrypt password hashing, 15m JWT access tokens, 7d rotating refresh tokens",
			RequiredRole:   domain.RoleRequiredDev,
			Priority:       domain.PriorityUrgent,
			Status:         domain.TaskStatusDone,
			EstimatedHours: 20.0,
			LoggedHours:    18.0,
			StartDate:      t3Start,
			DueDate:        t3Due,
			AssignedTo:     &devID,
		})

		t4Start := start.AddDate(0, 0, 19)
		t4Due := start.AddDate(0, 0, 26)
		task4, _ := taskService.CreateTask(ctx, &domain.Task{
			ID:             "task-104",
			ProjectID:      proj.ID,
			Title:          "Task Timeline Range Guard & DAG Cycle Detector",
			Description:    "Enforce task dates within project bounds, DFS cycle detector, and developer workload analyzer",
			RequiredRole:   domain.RoleRequiredDev,
			Priority:       domain.PriorityHigh,
			Status:         domain.TaskStatusInProgress,
			EstimatedHours: 24.0,
			LoggedHours:    8.0,
			StartDate:      t4Start,
			DueDate:        t4Due,
			AssignedTo:     &devID,
		})

		t5Start := start.AddDate(0, 0, 27)
		t5Due := start.AddDate(0, 1, 5)
		task5, _ := taskService.CreateTask(ctx, &domain.Task{
			ID:             "task-105",
			ProjectID:      proj.ID,
			Title:          "Quality Gate Test Suite & Zero-Bug CI",
			Description:    "Verify all invariant boundary conditions, multi-node cycles, and state machine transitions",
			RequiredRole:   domain.RoleRequiredQA,
			Priority:       domain.PriorityHigh,
			Status:         domain.TaskStatusTodo,
			EstimatedHours: 20.0,
			LoggedHours:    0.0,
			StartDate:      t5Start,
			DueDate:        t5Due,
			AssignedTo:     &qaID,
		})

		// Seed initial dependency (Task 5 depends on Task 4)
		if task4 != nil && task5 != nil {
			_, _ = taskService.AddDependency(ctx, proj.ID, task5.ID, task4.ID, "FS")
		}
		if task2 != nil && task3 != nil {
			_, _ = taskService.AddDependency(ctx, proj.ID, task3.ID, task2.ID, "FS")
		}
		if task1 != nil && task2 != nil {
			_, _ = taskService.AddDependency(ctx, proj.ID, task2.ID, task1.ID, "FS")
		}
	}
}
