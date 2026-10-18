import React, { useState } from 'react';
import {
  X,
  Gauge,
  TrendingUp,
  AlertTriangle,
  CheckCircle2,
  Calendar,
  Clock,
  Users,
  GitCommit,
  Sliders,
  Sparkles,
  ArrowRight,
  ShieldAlert,
} from 'lucide-react';
import { simulateProjectFeasibility } from '../services/api';

export default function FeasibilityModal({
  isOpen,
  onClose,
  feasibility,
  project,
  tasks,
  onRefresh,
}) {
  const [activeTab, setActiveTab] = useState('overview'); // 'overview', 'cpm', 'simulate'
  const [addDevs, setAddDevs] = useState(1);
  const [addQA, setAddQA] = useState(0);
  const [addBA, setAddBA] = useState(0);
  const [overtimeHrs, setOvertimeHrs] = useState(0);
  const [extendDays, setExtendDays] = useState(0);

  const [simResult, setSimResult] = useState(null);
  const [isSimulating, setIsSimulating] = useState(false);
  const [simError, setSimError] = useState('');

  if (!isOpen || !feasibility) return null;

  const score = feasibility.feasibility_score || 0;
  const isCanMeet = feasibility.can_meet_deadline;
  const daysBuffer = feasibility.days_ahead_or_behind || 0;

  let gaugeColor = '#10b981';
  if (score < 60 || !isCanMeet) {
    gaugeColor = '#ef4444';
  } else if (score < 80 || feasibility.feasibility_status === 'AT_RISK') {
    gaugeColor = '#f59e0b';
  }

  const handleRunSimulation = async (e) => {
    e.preventDefault();
    setIsSimulating(true);
    setSimError('');

    try {
      const res = await simulateProjectFeasibility(project.id, {
        additional_devs: parseInt(addDevs, 10),
        additional_qa: parseInt(addQA, 10),
        additional_ba: parseInt(addBA, 10),
        overtime_hours_per_day: parseFloat(overtimeHrs),
        extend_deadline_days: parseInt(extendDays, 10),
      });
      setSimResult(res);
    } catch (err) {
      setSimError(err.message || 'Simulation failed');
    } finally {
      setIsSimulating(false);
    }
  };

  const roles = feasibility.roles || {};
  const bottlenecks = feasibility.individual_bottlenecks || [];
  const criticalPathTasks = (feasibility.critical_path || []).map((id) => {
    return tasks.find((t) => t.id === id) || { id, title: id, required_role: 'DEVELOPER' };
  });

  return (
    <div className="modal-backdrop" onClick={onClose}>
      <div className="modal-card" style={{ width: '780px' }} onClick={(e) => e.stopPropagation()}>
        <div className="modal-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <Gauge size={22} color={gaugeColor} />
            <div>
              <h2>Deadline Feasibility & CPM Engine (Phase 5)</h2>
              <div style={{ fontSize: '0.75rem', color: 'var(--text-dim)' }}>
                Mathematical capacity modeling, Critical Path Method (CPM), and in-memory What-If simulator
              </div>
            </div>
          </div>
          <button className="modal-close" onClick={onClose}>
            <X size={18} />
          </button>
        </div>

        <div className="modal-body">
          {/* Executive Top Banner: Feasibility Score & Forecast */}
          <div className="feasibility-hero-card">
            <div className="feasibility-score-ring" style={{ borderColor: gaugeColor }}>
              <span className="score-num" style={{ color: gaugeColor }}>
                {score}%
              </span>
              <span className="score-label">Feasibility</span>
            </div>

            <div className="feasibility-hero-meta">
              <div className="hero-status-row">
                <span className={`feasibility-badge status-${feasibility.feasibility_status?.toLowerCase() || 'ontrack'}`}>
                  ● {feasibility.feasibility_status?.replace('_', ' ') || 'ON TRACK'}
                </span>
                <span className="hero-verdict">
                  {isCanMeet ? '✓ Can Meet Deadline' : '⚠ Deadline at Risk'}
                </span>
              </div>

              <div className="hero-forecast-grid">
                <div className="forecast-item">
                  <Calendar size={13} color="#818cf8" />
                  <span>
                    Projected Finish:{' '}
                    <strong>
                      {feasibility.projected_completion_date
                        ? feasibility.projected_completion_date.split('T')[0]
                        : 'N/A'}
                    </strong>
                  </span>
                </div>

                <div className="forecast-item">
                  <Clock size={13} color={daysBuffer >= 0 ? '#10b981' : '#ef4444'} />
                  <span>
                    Schedule Buffer:{' '}
                    <strong style={{ color: daysBuffer >= 0 ? '#10b981' : '#ef4444' }}>
                      {daysBuffer >= 0 ? `+${daysBuffer} days ahead` : `${daysBuffer} days behind`}
                    </strong>
                  </span>
                </div>

                <div className="forecast-item">
                  <Users size={13} color="#f59e0b" />
                  <span>
                    Remaining Work: <strong>{feasibility.total_remaining_hours || 0}h</strong> vs{' '}
                    <strong>{feasibility.total_capacity_hours || 0}h</strong> Capacity
                  </span>
                </div>
              </div>
            </div>
          </div>

          {/* Navigation Tabs */}
          <div className="tabs-header" style={{ marginTop: '16px' }}>
            <button
              className={`tab-btn ${activeTab === 'overview' ? 'active' : ''}`}
              onClick={() => setActiveTab('overview')}
            >
              <TrendingUp size={14} />
              <span>Role Capacities & Bottlenecks</span>
            </button>
            <button
              className={`tab-btn ${activeTab === 'cpm' ? 'active' : ''}`}
              onClick={() => setActiveTab('cpm')}
            >
              <GitCommit size={14} />
              <span>Critical Path Method (CPM)</span>
            </button>
            <button
              className={`tab-btn ${activeTab === 'simulate' ? 'active' : ''}`}
              onClick={() => setActiveTab('simulate')}
            >
              <Sliders size={14} />
              <span>What-If Simulator (TSK-025)</span>
            </button>
          </div>

          {/* TAB 1: ROLE CAPACITIES & BOTTLENECK ANALYSIS */}
          {activeTab === 'overview' && (
            <div className="tab-pane">
              <h4 style={{ fontSize: '0.88rem', color: '#fff', marginBottom: '10px' }}>
                Functional Role Capacity vs. Remaining Effort (TSK-022)
              </h4>

              <div className="role-metrics-grid">
                {Object.keys(roles).map((roleKey) => {
                  const m = roles[roleKey];
                  const isOk = m.status === 'ON_TRACK';
                  const isWarn = m.status === 'AT_RISK';
                  const barColor = isOk ? '#10b981' : isWarn ? '#f59e0b' : '#ef4444';

                  return (
                    <div key={roleKey} className="role-metric-card">
                      <div className="role-metric-header">
                        <span className="role-metric-title">{m.role}</span>
                        <span
                          className="role-metric-status"
                          style={{
                            color: barColor,
                            background: `${barColor}1a`,
                          }}
                        >
                          {m.status?.replace('_', ' ')}
                        </span>
                      </div>

                      <div className="role-metric-hours">
                        <div>
                          <span className="hrs-label">Remaining Work:</span>
                          <span className="hrs-val">{m.remaining_task_hours}h</span>
                        </div>
                        <div>
                          <span className="hrs-label">Available Cap:</span>
                          <span className="hrs-val">{m.available_capacity_hours}h</span>
                        </div>
                      </div>

                      <div className="role-util-bar-bg">
                        <div
                          className="role-util-bar-fill"
                          style={{
                            width: `${Math.min(100, m.utilization_pct || 0)}%`,
                            backgroundColor: barColor,
                          }}
                        />
                      </div>
                      <div className="role-metric-footer">
                        <span>Utilization: {m.utilization_pct || 0}%</span>
                        <span style={{ color: m.buffer_or_deficit_hours >= 0 ? '#10b981' : '#ef4444' }}>
                          {m.buffer_or_deficit_hours >= 0 ? `+${m.buffer_or_deficit_hours}h buffer` : `${m.buffer_or_deficit_hours}h deficit`}
                        </span>
                      </div>
                    </div>
                  );
                })}
              </div>

              {/* Individual Bottleneck Section (TSK-023) */}
              <div style={{ marginTop: '20px' }}>
                <h4 style={{ fontSize: '0.88rem', color: '#fff', marginBottom: '10px' }}>
                  Individual Developer Bottleneck Analyzer (TSK-023)
                </h4>

                {bottlenecks.length === 0 ? (
                  <div className="alert-success" style={{ margin: 0 }}>
                    <CheckCircle2 size={16} />
                    <span>Zero developer-level bottlenecks detected. Workload is balanced across all team members.</span>
                  </div>
                ) : (
                  <div className="bottlenecks-list">
                    {bottlenecks.map((b) => (
                      <div key={b.user_id} className="bottleneck-card">
                        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                          <AlertTriangle size={16} color="#ef4444" />
                          <span style={{ fontWeight: '700', color: '#fff', fontSize: '0.88rem' }}>
                            {b.name} ({b.role}) — {b.deficit_hours}h Deficit
                          </span>
                        </div>
                        <div style={{ fontSize: '0.82rem', color: '#fca5a5', marginTop: '4px' }}>
                          Allocated: {b.allocated_task_hours}h vs Available: {b.available_capacity_hours}h
                        </div>
                        <div style={{ fontSize: '0.78rem', color: '#cbd5e1', marginTop: '6px' }}>
                          💡 <strong>Recommendation:</strong> {b.recommendation}
                        </div>
                      </div>
                    ))}
                  </div>
                )}
              </div>
            </div>
          )}

          {/* TAB 2: CRITICAL PATH METHOD (CPM) */}
          {activeTab === 'cpm' && (
            <div className="tab-pane">
              <div style={{ fontSize: '0.85rem', color: 'var(--text-dim)', marginBottom: '14px' }}>
                Calculates early start (ES), early finish (EF), and zero-slack critical path across tasks and
                Finish-to-Start dependencies. Any slippage on this path directly delays delivery.
              </div>

              {criticalPathTasks.length === 0 ? (
                <div className="empty-state-small">All tasks are completed or no dependencies linked.</div>
              ) : (
                <div className="cpm-sequence-strip">
                  {criticalPathTasks.map((t, idx) => (
                    <React.Fragment key={t.id}>
                      <div className="cpm-node-card">
                        <div className="cpm-node-header">
                          <span className="cpm-node-role">{t.required_role || 'DEV'}</span>
                          <span className="cpm-node-hours">{t.estimated_hours || 0}h</span>
                        </div>
                        <div className="cpm-node-title">{t.title}</div>
                        <div className="cpm-node-badge">Slack: 0 days</div>
                      </div>
                      {idx < criticalPathTasks.length - 1 && (
                        <div className="cpm-arrow">
                          <ArrowRight size={16} color="#6366f1" />
                        </div>
                      )}
                    </React.Fragment>
                  ))}
                </div>
              )}

              <div className="cpm-summary-box">
                <div className="cpm-summary-item">
                  <span className="cpm-sum-label">Critical Path Tasks</span>
                  <span className="cpm-sum-val">{criticalPathTasks.length} tasks</span>
                </div>
                <div className="cpm-summary-item">
                  <span className="cpm-sum-label">Projected Finish Date</span>
                  <span className="cpm-sum-val highlight">
                    {feasibility.projected_completion_date ? feasibility.projected_completion_date.split('T')[0] : 'N/A'}
                  </span>
                </div>
                <div className="cpm-summary-item">
                  <span className="cpm-sum-label">Total Calendar Duration</span>
                  <span className="cpm-sum-val">{feasibility.remaining_working_days || 0} working days</span>
                </div>
              </div>
            </div>
          )}

          {/* TAB 3: WHAT-IF SIMULATOR (TSK-025) */}
          {activeTab === 'simulate' && (
            <div className="tab-pane">
              <div style={{ fontSize: '0.85rem', color: 'var(--text-dim)', marginBottom: '14px' }}>
                Simulate adding staff, overtime, or deadline extensions. Executes on cloned in-memory structs with{' '}
                <strong>zero database mutation</strong> in &lt; 50ms.
              </div>

              <form onSubmit={handleRunSimulation} className="simulation-form">
                <div className="simulation-inputs-grid">
                  <div className="sim-group">
                    <label>Additional Developers</label>
                    <div className="stepper-wrap">
                      <button type="button" onClick={() => setAddDevs(Math.max(0, addDevs - 1))}>-</button>
                      <span>+{addDevs}</span>
                      <button type="button" onClick={() => setAddDevs(addDevs + 1)}>+</button>
                    </div>
                  </div>

                  <div className="sim-group">
                    <label>Additional QA Engineers</label>
                    <div className="stepper-wrap">
                      <button type="button" onClick={() => setAddQA(Math.max(0, addQA - 1))}>-</button>
                      <span>+{addQA}</span>
                      <button type="button" onClick={() => setAddQA(addQA + 1)}>+</button>
                    </div>
                  </div>

                  <div className="sim-group">
                    <label>Additional Business Analysts</label>
                    <div className="stepper-wrap">
                      <button type="button" onClick={() => setAddBA(Math.max(0, addBA - 1))}>-</button>
                      <span>+{addBA}</span>
                      <button type="button" onClick={() => setAddBA(addBA + 1)}>+</button>
                    </div>
                  </div>

                  <div className="sim-group">
                    <label>Overtime (hrs/day)</label>
                    <div className="stepper-wrap">
                      <button type="button" onClick={() => setOvertimeHrs(Math.max(0, overtimeHrs - 0.5))}>-</button>
                      <span>+{overtimeHrs}h</span>
                      <button type="button" onClick={() => setOvertimeHrs(overtimeHrs + 0.5)}>+</button>
                    </div>
                  </div>

                  <div className="sim-group" style={{ gridColumn: 'span 2' }}>
                    <label>Extend Project Deadline (+{extendDays} calendar days)</label>
                    <input
                      type="range"
                      min="0"
                      max="30"
                      step="5"
                      value={extendDays}
                      onChange={(e) => setExtendDays(parseInt(e.target.value, 10))}
                    />
                  </div>
                </div>

                <div style={{ marginTop: '14px', display: 'flex', justifyContent: 'flex-end' }}>
                  <button type="submit" className="btn-primary" disabled={isSimulating}>
                    <Sparkles size={15} />
                    <span>{isSimulating ? 'Simulating (<50ms)...' : 'Run What-If Simulation'}</span>
                  </button>
                </div>
              </form>

              {simError && (
                <div className="timeline-conflict-alert" style={{ marginTop: '14px' }}>
                  <span>{simError}</span>
                </div>
              )}

              {/* Simulation Result Comparison */}
              {simResult && (
                <div className="simulation-result-card">
                  <div className="sim-result-header">
                    <Sparkles size={18} color="#a855f7" />
                    <h4>What-If Simulation Outcome</h4>
                  </div>

                  <div className="sim-comparison-grid">
                    <div className="sim-compare-col">
                      <span className="compare-title">Baseline State</span>
                      <div className="compare-score" style={{ color: simResult.original_score < 70 ? '#ef4444' : '#10b981' }}>
                        {simResult.original_score}%
                      </div>
                      <div className="compare-detail">
                        Buffer: {simResult.original_days_buffer} days
                      </div>
                      <div className="compare-detail">
                        Finish: {simResult.original_projected_date?.split('T')[0]}
                      </div>
                    </div>

                    <div className="sim-compare-arrow">
                      <ArrowRight size={22} color="#a855f7" />
                    </div>

                    <div className="sim-compare-col highlight">
                      <span className="compare-title">Simulated State</span>
                      <div className="compare-score" style={{ color: '#10b981' }}>
                        {simResult.simulated_score}%
                      </div>
                      <div className="compare-detail">
                        Buffer: +{simResult.simulated_days_buffer} days
                      </div>
                      <div className="compare-detail">
                        Finish: {simResult.simulated_projected_date?.split('T')[0]}
                      </div>
                    </div>
                  </div>

                  <div className="sim-recommendation-box">
                    <span>💡 {simResult.recommendation_summary}</span>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>

        <div className="modal-footer">
          <button className="btn-secondary" onClick={onClose}>
            Close
          </button>
        </div>
      </div>
    </div>
  );
}
