import React, { useState } from 'react';
import { X, UserPlus } from 'lucide-react';

export default function AssignMemberModal({ isOpen, onClose, users, onAssign }) {
  const [userId, setUserId] = useState(users[0]?.id || '');
  const [role, setRole] = useState('DEVELOPER');
  const [allocationPct, setAllocationPct] = useState(100);
  const [dailyHours, setDailyHours] = useState(8.0);
  const [isLoading, setIsLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');

  if (!isOpen) return null;

  const handleSubmit = async (e) => {
    e.preventDefault();
    if (!userId) return;
    setErrorMsg('');
    setIsLoading(true);

    try {
      await onAssign({
        user_id: userId,
        project_role: role,
        allocation_pct: parseInt(allocationPct, 10),
        effective_daily_hours: parseFloat(dailyHours),
      });
      onClose();
    } catch (err) {
      setErrorMsg(err.message || 'Failed to assign member to project');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal-card" style={{ width: '480px' }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <UserPlus size={18} color="#6366f1" />
            <h2>Assign Team Member to Project</h2>
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

            <div className="form-group">
              <label>Select Team Member *</label>
              <select value={userId} onChange={(e) => setUserId(e.target.value)} required>
                {users.map((u) => (
                  <option key={u.id} value={u.id}>
                    {u.full_name} ({u.email}) — {u.primary_role}
                  </option>
                ))}
              </select>
            </div>

            <div className="form-group">
              <label>Project Functional Role *</label>
              <select value={role} onChange={(e) => setRole(e.target.value)}>
                <option value="DEVELOPER">💻 Developer</option>
                <option value="QA">🧪 QA Engineer</option>
                <option value="BA">📊 Business Analyst</option>
                <option value="PM">👑 Project Manager</option>
              </select>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label>Allocation ({allocationPct}%)</label>
                <input
                  type="range"
                  min="10"
                  max="100"
                  step="10"
                  value={allocationPct}
                  onChange={(e) => setAllocationPct(e.target.value)}
                />
              </div>

              <div className="form-group">
                <label>Daily Hours</label>
                <input
                  type="number"
                  min="1"
                  max="24"
                  step="0.5"
                  value={dailyHours}
                  onChange={(e) => setDailyHours(e.target.value)}
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
              {isLoading ? 'Assigning...' : 'Confirm Assignment'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
