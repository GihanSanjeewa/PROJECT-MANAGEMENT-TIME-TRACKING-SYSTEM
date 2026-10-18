import React from 'react';
import { Layers, Plus, Search, Server, Activity } from 'lucide-react';

export default function Navbar({
  projects,
  currentProjectId,
  onSelectProject,
  searchTerm,
  onSearchChange,
  onOpenCreateModal,
  onOpenServicesModal,
  isGatewayOnline,
}) {
  return (
    <nav className="navbar">
      <div className="nav-left">
        <div className="brand-badge">
          <div className="logo-icon">
            <Layers size={18} color="#fff" />
          </div>
          <span>PulseFlow</span>
        </div>

        <select
          className="project-select"
          value={currentProjectId}
          onChange={(e) => onSelectProject(e.target.value)}
        >
          {projects.map((p) => (
            <option key={p.id} value={p.id}>
              {p.key} - {p.name}
            </option>
          ))}
        </select>
      </div>

      <div className="nav-center">
        <div className="search-box">
          <Search size={15} />
          <input
            type="text"
            placeholder="Search tasks, assignees, keywords..."
            value={searchTerm}
            onChange={(e) => onSearchChange(e.target.value)}
          />
        </div>
      </div>

      <div className="nav-right">
        <div
          className="services-indicator"
          onClick={onOpenServicesModal}
          title="Click to view Go Microservices Architecture & Port mappings"
        >
          <div className="pulse-dot" style={{ backgroundColor: isGatewayOnline ? '#10b981' : '#eab308' }} />
          <span>Microservices :8000</span>
        </div>

        <button className="btn-primary" onClick={() => onOpenCreateModal('col-todo')}>
          <Plus size={16} />
          <span>New Task</span>
        </button>

        <img
          src="https://api.dicebear.com/7.x/avataaars/svg?seed=Alex"
          alt="Current User"
          className="user-avatar-sm"
          title="Alex Morgan (Project Lead)"
        />
      </div>
    </nav>
  );
}
