import React from 'react';
import { X, Server, ShieldCheck, FolderKanban, CheckSquare, Network } from 'lucide-react';

export default function MicroservicesModal({ isOpen, onClose, serviceHealth }) {
  if (!isOpen) return null;

  const services = [
    {
      name: 'API Gateway',
      port: '8000',
      description: 'Single entry point, handles CORS, reverse proxy & JWT verification',
      routes: 'Routing to all internal microservices',
      icon: <Network size={18} color="#818cf8" />,
      online: serviceHealth.gateway,
    },
    {
      name: 'User & Auth Service',
      port: '8001',
      description: 'Manages user registration, login, JWT issuance, and team directories',
      routes: '/api/auth/*, /api/users/*',
      icon: <ShieldCheck size={18} color="#38bdf8" />,
      online: serviceHealth.user,
    },
    {
      name: 'Project Service',
      port: '8002',
      description: 'Workspaces, project metadata, and custom workflow column schemes',
      routes: '/api/projects/*',
      icon: <FolderKanban size={18} color="#a855f7" />,
      online: serviceHealth.project,
    },
    {
      name: 'Task & Board Service',
      port: '8003',
      description: 'High-speed Kanban operations, task CRUD, and drag-and-drop state',
      routes: '/api/tasks/*',
      icon: <CheckSquare size={18} color="#10b981" />,
      online: serviceHealth.task,
    },
  ];

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal-card" style={{ width: '600px' }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <Server size={20} color="#6366f1" />
            <h2>Microservices Architecture Status</h2>
          </div>
          <button className="modal-close" onClick={onClose}>
            <X size={18} />
          </button>
        </div>

        <div className="modal-body">
          <p style={{ fontSize: '0.85rem', color: 'var(--text-muted)' }}>
            Your backend is distributed across 4 independent Go microservices communicating via an API Gateway.
          </p>

          <div className="services-grid">
            {services.map((s) => (
              <div key={s.port} className="service-card">
                <div className="service-card-info">
                  <div className="service-name">
                    {s.icon}
                    <span>{s.name}</span>
                    <span className="service-port">:{s.port}</span>
                  </div>
                  <div style={{ fontSize: '0.82rem', color: 'var(--text-muted)' }}>{s.description}</div>
                  <div className="service-routes">Endpoints: {s.routes}</div>
                </div>
                <div>
                  <span className={`status-badge ${s.online ? 'online' : 'standby'}`}>
                    {s.online ? '● ONLINE' : '○ STANDBY'}
                  </span>
                </div>
              </div>
            ))}
          </div>
        </div>

        <div className="modal-footer">
          <button className="btn-primary" onClick={onClose}>
            Close Monitor
          </button>
        </div>
      </div>
    </div>
  );
}
