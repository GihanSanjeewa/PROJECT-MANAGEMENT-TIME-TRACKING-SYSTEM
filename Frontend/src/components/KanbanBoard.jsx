import React from 'react';
import { Plus } from 'lucide-react';
import TaskCard from './TaskCard';

export default function KanbanBoard({
  columns,
  tasks,
  onMoveTask,
  onDeleteTask,
  onOpenCreateModal,
  onSelectTask,
  activeTimer,
  onStartTimer,
  onStopTimer,
}) {
  return (
    <div className="board-container">
      {columns.map((column) => {
        const columnTasks = tasks.filter((t) => {
          if (t.status) return t.status.toUpperCase() === column.id.toUpperCase();
          if (t.column_id) {
            const mapped = t.column_id.replace('col-', '').toUpperCase();
            return mapped === column.id.toUpperCase() || t.column_id === column.id;
          }
          return column.id === 'TODO';
        });

        return (
          <div key={column.id} className="kanban-column">
            <div className="column-header">
              <div className="column-title-wrap">
                <span className="column-dot" style={{ backgroundColor: column.color }} />
                <span className="column-title">{column.title}</span>
                <span className="column-count">{columnTasks.length}</span>
              </div>
              <button
                className="column-add-btn"
                title={`Add task to ${column.title}`}
                onClick={() => onOpenCreateModal(column.id)}
              >
                <Plus size={16} />
              </button>
            </div>

            <div className="column-tasks">
              {columnTasks.length === 0 ? (
                <div
                  style={{
                    textAlign: 'center',
                    padding: '30px 10px',
                    color: 'var(--text-dim)',
                    fontSize: '0.8rem',
                  }}
                >
                  No tasks in this column
                </div>
              ) : (
                columnTasks.map((task) => (
                  <TaskCard
                    key={task.id}
                    task={task}
                    columns={columns}
                    onMove={onMoveTask}
                    onDelete={onDeleteTask}
                    onClick={onSelectTask}
                    activeTimer={activeTimer}
                    onStartTimer={onStartTimer}
                    onStopTimer={onStopTimer}
                  />
                ))
              )}
            </div>
          </div>
        );
      })}
    </div>
  );
}
