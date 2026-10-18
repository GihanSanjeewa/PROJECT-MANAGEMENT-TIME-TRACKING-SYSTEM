import React, { useState, useEffect } from 'react';
import {
  X,
  Clock,
  Calendar,
  User,
  ArrowRight,
  GitBranch,
  ShieldAlert,
  CheckCircle2,
  AlertTriangle,
  History,
  Trash2,
  Plus,
} from 'lucide-react';
import {
  updateTaskStatus,
  fetchTaskAudit,
  addDependency,
  deleteDependency,
} from '../services/api';

export default function TaskDetailModal({
  isOpen,
  onClose,
  task,
  allTasks,
  dependencies,
  users,
  currentUser,
  onTaskUpdated,
}) {
  const [activeTab, setActiveTab] = useState('workflow'); // 'workflow', 'dependencies', 'audit'
  const [comment, setComment] = useState('');
  const [auditLogs, setAuditLogs] = useState([]);
  const [dependsOnId, setDependsOnId] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');
  const [successMsg, setSuccessMsg] = useState('');

  useEffect(() => {
    if (!isOpen || !task) return;
    setErrorMsg('');
    setSuccessMsg('');
    loadAudit();
  }, [isOpen, task]);

  const loadAudit = async () => {
    if (!task) return;
    const logs = await fetchTaskAudit(task.id);
    setAuditLogs(logs);
  };

  if (!isOpen || !task) return null;

  // Next valid statuses per state machine (TSK-021)
  const statusTransitions = {
    BACKLOG: ['TODO', 'BLOCKED'],
    TODO: ['IN_PROGRESS', 'BLOCKED', 'BACKLOG'],
    IN_PROGRESS: ['CODE_REVIEW', 'BLOCKED', 'TODO'],
    CODE_REVIEW: ['IN_QA', 'IN_PROGRESS', 'BLOCKED'],
    IN_QA: ['DONE', 'IN_PROGRESS', 'BLOCKED'],
    BLOCKED: ['BACKLOG', 'TODO', 'IN_PROGRESS', 'CODE_REVIEW', 'IN_QA'],
    DONE: ['TODO'],
  };

  const allowedNext = statusTransitions[task.status] || [];

  const handleStatusChange = async (nextStatus) => {
    setErrorMsg('');
    setSuccessMsg('');
    setIsLoading(true);

    try {
      await updateTaskStatus(task.id, nextStatus, comment);
      setSuccessMsg(`Status updated to ${nextStatus}`);
      setComment('');
      await loadAudit();
      if (onTaskUpdated) onTaskUpdated();
    } catch (err) {
      setErrorMsg(err.message || 'Status transition rejected');
    } finally {
      setIsLoading(false);
    }
  };

  const handleAddDependency = async (e) => {
    e.preventDefault();
    if (!dependsOnId) return;

    setErrorMsg('');
    setSuccessMsg('');
    setIsLoading(true);

    try {
      await addDependency(task.id, dependsOnId, 'FS');
      setSuccessMsg('Finish-to-Start dependency added successfully');
      setDependsOnId('');
      if (onTaskUpdated) onTaskUpdated();
    } catch (err) {
      setErrorMsg(err.message || 'Cycle detected or invalid dependency');
    } finally {
      setIsLoading(false);
    }
  };

  const handleDeleteDependency = async (depId) => {
    await deleteDependency(depId);
    if (onTaskUpdated) onTaskUpdated();
  };

  // Find prerequisite dependencies for this task
  const prerequisites = dependencies.filter((d) => d.task_id === task.id);
  // Find tasks that depend on this task
  const successors = dependencies.filter((d) => d.depends_on_task_id === task.id);

  // Filter tasks eligible to be prerequisites (exclude self)
  const candidatePrereqs = allTasks.filter((t) => t.id !== task.id);

  const assigneeUser = users.find((u) => u.id === task.assigned_to) || {
    full_name: task.assignee || 'Unassigned',
  };

  const roleColors = {
    DEVELOPER: { bg: 'rgba(56, 189, 248, 0.15)', text: '#38bdf8', icon: '💻' },
    QA: { bg: 'rgba(16, 185, 129, 0.15)', text: '#10b981', icon: '🧪' },
    BA: { bg: 'rgba(245, 158, 11, 0.15)', text: '#f59e0b', icon: '📊' },
  };

  const roleMeta = roleColors[task.required_role] || roleColors.DEVELOPER;

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal-card" style={{ width: '640px' }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <span
              className="role-badge"
              style={{ background: roleMeta.bg, color: roleMeta.text }}
            >
              {roleMeta.icon} {task.required_role}
            </span>
            <span className={`priority-badge priority-${task.priority?.toLowerCase()}`}>
              ● {task.priority}
            </span>
            <span className="task-status-pill">Status: {task.status}</span>
          </div>
          <button className="modal-close" onClick={onClose}>
            <X size={18} />
          </button>
        </div>

        <div className="modal-body">
          <h2 style={{ fontSize: '1.25rem', marginBottom: '8px', color: '#fff' }}>{task.title}</h2>
          {task.description && (
            <p style={{ color: 'var(--text-dim)', fontSize: '0.88rem', marginBottom: '16px' }}>
              {task.description}
            </p>
          )}

          {/* Quick Metrics Bar */}
          <div className="task-detail-metrics">
            <div className="metric-item">
              <Calendar size={14} color="#818cf8" />
              <span>
                {task.start_date ? task.start_date.split('T')[0] : 'N/A'} —{' '}
                {task.due_date ? task.due_date.split('T')[0] : 'N/A'}
              </span>
            </div>
            <div className="metric-item">
              <Clock size={14} color="#10b981" />
              <span>
                Est: {task.estimated_hours || 0}h (Logged: {task.logged_hours || 0}h)
              </span>
            </div>
            <div className="metric-item">
              <User size={14} color="#f59e0b" />
              <span>Assignee: {assigneeUser.full_name}</span>
            </div>
          </div>

          {/* Error / Success Feedback */}
          {errorMsg && (
            <div className="timeline-conflict-alert" style={{ margin: '14px 0' }}>
              <div style={{ display: 'flex', gap: '8px', alignItems: 'flex-start' }}>
                <AlertTriangle size={18} color="#ef4444" style={{ flexShrink: 0, marginTop: '2px' }} />
                <div>
                  <span style={{ fontWeight: '700' }}>Workflow or DAG Invariant Violation:</span>
                  <div style={{ marginTop: '4px', fontSize: '0.85rem' }}>{errorMsg}</div>
                </div>
              </div>
            </div>
          )}

          {successMsg && (
            <div className="alert-success" style={{ margin: '14px 0' }}>
              <CheckCircle2 size={16} />
              <span>{successMsg}</span>
            </div>
          )}

          {/* Tab Navigation */}
          <div className="tabs-header" style={{ marginTop: '16px' }}>
            <button
              className={`tab-btn ${activeTab === 'workflow' ? 'active' : ''}`}
              onClick={() => setActiveTab('workflow')}
            >
              <ArrowRight size={14} />
              <span>Workflow State Machine (TSK-021)</span>
            </button>
            <button
              className={`tab-btn ${activeTab === 'dependencies' ? 'active' : ''}`}
              onClick={() => setActiveTab('dependencies')}
            >
              <GitBranch size={14} />
              <span>DAG Dependencies ({prerequisites.length})</span>
            </button>
            <button
              className={`tab-btn ${activeTab === 'audit' ? 'active' : ''}`}
              onClick={() => setActiveTab('audit')}
            >
              <History size={14} />
              <span>Audit Trail ({auditLogs.length})</span>
            </button>
          </div>

          {/* TAB 1: WORKFLOW STATE MACHINE */}
          {activeTab === 'workflow' && (
            <div className="tab-pane">
              <div style={{ fontSize: '0.85rem', color: 'var(--text-dim)', marginBottom: '12px' }}>
                Current Status: <strong style={{ color: '#fff' }}>{task.status}</strong>. Advance
                task across BA → Dev → QA hand-off pipeline:
              </div>

              <div className="workflow-action-buttons">
                {allowedNext.length === 0 ? (
                  <div style={{ fontSize: '0.85rem', color: 'var(--text-dim)' }}>
                    No valid forward transitions from this state.
                  </div>
                ) : (
                  allowedNext.map((nextState) => (
                    <button
                      key={nextState}
                      className="btn-status-advance"
                      onClick={() => handleStatusChange(nextState)}
                      disabled={isLoading}
                    >
                      <ArrowRight size={13} />
                      <span>Advance to {nextState}</span>
                    </button>
                  ))
                )}
              </div>

              <div className="form-group" style={{ marginTop: '16px' }}>
                <label>Transition Comment / Hand-off Note (optional)</label>
                <input
                  type="text"
                  placeholder="e.g. Unit tests passing, ready for QA verification"
                  value={comment}
                  onChange={(e) => setComment(e.target.value)}
                />
              </div>

              {task.status === 'IN_QA' && (
                <div className="qa-role-notice">
                  <ShieldAlert size={16} color="#38bdf8" />
                  <span>
                    Zero-Bug Quality Gate: Transitioning from <strong>IN_QA</strong> to{' '}
                    <strong>DONE</strong> requires QA Engineer role or Project Manager authorization.
                  </span>
                </div>
              )}
            </div>
          )}

          {/* TAB 2: DAG DEPENDENCIES */}
          {activeTab === 'dependencies' && (
            <div className="tab-pane">
              <div style={{ marginBottom: '14px' }}>
                <h4 style={{ fontSize: '0.88rem', color: '#fff', marginBottom: '6px' }}>
                  Prerequisites (This task depends on):
                </h4>
                {prerequisites.length === 0 ? (
                  <div className="empty-state-small">No predecessor dependencies linked yet.</div>
                ) : (
                  <div className="dependency-list">
                    {prerequisites.map((d) => {
                      const predTask = allTasks.find((t) => t.id === d.depends_on_task_id) || {
                        title: d.depends_on_task_id,
                      };
                      return (
                        <div key={d.id} className="dependency-item">
                          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                            <span className="dep-type-badge">{d.dependency_type || 'FS'}</span>
                            <span style={{ fontSize: '0.85rem', color: '#e2e8f0' }}>
                              {predTask.title}
                            </span>
                          </div>
                          <button
                            className="task-action-btn delete"
                            title="Remove dependency"
                            onClick={() => handleDeleteDependency(d.id)}
                          >
                            <Trash2 size={12} />
                          </button>
                        </div>
                      );
                    })}
                  </div>
                )}
              </div>

              {successors.length > 0 && (
                <div style={{ marginBottom: '16px' }}>
                  <h4 style={{ fontSize: '0.88rem', color: '#fff', marginBottom: '6px' }}>
                    Successors (Prerequisite for):
                  </h4>
                  <div className="dependency-list">
                    {successors.map((d) => {
                      const succTask = allTasks.find((t) => t.id === d.task_id) || {
                        title: d.task_id,
                      };
                      return (
                        <div key={d.id} className="dependency-item">
                          <span className="dep-type-badge">BLOCKS</span>
                          <span style={{ fontSize: '0.85rem', color: '#e2e8f0' }}>
                            {succTask.title}
                          </span>
                        </div>
                      );
                    })}
                  </div>
                </div>
              )}

              <form onSubmit={handleAddDependency} style={{ marginTop: '16px' }}>
                <h4 style={{ fontSize: '0.88rem', color: '#fff', marginBottom: '8px' }}>
                  Link New Finish-to-Start Dependency:
                </h4>
                <div style={{ display: 'flex', gap: '8px' }}>
                  <select
                    value={dependsOnId}
                    onChange={(e) => setDependsOnId(e.target.value)}
                    style={{ flex: 1 }}
                    required
                  >
                    <option value="">Select Predecessor Task...</option>
                    {candidatePrereqs.map((t) => (
                      <option key={t.id} value={t.id}>
                        {t.title} ({t.required_role} — due: {t.due_date ? t.due_date.split('T')[0] : 'N/A'})
                      </option>
                    ))}
                  </select>
                  <button type="submit" className="btn-primary" disabled={isLoading || !dependsOnId}>
                    <Plus size={14} />
                    <span>Add</span>
                  </button>
                </div>
                <div style={{ fontSize: '0.76rem', color: 'var(--text-dim)', marginTop: '6px' }}>
                  DFS cycle detection validates in &lt;5ms to prevent circular deadlock loops.
                </div>
              </form>
            </div>
          )}

          {/* TAB 3: AUDIT TRAIL */}
          {activeTab === 'audit' && (
            <div className="tab-pane">
              {auditLogs.length === 0 ? (
                <div className="empty-state-small">No status transitions recorded yet.</div>
              ) : (
                <div className="audit-timeline">
                  {auditLogs.map((log) => (
                    <div key={log.id} className="audit-entry">
                      <div className="audit-dot" />
                      <div className="audit-content">
                        <div className="audit-header">
                          <span className="audit-change">
                            {log.old_status} → <strong>{log.new_status}</strong>
                          </span>
                          <span className="audit-time">
                            {log.created_at ? new Date(log.created_at).toLocaleTimeString() : ''}
                          </span>
                        </div>
                        {log.comment && <div className="audit-comment">"{log.comment}"</div>}
                        <div className="audit-author">By: {log.changed_by}</div>
                      </div>
                    </div>
                  ))}
                </div>
              )}
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
