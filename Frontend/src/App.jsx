import React, { useState, useEffect } from 'react';
import Navbar from './components/Navbar';
import KanbanBoard from './components/KanbanBoard';
import CreateTaskModal from './components/CreateTaskModal';
import MicroservicesModal from './components/MicroservicesModal';
import {
  fetchProjects,
  fetchColumns,
  fetchTasks,
  fetchUsers,
  createTask,
  moveTask,
  deleteTask,
  fetchHealth,
  checkServiceHealth,
} from './services/api';

export default function App() {
  const [projects, setProjects] = useState([]);
  const [selectedProjectId, setSelectedProjectId] = useState('proj-1');
  const [columns, setColumns] = useState([]);
  const [tasks, setTasks] = useState([]);
  const [users, setUsers] = useState([]);
  const [searchTerm, setSearchTerm] = useState('');

  // Modals state
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [defaultColumnId, setDefaultColumnId] = useState('col-todo');
  const [isServicesModalOpen, setIsServicesModalOpen] = useState(false);

  // Microservices live health check
  const [serviceHealth, setServiceHealth] = useState({
    gateway: false,
    user: false,
    project: false,
    task: false,
  });

  // Initial Data Load
  useEffect(() => {
    async function init() {
      const [projData, colData, taskData, userData] = await Promise.all([
        fetchProjects(),
        fetchColumns(),
        fetchTasks(selectedProjectId),
        fetchUsers(),
      ]);

      setProjects(projData);
      setColumns(colData);
      setTasks(taskData);
      setUsers(userData);
    }
    init();
  }, []);

  // Poll service health
  useEffect(() => {
    async function pingServices() {
      const [gw, usr, prj, tsk] = await Promise.all([
        checkServiceHealth(8000),
        checkServiceHealth(8001),
        checkServiceHealth(8002),
        checkServiceHealth(8003),
      ]);
      setServiceHealth({ gateway: gw, user: usr, project: prj, task: tsk });
    }
    pingServices();
    const interval = setInterval(pingServices, 5000);
    return () => clearInterval(interval);
  }, []);

  // Change active project
  const handleSelectProject = async (id) => {
    setSelectedProjectId(id);
    const updatedTasks = await fetchTasks(id);
    setTasks(updatedTasks);
  };

  // Create Task
  const handleCreateTask = async (taskData) => {
    const created = await createTask({
      ...taskData,
      project_id: selectedProjectId,
    });
    setTasks((prev) => [...prev, created]);
  };

  // Move Task
  const handleMoveTask = async (taskId, nextColumnId) => {
    setTasks((prev) =>
      prev.map((t) => (t.id === taskId ? { ...t, column_id: nextColumnId } : t))
    );
    await moveTask(taskId, nextColumnId);
  };

  // Delete Task
  const handleDeleteTask = async (taskId) => {
    setTasks((prev) => prev.filter((t) => t.id !== taskId));
    await deleteTask(taskId);
  };

  const handleOpenCreateModal = (colId = 'col-todo') => {
    setDefaultColumnId(colId);
    setIsCreateModalOpen(true);
  };

  // Filter tasks by search
  const filteredTasks = tasks.filter((t) => {
    const query = searchTerm.toLowerCase();
    return (
      t.title.toLowerCase().includes(query) ||
      (t.description && t.description.toLowerCase().includes(query)) ||
      (t.assignee && t.assignee.toLowerCase().includes(query)) ||
      (t.priority && t.priority.toLowerCase().includes(query))
    );
  });

  const currentProject =
    projects.find((p) => p.id === selectedProjectId) || {
      name: 'PulseFlow SaaS Core',
      key: 'CORE',
      description: 'Distributed Agile Project Management System powered by Go Microservices & React',
    };

  return (
    <div className="app-container">
      <Navbar
        projects={projects}
        currentProjectId={selectedProjectId}
        onSelectProject={handleSelectProject}
        searchTerm={searchTerm}
        onSearchChange={setSearchTerm}
        onOpenCreateModal={handleOpenCreateModal}
        onOpenServicesModal={() => setIsServicesModalOpen(true)}
        isGatewayOnline={serviceHealth.gateway}
      />

      <div className="project-banner">
        <div className="project-info">
          <h1>
            <span>{currentProject.name}</span>
            <span className="project-key">{currentProject.key}</span>
          </h1>
          <p className="project-desc">{currentProject.description}</p>
        </div>

        <div style={{ display: 'flex', gap: '10px' }}>
          <button className="btn-secondary" onClick={() => setIsServicesModalOpen(true)}>
            View Architecture (:8000)
          </button>
        </div>
      </div>

      <KanbanBoard
        columns={columns}
        tasks={filteredTasks}
        onMoveTask={handleMoveTask}
        onDeleteTask={handleDeleteTask}
        onOpenCreateModal={handleOpenCreateModal}
      />

      <CreateTaskModal
        isOpen={isCreateModalOpen}
        onClose={() => setIsCreateModalOpen(false)}
        columns={columns}
        users={users}
        defaultColumnId={defaultColumnId}
        onCreate={handleCreateTask}
      />

      <MicroservicesModal
        isOpen={isServicesModalOpen}
        onClose={() => setIsServicesModalOpen(false)}
        serviceHealth={serviceHealth}
      />
    </div>
  );
}
