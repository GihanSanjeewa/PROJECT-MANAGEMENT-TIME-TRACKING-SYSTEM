import React from 'react';
import { X, Activity, AlertTriangle, CheckCircle, ShieldCheck } from 'lucide-react';

export default function WorkloadModal({ isOpen, onClose, workloads }) {
  if (!isOpen) return null;

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal-card" style={{ width: '740px' }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <Activity size={20} color="#f59e0b" />
            <h2>Developer Daily Workload & Over-Allocation Detector (TSK-020)</h2>
          </div>
          <button className="modal-close" onClick={onClose}>
            <X size={18} />
          </button>
        </div>

        <div className="modal-body">
          <p style={{ fontSize: '0.85rem', color: 'var(--text-dim)', marginBottom: '16px' }}>
            Calculates daily workload by spreading estimated hours across working days
            (excluding weekends & holidays). Highlights team members who exceed daily capacity.
          </p>

          {workloads.length === 0 ? (
            <div className="empty-state-small">No member workload data available.</div>
          ) : (
            <div className="workload-list">
              {workloads.map((w) => {
                const isOver = w.is_overallocated;
                return (
                  <div key={w.user_id} className={`workload-card ${isOver ? 'overallocated' : ''}`}>
                    <div className="workload-card-header">
                      <div className="workload-user-meta">
                        <img
                          src={`https://api.dicebear.com/7.x/avataaars/svg?seed=${w.user_name || w.user_id}`}
                          alt="Avatar"
                          className="member-avatar"
                        />
                        <div>
                          <div className="workload-user-name">{w.user_name}</div>
                          <div className="workload-user-role">
                            <span>{w.role}</span>
                            <span>•</span>
                            <span>Daily Cap: {w.daily_capacity}h/day</span>
                            <span>•</span>
                            <span>Total Assigned: {w.total_assigned_hrs}h</span>
                          </div>
                        </div>
                      </div>

                      <div className="workload-status-badge">
                        {isOver ? (
                          <div className="badge-overloaded">
                            <AlertTriangle size={14} />
                            <span>Overallocated (Peak: {w.max_daily_hours}h)</span>
                          </div>
                        ) : (
                          <div className="badge-healthy">
                            <ShieldCheck size={14} />
                            <span>Healthy (Peak: {w.max_daily_hours}h)</span>
                          </div>
                        )}
                      </div>
                    </div>

                    {/* Daily Breakdown Strip */}
                    <div className="daily-workload-strip">
                      {w.daily_breakdown && w.daily_breakdown.slice(0, 20).map((day) => {
                        const pct = Math.min(100, Math.round((day.hours / day.capacity_hrs) * 100));
                        return (
                          <div
                            key={day.date}
                            className={`day-cell ${day.is_overloaded ? 'cell-over' : day.hours > 0 ? 'cell-active' : 'cell-idle'}`}
                            title={`${day.date}: ${day.hours}h / ${day.capacity_hrs}h capacity ${day.is_overloaded ? `(+${day.excess_hours}h EXCESS)` : ''}`}
                          >
                            <span className="day-date">{day.date.split('-').slice(1).join('/')}</span>
                            <div className="day-bar-bg">
                              <div
                                className="day-bar-fill"
                                style={{
                                  height: `${pct}%`,
                                  backgroundColor: day.is_overloaded ? '#ef4444' : '#6366f1',
                                }}
                              />
                            </div>
                            <span className="day-hrs">{day.hours > 0 ? `${day.hours}h` : '-'}</span>
                          </div>
                        );
                      })}
                    </div>
                  </div>
                );
              })}
            </div>
          )}
        </div>

        <div className="modal-footer">
          <button className="btn-secondary" onClick={onClose}>
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
