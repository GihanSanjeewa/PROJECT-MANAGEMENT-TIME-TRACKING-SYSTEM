const API_BASE = "http://localhost:8000/api";

export async function fetchHealth() {
  try {
    const res = await fetch("http://localhost:8000/health");
    if (!res.ok) throw new Error("Gateway offline");
    return await res.json();
  } catch (err) {
    return { status: "offline", error: err.message };
  }
}

export async function checkServiceHealth(port) {
  try {
    const res = await fetch(`http://localhost:${port}/health`);
    if (!res.ok) return false;
    const data = await res.json();
    return data.status === "healthy";
  } catch {
    return false;
  }
}

export async function fetchProjects() {
  try {
    const res = await fetch(`${API_BASE}/projects`);
    if (!res.ok) throw new Error("Failed to fetch projects");
    return await res.json();
  } catch (err) {
    console.warn("Backend not reachable, using fallback projects:", err);
    return [
      {
        id: "proj-1",
        key: "CORE",
        name: "SaaS Core Platform",
        description: "High-performance project management & team workflow system",
        category: "Engineering",
        lead: "Alex Morgan",
        task_count: 5,
      },
      {
        id: "proj-2",
        key: "MOB",
        name: "Mobile Application",
        description: "React Native mobile client for iOS and Android",
        category: "Mobile",
        lead: "Maya Patel",
        task_count: 3,
      },
    ];
  }
}

export async function fetchColumns() {
  try {
    const res = await fetch(`${API_BASE}/projects/columns`);
    if (!res.ok) throw new Error("Failed to fetch columns");
    return await res.json();
  } catch (err) {
    return [
      { id: "col-backlog", title: "Backlog", color: "#94a3b8", order: 1 },
      { id: "col-todo", title: "To Do", color: "#60a5fa", order: 2 },
      { id: "col-inprogress", title: "In Progress", color: "#f59e0b", order: 3 },
      { id: "col-review", title: "In Review", color: "#a855f7", order: 4 },
      { id: "col-done", title: "Done", color: "#10b981", order: 5 },
    ];
  }
}

export async function fetchTasks(projectId = "proj-1") {
  try {
    const res = await fetch(`${API_BASE}/tasks?project_id=${projectId}`);
    if (!res.ok) throw new Error("Failed to fetch tasks");
    return await res.json();
  } catch (err) {
    console.warn("Backend not reachable, using fallback tasks:", err);
    return [
      {
        id: "task-1",
        project_id: "proj-1",
        column_id: "col-todo",
        title: "Set up Go Microservices Architecture",
        description: "Configure API Gateway, User Service, Project Service, and Task Service with routing.",
        priority: "High",
        assignee: "Jordan Lee",
        due_date: "Oct 24",
      },
      {
        id: "task-2",
        project_id: "proj-1",
        column_id: "col-inprogress",
        title: "Design React Kanban Board UI",
        description: "Implement sleek drag-and-drop columns, task priority badges, and quick creation modal.",
        priority: "Urgent",
        assignee: "Maya Patel",
        due_date: "Oct 22",
      },
      {
        id: "task-3",
        project_id: "proj-1",
        column_id: "col-review",
        title: "Implement JWT Authentication Middleware",
        description: "Verify tokens on the API Gateway before forwarding requests to internal services.",
        priority: "Medium",
        assignee: "Alex Morgan",
        due_date: "Oct 25",
      },
      {
        id: "task-4",
        project_id: "proj-1",
        column_id: "col-done",
        title: "Initialize Workspace & Git Repository",
        description: "Set up folder layout, Frontend and Backend structures.",
        priority: "Low",
        assignee: "Taylor Kim",
        due_date: "Oct 18",
      },
    ];
  }
}

export async function createTask(task) {
  try {
    const res = await fetch(`${API_BASE}/tasks`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(task),
    });
    if (!res.ok) throw new Error("Failed to create task");
    return await res.json();
  } catch (err) {
    console.warn("Using local task creation fallback");
    return {
      ...task,
      id: "task-" + Date.now(),
      created_at: new Date().toISOString(),
    };
  }
}

export async function moveTask(taskId, columnId) {
  try {
    const res = await fetch(`${API_BASE}/tasks/${taskId}/move`, {
      method: "PATCH",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ column_id: columnId }),
    });
    if (!res.ok) throw new Error("Failed to move task");
    return await res.json();
  } catch (err) {
    console.warn("Using local move task fallback");
    return { id: taskId, column_id: columnId };
  }
}

export async function deleteTask(taskId) {
  try {
    const res = await fetch(`${API_BASE}/tasks/${taskId}`, {
      method: "DELETE",
    });
    return res.ok;
  } catch (err) {
    return true;
  }
}

export async function fetchUsers() {
  try {
    const res = await fetch(`${API_BASE}/users`);
    if (!res.ok) throw new Error("Failed to fetch users");
    return await res.json();
  } catch (err) {
    return [
      { id: "usr-1", name: "Alex Morgan", role: "Project Lead", avatar: "https://api.dicebear.com/7.x/avataaars/svg?seed=Alex" },
      { id: "usr-2", name: "Maya Patel", role: "Frontend Lead", avatar: "https://api.dicebear.com/7.x/avataaars/svg?seed=Maya" },
      { id: "usr-3", name: "Jordan Lee", role: "Go Backend Dev", avatar: "https://api.dicebear.com/7.x/avataaars/svg?seed=Jordan" },
      { id: "usr-4", name: "Taylor Kim", role: "Product Designer", avatar: "https://api.dicebear.com/7.x/avataaars/svg?seed=Taylor" },
    ];
  }
}
