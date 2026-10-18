# PulseFlow - Microservices Project Management System

PulseFlow is a full-stack, distributed project management system built with **React** (Frontend SPA) and **Go (Golang)** microservices.

---

## 🏛 Architecture Overview

PulseFlow follows a **Decoupled Microservices Architecture** with an **API Gateway** acting as the single reverse proxy for the React client.

```
                          ┌───────────────────────┐
                          │   React Frontend      │
                          │ http://localhost:5173 │
                          └───────────┬───────────┘
                                      │ REST (HTTP/JSON)
                                      ▼
                          ┌───────────────────────┐
                          │     API Gateway       │
                          │ http://localhost:8000 │
                          └───┬───────┬───────┬───┘
                              │       │       │
              ┌───────────────┘       │       └───────────────┐
              ▼                       ▼                       ▼
   ┌────────────────────┐  ┌────────────────────┐  ┌────────────────────┐
   │ User & Auth Service│  │  Project Service   │  │Task & Board Service│
   │    Port :8001      │  │    Port :8002      │  │    Port :8003      │
   │  /api/auth, /users │  │    /api/projects   │  │    /api/tasks      │
   └────────────────────┘  └────────────────────┘  └────────────────────┘
```

---

## 📁 Directory Structure

```
Project Management Tool/
├── start-services.ps1               # One-click launcher for all microservices & frontend
├── Backend/
│   ├── api-gateway/                 # Go API Gateway (:8000)
│   │   ├── main.go                  # Reverse proxy with CORS & route dispatching
│   │   └── go.mod
│   │
│   └── services/
│       ├── user-service/            # Auth & Users (:8001)
│       │   ├── cmd/main.go
│       │   └── go.mod
│       │
│       ├── project-service/         # Workspaces, Projects & Columns (:8002)
│       │   ├── cmd/main.go
│       │   └── go.mod
│       │
│       └── task-service/            # High-speed Kanban Board & Tasks (:8003)
│           ├── cmd/main.go
│           └── go.mod
│
└── Frontend/                        # React SPA (Vite)
    ├── src/
    │   ├── components/
    │   │   ├── KanbanBoard.jsx      # Multi-column Kanban board
    │   │   ├── TaskCard.jsx         # Card with priority, tags & move actions
    │   │   ├── Navbar.jsx           # Global navigation & service health monitor
    │   │   ├── CreateTaskModal.jsx  # Task creation form
    │   │   └── MicroservicesModal.jsx # Live architecture inspector
    │   ├── services/
    │   │   └── api.js               # Centralized API service talking to :8000
    │   ├── App.jsx                  # Main application state & layout
    │   └── index.css                # Dark-mode design system & animations
    └── package.json
```

---

## 🔌 Microservices & Port Reference

| Service | Port | Endpoint Prefix | Purpose |
| :--- | :--- | :--- | :--- |
| **API Gateway** | `8000` | `/health`, `/*` | Single ingress point, reverse proxy routing, centralized CORS |
| **User & Auth Service** | `8001` | `/api/auth`, `/api/users` | User authentication, JWT tokens, team members |
| **Project Service** | `8002` | `/api/projects` | Workspaces, project settings, column definitions |
| **Task & Board Service** | `8003` | `/api/tasks` | Task creation, Kanban status updates, move/reorder, deletion |
| **React Frontend** | `5173` | `/` | Responsive SPA with drag-and-drop & live optimistic updates |

---

## 🚀 Quickstart Guide

### 1. Launch Everything with One Script
In PowerShell from the project root:
```powershell
.\start-services.ps1
```
This script automatically starts:
- User Service (:8001)
- Project Service (:8002)
- Task Service (:8003)
- API Gateway (:8000)
- Vite Dev Server (:5173)

### 2. Manual Start (Optional)

#### Run API Gateway:
```powershell
cd Backend\api-gateway
go run main.go
```

#### Run Services:
```powershell
cd Backend\services\user-service
go run cmd/main.go

cd Backend\services\project-service
go run cmd/main.go

cd Backend\services\task-service
go run cmd/main.go
```

#### Run Frontend:
```powershell
cd Frontend
npm run dev
```
Open **`http://localhost:5173`** in your browser!
