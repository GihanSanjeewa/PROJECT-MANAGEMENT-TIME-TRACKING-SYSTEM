import React, { useState } from 'react';
import { X, FolderPlus } from 'lucide-react';

export default function CreateProjectModal({ isOpen, onClose, onCreate }) {
  const [code, setCode] = useState('PRJ-02');
  const [name, setName] = useState('');
  const [description, setDescription] = useState('');
  const [startDate, setStartDate] = useState('2026-11-01');
  const [targetEndDate, setTargetEndDate] = useState('2026-12-31');
  const [budgetHours, setBudgetHours] = useState(400);

  const [isLoading, setIsLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');

  if (!isOpen) return null;

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!code || !name) return;
    setErrorMsg('');
    setIsLoading(true);

    try {
      await onCreate({
        code: code.trim().toUpperCase(),
        name: name.trim(),
        description: description.trim(),
        start_date: new Date(startDate).toISOString(),
        target_end_date: new Date(targetEndDate).toISOString(),
        budget_hours: parseFloat(budgetHours),
      });
      onClose();
    } catch (err) {
      setErrorMsg(err.message || 'Failed to create project');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal-card" style={{ width: '540px' }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <FolderPlus size={18} color="#6366f1" />
            <h2>Create New Project</h2>
          </div>
          <button className="modal-close" onClick={onClose}>
            <X size={18} />
          </button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="modal-body">
            {errorMsg && (
              <div className="login-error-alert" style={{ marginBottom: '14px' }}>
                <span>{errorMsg}</span>
              </div>
            )}

            <div className="form-row">
              <div className="form-group">
                <label>Project Code *</label>
                <input
                  type="text"
                  placeholder="e.g. CORE, MOB, SEC"
                  value={code}
                  onChange={(e) => setCode(e.target.value)}
                  required
                />
              </div>

              <div className="form-group">
                <label>Budget Hours</label>
                <input
                  type="number"
                  min="0"
                  step="50"
                  value={budgetHours}
                  onChange={(e) => setBudgetHours(e.target.value)}
                />
              </div>
            </div>

            <div className="form-group">
              <label>Project Name *</label>
              <input
                type="text"
                placeholder="e.g. Next-Gen Mobile App"
                value={name}
                onChange={(e) => setName(e.target.value)}
                required
              />
            </div>

            <div className="form-group">
              <label>Description</label>
              <textarea
                rows="2"
                placeholder="High-level project scope and goals..."
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
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
          </div>

          <div className="modal-footer">
            <button type="button" className="btn-secondary" onClick={onClose}>
              Cancel
            </button>
            <button type="submit" className="btn-primary" disabled={isLoading}>
              {isLoading ? 'Creating...' : 'Create Project'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
