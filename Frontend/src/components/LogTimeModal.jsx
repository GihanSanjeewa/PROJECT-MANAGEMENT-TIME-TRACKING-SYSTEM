import React, { useState } from 'react';
import { X, Clock, Calendar, FileText, CheckCircle2, AlertCircle } from 'lucide-react';
import { createTimeLog } from '../services/api';

export default function LogTimeModal({ isOpen, onClose, task, onLogSuccess }) {
  const today = new Date().toISOString().split('T')[0];

  const [mode, setMode] = useState('duration'); // 'duration' or 'range'
  const [workDate, setWorkDate] = useState(today);
  const [hours, setHours] = useState('1');
  const [minutes, setMinutes] = useState('30');
  const [startTime, setStartTime] = useState('09:00');
  const [endTime, setEndTime] = useState('11:00');
  const [description, setDescription] = useState('');
  const [isSubmitting, setIsSubmitting] = useState(false);
  const [error, setError] = useState('');

  if (!isOpen || !task) return null;

  const handleSubmit = async (e) => {
    e.preventDefault();
    setIsSubmitting(true);
    setError('');

    try {
      let payload = {
        work_date: workDate,
        description: description.trim() || 'Work session',
      };

      if (mode === 'duration') {
        const totalMinutes = parseInt(hours || 0, 10) * 60 + parseInt(minutes || 0, 10);
        if (totalMinutes <= 0) {
          throw new Error('Total logged duration must be greater than 0 minutes');
        }
        if (totalMinutes > 1440) {
          throw new Error('Duration cannot exceed 24 hours in a single log');
        }
        payload.duration_minutes = totalMinutes;
      } else {
        if (!startTime || !endTime) {
          throw new Error('Both start and end time are required in range mode');
        }
        const startISO = `${workDate}T${startTime}:00Z`;
        const endISO = `${workDate}T${endTime}:00Z`;
        if (new Date(endISO) <= new Date(startISO)) {
          throw new Error('End time must be strictly after start time');
        }
        payload.start_time = startISO;
        payload.end_time = endISO;
      }

      await createTimeLog(task.id, payload);
      if (onLogSuccess) onLogSuccess();
      onClose();
    } catch (err) {
      setError(err.message);
    } finally {
      setIsSubmitting(false);
    }
  };

  return (
    <div className="modal-backdrop">
      <div className="modal-content" style={{ maxWidth: '480px' }}>
        <div className="modal-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
            <Clock size={20} color="var(--primary)" />
            <h2 style={{ fontSize: '1.15rem', margin: 0 }}>Log Time on Task</h2>
          </div>
          <button className="close-btn" onClick={onClose}>
            <X size={18} />
          </button>
        </div>

        <div style={{ padding: '0 20px', marginTop: '10px' }}>
          <div style={{ background: 'rgba(255, 255, 255, 0.03)', padding: '10px 14px', borderRadius: '8px', border: '1px solid var(--border)' }}>
            <div style={{ fontSize: '0.75rem', color: 'var(--text-muted)' }}>Target Task:</div>
            <div style={{ fontSize: '0.9rem', fontWeight: 600, color: 'var(--text-main)', marginTop: '2px' }}>
              {task.title}
            </div>
          </div>
        </div>

        <form onSubmit={handleSubmit} style={{ padding: '20px', display: 'flex', flexDirection: 'column', gap: '16px' }}>
          {error && (
            <div className="alert-box" style={{ background: 'rgba(239, 68, 68, 0.1)', border: '1px solid #ef4444', color: '#f87171', padding: '10px 14px', borderRadius: '8px', display: 'flex', alignItems: 'center', gap: '8px', fontSize: '0.85rem' }}>
              <AlertCircle size={16} />
              <span>{error}</span>
            </div>
          )}

          {/* Date Picker (No future dates allowed) */}
          <div className="form-group">
            <label style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
              <Calendar size={14} />
              <span>Work Date</span>
            </label>
            <input
              type="date"
              className="form-input"
              value={workDate}
              max={today}
              onChange={(e) => setWorkDate(e.target.value)}
              required
            />
          </div>

          {/* Mode Selector Toggle */}
          <div style={{ display: 'flex', borderRadius: '8px', overflow: 'hidden', border: '1px solid var(--border)' }}>
            <button
              type="button"
              style={{
                flex: 1,
                padding: '8px',
                background: mode === 'duration' ? 'var(--primary)' : 'transparent',
                color: mode === 'duration' ? '#fff' : 'var(--text-muted)',
                border: 'none',
                fontWeight: 600,
                fontSize: '0.85rem',
                cursor: 'pointer',
              }}
              onClick={() => setMode('duration')}
            >
              Duration (Hours/Min)
            </button>
            <button
              type="button"
              style={{
                flex: 1,
                padding: '8px',
                background: mode === 'range' ? 'var(--primary)' : 'transparent',
                color: mode === 'range' ? '#fff' : 'var(--text-muted)',
                border: 'none',
                fontWeight: 600,
                fontSize: '0.85rem',
                cursor: 'pointer',
              }}
              onClick={() => setMode('range')}
            >
              Time Range (Start/End)
            </button>
          </div>

          {mode === 'duration' ? (
            <div style={{ display: 'flex', gap: '12px' }}>
              <div className="form-group" style={{ flex: 1 }}>
                <label>Hours</label>
                <input
                  type="number"
                  className="form-input"
                  min="0"
                  max="24"
                  value={hours}
                  onChange={(e) => setHours(e.target.value)}
                />
              </div>
              <div className="form-group" style={{ flex: 1 }}>
                <label>Minutes</label>
                <input
                  type="number"
                  className="form-input"
                  min="0"
                  max="59"
                  step="5"
                  value={minutes}
                  onChange={(e) => setMinutes(e.target.value)}
                />
              </div>
            </div>
          ) : (
            <div style={{ display: 'flex', gap: '12px' }}>
              <div className="form-group" style={{ flex: 1 }}>
                <label>Start Time</label>
                <input
                  type="time"
                  className="form-input"
                  value={startTime}
                  onChange={(e) => setStartTime(e.target.value)}
                  required
                />
              </div>
              <div className="form-group" style={{ flex: 1 }}>
                <label>End Time</label>
                <input
                  type="time"
                  className="form-input"
                  value={endTime}
                  onChange={(e) => setEndTime(e.target.value)}
                  required
                />
              </div>
            </div>
          )}

          {/* Description */}
          <div className="form-group">
            <label style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
              <FileText size={14} />
              <span>Notes / Description</span>
            </label>
            <textarea
              className="form-input"
              rows={3}
              placeholder="What did you accomplish during this session?"
              value={description}
              onChange={(e) => setDescription(e.target.value)}
            />
          </div>

          <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px', marginTop: '10px' }}>
            <button type="button" className="btn-secondary" onClick={onClose} disabled={isSubmitting}>
              Cancel
            </button>
            <button type="submit" className="btn-primary" disabled={isSubmitting}>
              {isSubmitting ? 'Logging...' : 'Save Time Log'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
