import React from 'react';
import { X, UserPlus, Trash2, Users, Briefcase } from 'lucide-react';

export default function TeamRosterModal({
  isOpen,
  onClose,
  members,
  users,
  capacity,
  onOpenAssignModal,
  onRemoveMember,
  currentUser,
}) {
  if (!isOpen) return null;

  const isPMOrAdmin =
    currentUser?.primary_role === 'PROJECT_MANAGER' || currentUser?.primary_role === 'ADMIN';

  const roleLabels = {
    DEVELOPER: { label: 'Developers', color: '#38bdf8', icon: '💻' },
    QA: { label: 'QA Engineers', color: '#10b981', icon: '🧪' },
    BA: { label: 'Business Analysts', color: '#f59e0b', icon: '📊' },
    PM: { label: 'Project Managers', color: '#a855f7', icon: '👑' },
  };

  const grouped = {
    DEVELOPER: [],
    QA: [],
    BA: [],
    PM: [],
  };

  members.forEach((m) => {
    const roleKey = m.project_role || 'DEVELOPER';
    if (!grouped[roleKey]) grouped[roleKey] = [];
    // Match with user details if available
    const u = users.find((usr) => usr.id === m.user_id) || {
      full_name: m.user_name || 'Team Member',
      email: m.user_email || m.user_id,
    };
    grouped[roleKey].push({ ...m, userDetails: u });
  });

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal-card" style={{ width: '680px' }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <Users size={20} color="#6366f1" />
            <h2>Project Team Roster & Role Capacity</h2>
          </div>
          <button className="modal-close" onClick={onClose}>
            <X size={18} />
          </button>
        </div>

        <div className="modal-body">
          <div className="roster-capacity-summary">
            <div className="roster-summary-item">
              <span className="summary-title">Total Working Days</span>
              <span className="summary-value">{capacity?.total_working_days || 44} days</span>
            </div>
            <div className="roster-summary-item">
              <span className="summary-title">Total Team Hours</span>
              <span className="summary-value highlight">{capacity?.total_capacity_hours || 0} hrs</span>
            </div>
            <div className="roster-summary-item">
              <span className="summary-title">Assigned Members</span>
              <span className="summary-value">{members.length} people</span>
            </div>
          </div>

          <div className="roster-roles-list">
            {Object.keys(grouped).map((roleKey) => {
              const meta = roleLabels[roleKey] || { label: roleKey, color: '#fff', icon: '👤' };
              const roleMembers = grouped[roleKey];
              const roleCap = capacity?.roles?.[roleKey]?.total_capacity_hours || 0;

              return (
                <div key={roleKey} className="roster-role-group">
                  <div className="roster-role-header">
                    <span className="roster-role-title">
                      <span>{meta.icon}</span> {meta.label} ({roleMembers.length})
                    </span>
                    <span className="roster-role-hours" style={{ color: meta.color }}>
                      Capacity: {roleCap} hrs
                    </span>
                  </div>

                  {roleMembers.length === 0 ? (
                    <div className="roster-empty-notice">No members assigned to this role yet.</div>
                  ) : (
                    <div className="roster-members-grid">
                      {roleMembers.map((m) => (
                        <div key={m.id || m.user_id} className="roster-member-card">
                          <img
                            src={`https://api.dicebear.com/7.x/avataaars/svg?seed=${m.userDetails.full_name || m.user_id}`}
                            alt="Avatar"
                            className="member-avatar"
                          />
                          <div className="member-info">
                            <span className="member-name">{m.userDetails.full_name}</span>
                            <span className="member-email">{m.userDetails.email}</span>
                            <div className="member-allocation-tag">
                              <span>Allocation: {m.allocation_pct}%</span>
                              <span>•</span>
                              <span>{m.effective_daily_hours || 8}h/day</span>
                            </div>
                          </div>

                          {isPMOrAdmin && (
                            <button
                              className="task-action-btn delete"
                              title="Remove from project"
                              onClick={() => onRemoveMember(m.user_id)}
                            >
                              <Trash2 size={13} />
                            </button>
                          )}
                        </div>
                      ))}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </div>

        <div className="modal-footer">
          {isPMOrAdmin && (
            <button className="btn-primary" onClick={onOpenAssignModal}>
              <UserPlus size={16} />
              <span>Assign New Member</span>
            </button>
          )}
          <button className="btn-secondary" onClick={onClose}>
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
