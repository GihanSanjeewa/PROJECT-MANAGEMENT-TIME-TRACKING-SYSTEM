import React, { useState } from 'react';
import { X, PlusCircle, AlertCircle, Calendar } from 'lucide-react';

export default function CreateTaskModal({
  isOpen,
  onClose,
  project,
  members,
  onCreate,
}) {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [requiredRole, setRequiredRole] = useState('DEVELOPER');
  const [priority, setPriority] = useState('HIGH');
  const [estimatedHours, setEstimatedHours] = useState(16);
  const [startDate, setStartDate] = useState(
    project?.start_date ? project.start_date.split('T')[0] : '2026-11-01'
  );
  const [dueDate, setDueDate] = useState(
    project?.start_date
      ? new Date(new Date(project.start_date).getTime() + 7 * 86400000).toISOString().split('T')[0]
      : '2026-11-08'
  );
  const [assignedTo, setAssignedTo] = useState('');
  const [isLoading, setIsLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');

  if (!isOpen) return null;

  // Filter members matching selected role
  const eligibleMembers = members.filter(
    (m) => (m.project_role || '').toUpperCase() === requiredRole.toUpperCase()
  );

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!title.trim()) return;

    setErrorMsg('');
    setIsLoading(true);

    try {
      await onCreate({
        project_id: project.id,
        title: title.trim(),
        description: description.trim(),
        required_role: requiredRole,
        priority: priority,
        estimated_hours: parseFloat(estimatedHours),
        start_date: startDate,
        due_date: dueDate,
        assigned_to: assignedTo ? assignedTo : null,
      });

      setTitle('');
      setDescription('');
      onClose();
    } catch (err) {
      setErrorMsg(err.message || 'Failed to create task');
    } finally {
      setIsLoading(false);
    }
  };

  const projStartStr = project?.start_date ? project.start_date.split('T')[0] : '2026-11-01';
  const projEndStr = project?.target_end_date ? project.target_end_date.split('T')[0] : '2026-12-31';

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal-card" style={{ width: '560px' }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <PlusCircle size={20} color="#6366f1" />
            <h2>Create Task with Timeline Enforcement</h2>
          </div>
          <button className="modal-close" onClick={onClose}>
            <X size={18} />
          </button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="modal-body">
            {/* Timeline Boundary Guide Badge */}
            <div className="timeline-helper-banner">
              <Calendar size={14} color="#818cf8" />
              <span>
                Project Range: <strong>{projStartStr}</strong> to <strong>{projEndStr}</strong>
              </span>
            </div>

            {errorMsg && (
              <div className="timeline-conflict-alert" style={{ marginBottom: '14px' }}>
                <div style={{ display: 'flex', gap: '8px', alignItems: 'flex-start' }}>
                  <AlertCircle size={18} color="#ef4444" style={{ flexShrink: 0, marginTop: '2px' }} />
                  <div>
                    <span style={{ fontWeight: '700' }}>Validation Invariant (422):</span>
                    <div style={{ marginTop: '4px', fontSize: '0.85rem' }}>{errorMsg}</div>
                  </div>
                </div>
              </div>
            )}

            <div className="form-group">
              <label>Task Title *</label>
              <input
                type="text"
                placeholder="e.g. Implement DAG cycle detector"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                autoFocus
                required
              />
            </div>

            <div className="form-group">
              <label>Description</label>
              <textarea
                rows="2"
                placeholder="Technical implementation details and acceptance criteria..."
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
            </div>

            <div className="form-row">
              <div className="form-group">
                <label>Required Functional Role *</label>
                <select
                  value={requiredRole}
                  onChange={(e) => {
                    setRequiredRole(e.target.value);
                    setAssignedTo('');
                  }}
                  required
                >
                  <option value="DEVELOPER">💻 Developer</option>
                  <option value="QA">🧪 QA Engineer</option>
                  <option value="BA">📊 Business Analyst</option>
                </select>
              </div>

              <div className="form-group">
                <label>Priority *</label>
                <select value={priority} onChange={(e) => setPriority(e.target.value)}>
                  <option value="LOW">Low</option>
                  <option value="MEDIUM">Medium</option>
                  <option value="HIGH">High</option>
                  <option value="URGENT">Urgent</option>
                </select>
              </div>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label>Estimated Hours *</label>
                <input
                  type="number"
                  min="0.5"
                  step="0.5"
                  value={estimatedHours}
                  onChange={(e) => setEstimatedHours(e.target.value)}
                  required
                />
              </div>

              <div className="form-group">
                <label>Assignee ({eligibleMembers.length} available)</label>
                <select value={assignedTo} onChange={(e) => setAssignedTo(e.target.value)}>
                  <option value="">Unassigned</option>
                  {eligibleMembers.map((m) => (
                    <option key={m.user_id} value={m.user_id}>
                      {m.userDetails?.full_name || m.user_id} ({m.allocation_pct}%)
                    </option>
                  ))}
                </select>
              </div>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label>Start Date *</label>
                <input
                  type="date"
                  value={startDate}
                  onChange={(e) => setStartDate(e.target.value)}
                  required
                />
              </div>

              <div className="form-group">
                <label>Due Date *</label>
                <input
                  type="date"
                  value={dueDate}
                  onChange={(e) => setDueDate(e.target.value)}
                  required
                />
              </div>
            </div>
          </div>

          <div className="modal-footer">
            <button type="button" className="btn-secondary" onClick={onClose}>
              Cancel
            </button>
            <button type="submit" className="btn-primary" disabled={isLoading}>
              {isLoading ? 'Validating Bounds...' : 'Create Task'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
