import React from 'react';
import { Layers, Plus, Search, LogOut } from 'lucide-react';

export default function Navbar({
  projects,
  currentProjectId,
  onSelectProject,
  searchTerm,
  onSearchChange,
  onOpenCreateModal,
  onOpenServicesModal,
  isGatewayOnline,
  currentUser,
  onLogout,
}) {
  const roleColors = {
    PROJECT_MANAGER: { bg: 'rgba(168, 85, 247, 0.15)', text: '#c084fc', border: 'rgba(168, 85, 247, 0.3)' },
    DEVELOPER: { bg: 'rgba(56, 189, 248, 0.15)', text: '#38bdf8', border: 'rgba(56, 189, 248, 0.3)' },
    QA_ENGINEER: { bg: 'rgba(16, 185, 129, 0.15)', text: '#34d399', border: 'rgba(16, 185, 129, 0.3)' },
    BUSINESS_ANALYST: { bg: 'rgba(245, 158, 11, 0.15)', text: '#fbbf24', border: 'rgba(245, 158, 11, 0.3)' },
    ADMIN: { bg: 'rgba(239, 68, 68, 0.15)', text: '#f87171', border: 'rgba(239, 68, 68, 0.3)' },
  };

  const userRole = currentUser?.primary_role || 'PROJECT_MANAGER';
  const roleStyle = roleColors[userRole] || roleColors.DEVELOPER;

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
              {p.code || p.key} - {p.name}
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
          title="Click to view Backend & Database Status"
        >
          <div className="pulse-dot" style={{ backgroundColor: isGatewayOnline ? '#10b981' : '#eab308' }} />
          <span>API :8080</span>
        </div>

        <button className="btn-primary" onClick={() => onOpenCreateModal('col-todo')}>
          <Plus size={16} />
          <span>New Task</span>
        </button>

        {currentUser && (
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginLeft: '6px' }}>
            <div style={{ display: 'flex', flexDirection: 'column', alignItems: 'flex-end', lineHeight: '1.2' }}>
              <span style={{ fontSize: '0.85rem', fontWeight: '700', color: 'var(--text-main)' }}>
                {currentUser.full_name || currentUser.email}
              </span>
              <span
                style={{
                  fontSize: '0.65rem',
                  fontWeight: '800',
                  padding: '1px 6px',
                  borderRadius: '4px',
                  backgroundColor: roleStyle.bg,
                  color: roleStyle.text,
                  border: `1px solid ${roleStyle.border}`,
                  marginTop: '2px',
                }}
              >
                {userRole}
              </span>
            </div>

            <button
              onClick={onLogout}
              className="task-action-btn delete"
              title="Sign Out"
              style={{ width: '32px', height: '32px', borderRadius: '8px' }}
            >
              <LogOut size={15} />
            </button>
          </div>
        )}
      </div>
    </nav>
  );
}
