import React, { useState } from 'react';
import { X, Settings, AlertTriangle, ShieldCheck } from 'lucide-react';

export default function ProjectSettingsModal({ isOpen, onClose, project, onUpdate }) {
  const [name, setName] = useState(project.name || '');
  const [description, setDescription] = useState(project.description || '');
  const [startDate, setStartDate] = useState(
    project.start_date ? project.start_date.split('T')[0] : '2026-11-01'
  );
  const [targetEndDate, setTargetEndDate] = useState(
    project.target_end_date ? project.target_end_date.split('T')[0] : '2026-12-31'
  );
  const [budgetHours, setBudgetHours] = useState(project.budget_hours || 600);
  const [status, setStatus] = useState(project.status || 'ACTIVE');

  const [isLoading, setIsLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');
  const [conflictDetails, setConflictDetails] = useState(null);

  if (!isOpen) return null;

  const handleSubmit = async (e) => {
    e.preventDefault();
    setErrorMsg('');
    setConflictDetails(null);
    setIsLoading(true);

    try {
      await onUpdate(project.id, {
        name,
        description,
        start_date: new Date(startDate).toISOString(),
        target_end_date: new Date(targetEndDate).toISOString(),
        budget_hours: parseFloat(budgetHours),
        status,
      });
      onClose();
    } catch (err) {
      setErrorMsg(err.message || 'Failed to update project timeline');
      if (err.details) {
        setConflictDetails(err.details);
      }
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal-card" style={{ width: '560px' }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <Settings size={20} color="#6366f1" />
            <h2>Project Timeline & Settings</h2>
          </div>
          <button className="modal-close" onClick={onClose}>
            <X size={18} />
          </button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="modal-body">
            {errorMsg && (
              <div className="timeline-conflict-alert">
                <div style={{ display: 'flex', gap: '8px', alignItems: 'flex-start' }}>
                  <AlertTriangle size={18} color="#ef4444" style={{ flexShrink: 0, marginTop: '2px' }} />
                  <div>
                    <span style={{ fontWeight: '700' }}>Timeline Invariant Violation (409 Conflict):</span>
                    <div style={{ marginTop: '4px', fontSize: '0.85rem' }}>{errorMsg}</div>
                    {conflictDetails && (
                      <div style={{ marginTop: '8px', fontSize: '0.78rem', color: '#fca5a5' }}>
                        Offending tasks must be rescheduled before shrinking the project deadline.
                      </div>
                    )}
                  </div>
                </div>
              </div>
            )}

            <div className="form-group">
              <label>Project Name *</label>
              <input type="text" value={name} onChange={(e) => setName(e.target.value)} required />
            </div>

            <div className="form-group">
              <label>Description</label>
              <textarea rows="2" value={description} onChange={(e) => setDescription(e.target.value)} />
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
                <label>Target End Date *</label>
                <input
                  type="date"
                  value={targetEndDate}
                  onChange={(e) => setTargetEndDate(e.target.value)}
                  required
                />
              </div>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label>Budget Hours</label>
                <input
                  type="number"
                  min="0"
                  step="10"
                  value={budgetHours}
                  onChange={(e) => setBudgetHours(e.target.value)}
                />
              </div>

              <div className="form-group">
                <label>Status</label>
                <select value={status} onChange={(e) => setStatus(e.target.value)}>
                  <option value="DRAFT">Draft</option>
                  <option value="ACTIVE">Active</option>
                  <option value="ON_HOLD">On Hold</option>
                  <option value="COMPLETED">Completed</option>
                </select>
              </div>
            </div>

            <div className="timeline-guard-hint">
              <ShieldCheck size={14} color="#10b981" />
              <span>
                Zero-Bug Rule: All child tasks must fit strictly within [Start Date, Target End Date].
              </span>
            </div>
          </div>

          <div className="modal-footer">
            <button type="button" className="btn-secondary" onClick={onClose}>
              Cancel
            </button>
            <button type="submit" className="btn-primary" disabled={isLoading}>
              {isLoading ? 'Validating & Saving...' : 'Save Timeline'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
