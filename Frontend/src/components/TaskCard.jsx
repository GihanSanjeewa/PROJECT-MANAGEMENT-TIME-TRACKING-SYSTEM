import React from 'react';
import { Calendar, ChevronLeft, ChevronRight, Trash2, User, Clock, Play, Square } from 'lucide-react';

export default function TaskCard({
  task,
  columns,
  onMove,
  onDelete,
  onClick,
  activeTimer,
  onStartTimer,
  onStopTimer,
}) {
  // Support both status string ('TODO') and legacy column_id ('col-todo')
  const taskStatus = task.status || (task.column_id ? task.column_id.replace('col-', '').toUpperCase() : 'TODO');
  const currentColIndex = columns.findIndex(
    (c) => c.id === taskStatus || c.id === task.column_id
  );
  const prevCol = currentColIndex > 0 ? columns[currentColIndex - 1] : null;
  const nextCol = currentColIndex < columns.length - 1 ? columns[currentColIndex + 1] : null;

  const roleMeta = {
    DEVELOPER: { label: 'DEV', icon: '💻', color: '#38bdf8', bg: 'rgba(56, 189, 248, 0.12)' },
    QA: { label: 'QA', icon: '🧪', color: '#10b981', bg: 'rgba(16, 185, 129, 0.12)' },
    BA: { label: 'BA', icon: '📊', color: '#f59e0b', bg: 'rgba(245, 158, 11, 0.12)' },
  }[task.required_role] || { label: task.required_role || 'DEV', icon: '💻', color: '#818cf8', bg: 'rgba(129, 140, 248, 0.12)' };

  const priorityKey = (task.priority || 'medium').toLowerCase();
  const isTimerRunning = activeTimer && activeTimer.task_id === task.id;
  const logged = task.logged_hours || 0;
  const estimated = task.estimated_hours || 0;
  const progressPct = estimated > 0 ? Math.min(100, Math.round((logged / estimated) * 100)) : 0;
  const isOverBudget = estimated > 0 && logged > estimated;

  return (
    <div className={`task-card ${isTimerRunning ? 'task-card-active-timer' : ''}`} onClick={() => onClick && onClick(task)}>
      <div className="task-card-header">
        <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
          <span
            className="role-pill-small"
            style={{ color: roleMeta.color, background: roleMeta.bg }}
          >
            {roleMeta.icon} {roleMeta.label}
          </span>
          <span className={`priority-badge priority-${priorityKey}`}>
            ● {task.priority || 'MEDIUM'}
          </span>
        </div>

        <div className="task-actions" onClick={(e) => e.stopPropagation()}>
          {/* Quick Timer Start / Stop Action */}
          {isTimerRunning ? (
            <button
              className="task-action-btn timer-running"
              title="Stop active timer"
              onClick={(e) => {
                e.stopPropagation();
                if (onStopTimer) onStopTimer();
              }}
            >
              <Square size={12} fill="#ef4444" color="#ef4444" />
            </button>
          ) : (
            <button
              className="task-action-btn"
              title="Start live timer on this task"
              onClick={(e) => {
                e.stopPropagation();
                if (onStartTimer) onStartTimer(task.id, task.title);
              }}
            >
              <Play size={12} color="#10b981" />
            </button>
          )}

          {prevCol && (
            <button
              className="task-action-btn"
              title={`Move to ${prevCol.title}`}
              onClick={(e) => {
                e.stopPropagation();
                onMove(task.id, prevCol.id);
              }}
            >
              <ChevronLeft size={14} />
            </button>
          )}
          {nextCol && (
            <button
              className="task-action-btn"
              title={`Advance to ${nextCol.title}`}
              onClick={(e) => {
                e.stopPropagation();
                onMove(task.id, nextCol.id);
              }}
            >
              <ChevronRight size={14} />
            </button>
          )}
          <button
            className="task-action-btn delete"
            title="Delete task"
            onClick={(e) => {
              e.stopPropagation();
              onDelete(task.id);
            }}
          >
            <Trash2 size={13} />
          </button>
        </div>
      </div>

      <div className="task-title">{task.title}</div>
      {task.description && <div className="task-desc">{task.description}</div>}

      {/* Logged vs Estimated Progress Bar */}
      {estimated > 0 && (
        <div style={{ marginTop: '8px', marginBottom: '8px' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: '0.68rem', color: isOverBudget ? '#ef4444' : 'var(--text-muted)', marginBottom: '3px' }}>
            <span>{logged}h / {estimated}h</span>
            <span>{isOverBudget ? `+${(logged - estimated).toFixed(1)}h over` : `${progressPct}%`}</span>
          </div>
          <div style={{ height: '4px', background: 'rgba(255, 255, 255, 0.08)', borderRadius: '2px', overflow: 'hidden' }}>
            <div
              style={{
                height: '100%',
                width: `${Math.min(100, (logged / estimated) * 100)}%`,
                background: isOverBudget ? '#ef4444' : progressPct >= 100 ? '#10b981' : '#38bdf8',
                borderRadius: '2px',
              }}
            />
          </div>
        </div>
      )}

      <div className="task-footer">
        <div className="task-assignee">
          <User size={13} />
          <span>{task.assignee || 'Unassigned'}</span>
        </div>

        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          {task.due_date && (
            <div className="task-due">
              <Calendar size={11} />
              <span>{task.due_date.split('T')[0]}</span>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
