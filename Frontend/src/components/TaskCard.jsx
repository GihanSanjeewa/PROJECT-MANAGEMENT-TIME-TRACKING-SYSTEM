import React from 'react';
import { Calendar, ChevronLeft, ChevronRight, Trash2, User } from 'lucide-react';

export default function TaskCard({ task, columns, onMove, onDelete }) {
  const currentColIndex = columns.findIndex((c) => c.id === task.column_id);
  const prevCol = currentColIndex > 0 ? columns[currentColIndex - 1] : null;
  const nextCol = currentColIndex < columns.length - 1 ? columns[currentColIndex + 1] : null;

  return (
    <div className="task-card">
      <div className="task-card-header">
        <span className={`priority-badge priority-${task.priority}`}>
          ● {task.priority}
        </span>
        <div className="task-actions">
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
              title={`Move to ${nextCol.title}`}
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

      <div className="task-footer">
        <div className="task-assignee">
          <User size={13} />
          <span>{task.assignee || 'Unassigned'}</span>
        </div>
        {task.due_date && (
          <div className="task-due">
            <Calendar size={12} />
            <span>{task.due_date}</span>
          </div>
        )}
      </div>
    </div>
  );
}
