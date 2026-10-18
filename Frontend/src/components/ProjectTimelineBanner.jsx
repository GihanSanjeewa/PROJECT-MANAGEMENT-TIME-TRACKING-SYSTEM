import React from 'react';
import { Calendar, Users, Settings, Plus, Clock, Gauge, TrendingDown } from 'lucide-react';

export default function ProjectTimelineBanner({
  project,
  capacity,
  feasibility,
  onOpenRoster,
  onOpenSettings,
  onOpenCreateProject,
  onOpenWorkload,
  onOpenFeasibility,
  onOpenTimesheet,
  currentUser,
}) {
  const isPMOrAdmin =
    currentUser?.primary_role === 'PROJECT_MANAGER' || currentUser?.primary_role === 'ADMIN';

  const roles = capacity?.roles || {};
  const devHours = roles.DEVELOPER?.total_capacity_hours || 0;
  const qaHours = roles.QA?.total_capacity_hours || 0;
  const baHours = roles.BA?.total_capacity_hours || 0;

  const score = feasibility?.feasibility_score ?? 94;
  const isCanMeet = feasibility?.can_meet_deadline ?? true;
  const bufferDays = feasibility?.days_ahead_or_behind ?? 15;
  const scoreColor = !isCanMeet || score < 60 ? '#ef4444' : score < 80 ? '#f59e0b' : '#10b981';

  return (
    <div className="project-banner-wrapper">
      <div className="project-banner-top">
        <div className="project-main-meta">
          <div className="project-title-row">
            <h1>{project.name}</h1>
            <span className="project-code-badge">{project.code}</span>
            <span className={`project-status-pill status-${project.status?.toLowerCase() || 'active'}`}>
              ● {project.status || 'ACTIVE'}
            </span>
          </div>
          <p className="project-desc-text">{project.description}</p>
        </div>

        <div className="project-banner-actions">
          <button className="btn-secondary" onClick={onOpenTimesheet} title="Weekly Timesheet & Project Burndown Chart">
            <TrendingDown size={15} color="#38bdf8" />
            <span>Timesheet & Burndown</span>
          </button>

          <button className="btn-secondary btn-feasibility-action" onClick={onOpenFeasibility} title="Deadline Feasibility & What-If Simulator">
            <Gauge size={15} color={scoreColor} />
            <span>Feasibility ({score}%)</span>
          </button>

          <button className="btn-secondary" onClick={onOpenWorkload} title="Developer Daily Workload & Overload Detector">
            <Clock size={15} color="#f59e0b" />
            <span>Workload</span>
          </button>

          <button className="btn-secondary" onClick={onOpenRoster} title="View team members & capacity by role">
            <Users size={15} />
            <span>Team Roster</span>
          </button>

          {isPMOrAdmin && (
            <button className="btn-secondary" onClick={onOpenSettings} title="Edit project timeline boundaries">
              <Settings size={15} />
              <span>Timeline Settings</span>
            </button>
          )}

          {isPMOrAdmin && (
            <button className="btn-primary" onClick={onOpenCreateProject} title="Create new project">
              <Plus size={15} />
              <span>New Project</span>
            </button>
          )}
        </div>
      </div>

      <div className="timeline-metrics-strip">
        <div className="metric-chip">
          <Calendar size={14} color="#818cf8" />
          <span className="metric-label">Timeline:</span>
          <span className="metric-val">
            {project.start_date ? new Date(project.start_date).toLocaleDateString() : 'Nov 01'} —{' '}
            {project.target_end_date ? new Date(project.target_end_date).toLocaleDateString() : 'Dec 31'}
          </span>
        </div>

        <div className="metric-chip">
          <Clock size={14} color="#10b981" />
          <span className="metric-label">Net Working Days:</span>
          <span className="metric-val highlight">
            {capacity?.total_working_days || 44} Days (Excl. Weekends/Holidays)
          </span>
        </div>

        <div
          className="metric-chip feasibility-chip-clickable"
          onClick={onOpenFeasibility}
          title="Click to view feasibility details & run What-If simulations"
        >
          <Gauge size={14} color={scoreColor} />
          <span className="metric-label">Feasibility:</span>
          <span className="metric-val" style={{ color: scoreColor, fontWeight: '700' }}>
            {score}% {bufferDays >= 0 ? `(+${bufferDays}d buffer)` : `(${bufferDays}d behind)`}
          </span>
        </div>

        <div className="role-capacity-pill-group">
          <div className="capacity-pill dev" title="Available Developer capacity">
            <span className="pill-dot" />
            <span>Dev: {devHours}h</span>
          </div>
          <div className="capacity-pill qa" title="Available QA Engineer capacity">
            <span className="pill-dot" />
            <span>QA: {qaHours}h</span>
          </div>
          <div className="capacity-pill ba" title="Available Business Analyst capacity">
            <span className="pill-dot" />
            <span>BA: {baHours}h</span>
          </div>
        </div>
      </div>
    </div>
  );
}
