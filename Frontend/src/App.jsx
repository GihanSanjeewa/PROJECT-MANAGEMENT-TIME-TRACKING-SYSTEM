import React, { useState, useEffect } from 'react';
import Navbar from './components/Navbar';
import KanbanBoard from './components/KanbanBoard';
import CreateTaskModal from './components/CreateTaskModal';
import MicroservicesModal from './components/MicroservicesModal';
import LoginPage from './components/LoginPage';
import ProjectTimelineBanner from './components/ProjectTimelineBanner';
import TeamRosterModal from './components/TeamRosterModal';
import AssignMemberModal from './components/AssignMemberModal';
import ProjectSettingsModal from './components/ProjectSettingsModal';
import CreateProjectModal from './components/CreateProjectModal';
import {
  fetchProjects,
  createProject,
  updateProject,
  fetchProjectMembers,
  assignProjectMember,
  removeProjectMember,
  fetchProjectCapacity,
  fetchColumns,
  fetchTasks,
  fetchUsers,
  createTask,
  moveTask,
  deleteTask,
  checkServiceHealth,
  getStoredUser,
  logout,
} from './services/api';

export default function App() {
  const [currentUser, setCurrentUser] = useState(() => getStoredUser());
  const [projects, setProjects] = useState([]);
  const [selectedProjectId, setSelectedProjectId] = useState('proj-1');
  const [projectMembers, setProjectMembers] = useState([]);
  const [projectCapacity, setProjectCapacity] = useState(null);
  const [columns, setColumns] = useState([]);
  const [tasks, setTasks] = useState([]);
  const [users, setUsers] = useState([]);
  const [searchTerm, setSearchTerm] = useState('');

  // Modals state
  const [isCreateModalOpen, setIsCreateModalOpen] = useState(false);
  const [defaultColumnId, setDefaultColumnId] = useState('col-todo');
  const [isServicesModalOpen, setIsServicesModalOpen] = useState(false);
  const [isRosterModalOpen, setIsRosterModalOpen] = useState(false);
  const [isAssignMemberModalOpen, setIsAssignMemberModalOpen] = useState(false);
  const [isSettingsModalOpen, setIsSettingsModalOpen] = useState(false);
  const [isCreateProjectModalOpen, setIsCreateProjectModalOpen] = useState(false);

  // Microservices & Backend live health check
  const [serviceHealth, setServiceHealth] = useState({
    gateway: false,
    user: false,
    project: false,
    task: false,
  });

  // Initial Data Load (when authenticated)
  useEffect(() => {
    if (!currentUser) return;

    async function init() {
      const [projData, colData, userData] = await Promise.all([
        fetchProjects(),
        fetchColumns(),
        fetchUsers(),
      ]);

      setProjects(projData);
      setColumns(colData);
      setUsers(userData);

      const activeProjId = projData.length > 0 ? projData[0].id : 'proj-1';
      setSelectedProjectId(activeProjId);
      loadProjectDetails(activeProjId);
    }
    init();
  }, [currentUser]);

  // Load project-specific data: tasks, members, and capacity
  const loadProjectDetails = async (projectId) => {
    const [taskData, memberData, capData] = await Promise.all([
      fetchTasks(projectId),
      fetchProjectMembers(projectId),
      fetchProjectCapacity(projectId),
    ]);
    setTasks(taskData);
    setProjectMembers(memberData);
    setProjectCapacity(capData);
  };

  // Poll backend health
  useEffect(() => {
    async function pingServices() {
      const [api8080, gw, usr, prj] = await Promise.all([
        checkServiceHealth(8080),
        checkServiceHealth(8000),
        checkServiceHealth(8001),
        checkServiceHealth(8002),
      ]);
      setServiceHealth({ gateway: api8080 || gw, user: usr, project: prj, task: api8080 });
    }
    pingServices();
    const interval = setInterval(pingServices, 5000);
    return () => clearInterval(interval);
  }, []);

  // Change active project
  const handleSelectProject = async (id) => {
    setSelectedProjectId(id);
    await loadProjectDetails(id);
  };

  // Create Project
  const handleCreateProject = async (projectData) => {
    const created = await createProject(projectData);
    const updatedProjects = await fetchProjects();
    setProjects(updatedProjects);
    if (created && created.id) {
      setSelectedProjectId(created.id);
      await loadProjectDetails(created.id);
    }
  };

  // Update Project (Timeline Guard)
  const handleUpdateProject = async (projectId, updateData) => {
    const updated = await updateProject(projectId, updateData);
    setProjects((prev) => prev.map((p) => (p.id === projectId ? { ...p, ...updated } : p)));
    const capData = await fetchProjectCapacity(projectId);
    setProjectCapacity(capData);
  };

  // Assign Team Member
  const handleAssignMember = async (memberData) => {
    await assignProjectMember(selectedProjectId, memberData);
    const [members, cap] = await Promise.all([
      fetchProjectMembers(selectedProjectId),
      fetchProjectCapacity(selectedProjectId),
    ]);
    setProjectMembers(members);
    setProjectCapacity(cap);
  };

  // Remove Team Member
  const handleRemoveMember = async (userId) => {
    await removeProjectMember(selectedProjectId, userId);
    const [members, cap] = await Promise.all([
      fetchProjectMembers(selectedProjectId),
      fetchProjectCapacity(selectedProjectId),
    ]);
    setProjectMembers(members);
    setProjectCapacity(cap);
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

  const handleLogout = async () => {
    await logout();
    setCurrentUser(null);
  };

  // If user is not authenticated, render LoginPage
  if (!currentUser) {
    return <LoginPage onLoginSuccess={(user) => setCurrentUser(user)} />;
  }

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
      id: selectedProjectId || 'proj-1',
      name: 'PulseFlow SaaS Core',
      code: 'CORE',
      key: 'CORE',
      description: 'Distributed Agile Project Management System powered by Go Clean Architecture & React',
      start_date: '2026-11-01',
      target_end_date: '2026-12-31',
      budget_hours: 600,
      status: 'ACTIVE',
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
        currentUser={currentUser}
        onLogout={handleLogout}
      />

      <ProjectTimelineBanner
        project={currentProject}
        capacity={projectCapacity}
        onOpenRoster={() => setIsRosterModalOpen(true)}
        onOpenSettings={() => setIsSettingsModalOpen(true)}
        onOpenCreateProject={() => setIsCreateProjectModalOpen(true)}
        currentUser={currentUser}
      />

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

      <TeamRosterModal
        isOpen={isRosterModalOpen}
        onClose={() => setIsRosterModalOpen(false)}
        members={projectMembers}
        users={users}
        capacity={projectCapacity}
        onOpenAssignModal={() => {
          setIsRosterModalOpen(false);
          setIsAssignMemberModalOpen(true);
        }}
        onRemoveMember={handleRemoveMember}
        currentUser={currentUser}
      />

      <AssignMemberModal
        isOpen={isAssignMemberModalOpen}
        onClose={() => {
          setIsAssignMemberModalOpen(false);
          setIsRosterModalOpen(true);
        }}
        users={users}
        onAssign={handleAssignMember}
      />

      <ProjectSettingsModal
        key={currentProject.id}
        isOpen={isSettingsModalOpen}
        onClose={() => setIsSettingsModalOpen(false)}
        project={currentProject}
        onUpdate={handleUpdateProject}
      />

      <CreateProjectModal
        isOpen={isCreateProjectModalOpen}
        onClose={() => setIsCreateProjectModalOpen(false)}
        onCreate={handleCreateProject}
      />
    </div>
  );
}
