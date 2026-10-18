import React, { useState } from 'react';
import { X } from 'lucide-react';

export default function CreateTaskModal({ isOpen, onClose, columns, users, defaultColumnId, onCreate }) {
  const [title, setTitle] = useState('');
  const [description, setDescription] = useState('');
  const [columnId, setColumnId] = useState(defaultColumnId || 'col-todo');
  const [priority, setPriority] = useState('Medium');
  const [assignee, setAssignee] = useState(users[0]?.name || 'Alex Morgan');
  const [dueDate, setDueDate] = useState('Oct 28');

  if (!isOpen) return null;

  const handleSubmit = (e) => {
    e.preventDefault();
    if (!title.trim()) return;

    onCreate({
      title: title.trim(),
      description: description.trim(),
      column_id: columnId,
      priority,
      assignee,
      due_date: dueDate,
      project_id: 'proj-1',
    });

    setTitle('');
    setDescription('');
    onClose();
  };

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal-card" onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <h2>Create New Task</h2>
          <button className="modal-close" onClick={onClose}>
            <X size={18} />
          </button>
        </div>

        <form onSubmit={handleSubmit}>
          <div className="modal-body">
            <div className="form-group">
              <label>Task Title *</label>
              <input
                type="text"
                placeholder="e.g. Implement OAuth login flow"
                value={title}
                onChange={(e) => setTitle(e.target.value)}
                autoFocus
                required
              />
            </div>

            <div className="form-group">
              <label>Description</label>
              <textarea
                rows="3"
                placeholder="Add more details or acceptance criteria..."
                value={description}
                onChange={(e) => setDescription(e.target.value)}
              />
            </div>

            <div className="form-row">
              <div className="form-group">
                <label>Status / Column</label>
                <select value={columnId} onChange={(e) => setColumnId(e.target.value)}>
                  {columns.map((c) => (
                    <option key={c.id} value={c.id}>
                      {c.title}
                    </option>
                  ))}
                </select>
              </div>

              <div className="form-group">
                <label>Priority</label>
                <select value={priority} onChange={(e) => setPriority(e.target.value)}>
                  <option value="Low">Low</option>
                  <option value="Medium">Medium</option>
                  <option value="High">High</option>
                  <option value="Urgent">Urgent</option>
                </select>
              </div>
            </div>

            <div className="form-row">
              <div className="form-group">
                <label>Assignee</label>
                <select value={assignee} onChange={(e) => setAssignee(e.target.value)}>
                  {users.map((u) => (
                    <option key={u.id} value={u.name}>
                      {u.name} ({u.role})
                    </option>
                  ))}
                </select>
              </div>

              <div className="form-group">
                <label>Due Date</label>
                <input
                  type="text"
                  placeholder="e.g. Oct 30"
                  value={dueDate}
                  onChange={(e) => setDueDate(e.target.value)}
                />
              </div>
            </div>
          </div>

          <div className="modal-footer">
            <button type="button" className="btn-secondary" onClick={onClose}>
              Cancel
            </button>
            <button type="submit" className="btn-primary">
              Create Task
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
