import React, { useState, useEffect } from 'react';
import {
  X,
  Calendar,
  ChevronLeft,
  ChevronRight,
  TrendingDown,
  Clock,
  User,
  CheckCircle2,
  AlertTriangle,
  Flame,
  BarChart2,
} from 'lucide-react';
import { fetchWeeklyTimesheet, fetchProjectBurndown } from '../services/api';

export default function TimesheetModal({
  isOpen,
  onClose,
  project,
  users = [],
  currentUser,
}) {
  const [activeTab, setActiveTab] = useState('timesheet'); // 'timesheet' | 'burndown'
  const [selectedUserId, setSelectedUserId] = useState(currentUser?.id || '');
  const [currentWeekMonday, setCurrentWeekMonday] = useState(() => {
    const d = new Date();
    const day = d.getDay();
    const diff = d.getDate() - day + (day === 0 ? -6 : 1);
    const monday = new Date(d.setDate(diff));
    return monday.toISOString().split('T')[0];
  });

  const [timesheet, setTimesheet] = useState(null);
  const [burndown, setBurndown] = useState(null);
  const [loading, setLoading] = useState(false);
  const [hoveredPoint, setHoveredPoint] = useState(null);

  const isPMOrAdmin =
    currentUser?.primary_role === 'PROJECT_MANAGER' || currentUser?.primary_role === 'ADMIN';

  // Load timesheet
  useEffect(() => {
    if (!isOpen) return;

    async function loadData() {
      setLoading(true);
      try {
        if (activeTab === 'timesheet') {
          const ts = await fetchWeeklyTimesheet(selectedUserId, currentWeekMonday);
          setTimesheet(ts);
        } else {
          if (project?.id) {
            const bd = await fetchProjectBurndown(project.id);
            setBurndown(bd);
          }
        }
      } catch (err) {
        console.error('Error loading timesheet/burndown:', err);
      } finally {
        setLoading(false);
      }
    }

    loadData();
  }, [isOpen, activeTab, selectedUserId, currentWeekMonday, project?.id]);

  if (!isOpen) return null;

  // Week navigation
  const handlePrevWeek = () => {
    const d = new Date(currentWeekMonday);
    d.setDate(d.getDate() - 7);
    setCurrentWeekMonday(d.toISOString().split('T')[0]);
  };

  const handleNextWeek = () => {
    const d = new Date(currentWeekMonday);
    d.setDate(d.getDate() + 7);
    setCurrentWeekMonday(d.toISOString().split('T')[0]);
  };

  // Helper for 7 week dates
  const weekDays = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];
  const weekDates = [];
  const baseD = new Date(currentWeekMonday);
  for (let i = 0; i < 7; i++) {
    const temp = new Date(baseD);
    temp.setDate(temp.getDate() + i);
    weekDates.push(temp.toISOString().split('T')[0]);
  }

  // Burndown calculations
  const totalEst = burndown?.total_estimated_hours || 0;
  const totalLogged = burndown?.total_logged_hours || 0;
  const points = burndown?.points || [];

  return (
    <div className="modal-backdrop">
      <div className="modal-content timesheet-modal-content" style={{ maxWidth: '980px', width: '95%' }}>
        {/* Header */}
        <div className="modal-header">
          <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
            <div className="logo-icon" style={{ background: 'rgba(56, 189, 248, 0.2)' }}>
              <Clock size={20} color="#38bdf8" />
            </div>
            <div>
              <h2 style={{ fontSize: '1.25rem', margin: 0, fontWeight: 700 }}>
                Time Tracking & Analytics
              </h2>
              <div style={{ fontSize: '0.8rem', color: 'var(--text-muted)' }}>
                {project?.name} ({project?.code})
              </div>
            </div>
          </div>
          <button className="close-btn" onClick={onClose}>
            <X size={18} />
          </button>
        </div>

        {/* Tab Switcher */}
        <div className="feasibility-tab-row" style={{ padding: '0 24px', marginTop: '12px' }}>
          <button
            className={`feas-tab-btn ${activeTab === 'timesheet' ? 'active' : ''}`}
            onClick={() => setActiveTab('timesheet')}
          >
            <Calendar size={15} />
            <span>Weekly Timesheet (TSK-029)</span>
          </button>
          <button
            className={`feas-tab-btn ${activeTab === 'burndown' ? 'active' : ''}`}
            onClick={() => setActiveTab('burndown')}
          >
            <TrendingDown size={15} />
            <span>Project Burn-Down Chart</span>
          </button>
        </div>

        {/* Content Body */}
        <div style={{ padding: '20px 24px', maxHeight: '72vh', overflowY: 'auto' }}>
          {activeTab === 'timesheet' && (
            <div>
              {/* Controls bar: Week selector & User filter */}
              <div
                style={{
                  display: 'flex',
                  justifyContent: 'space-between',
                  alignItems: 'center',
                  marginBottom: '16px',
                  flexWrap: 'wrap',
                  gap: '12px',
                }}
              >
                <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <button className="btn-secondary" onClick={handlePrevWeek} title="Previous Week" style={{ padding: '6px 10px' }}>
                    <ChevronLeft size={16} />
                  </button>
                  <div
                    style={{
                      background: 'rgba(255, 255, 255, 0.04)',
                      padding: '6px 14px',
                      borderRadius: '8px',
                      border: '1px solid var(--border)',
                      fontWeight: 600,
                      fontSize: '0.85rem',
                    }}
                  >
                    Week of {currentWeekMonday} to {weekDates[6]}
                  </div>
                  <button className="btn-secondary" onClick={handleNextWeek} title="Next Week" style={{ padding: '6px 10px' }}>
                    <ChevronRight size={16} />
                  </button>
                </div>

                {isPMOrAdmin && users.length > 0 && (
                  <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
                    <User size={15} color="var(--text-muted)" />
                    <select
                      className="form-input"
                      style={{ padding: '6px 12px', width: 'auto', fontSize: '0.85rem' }}
                      value={selectedUserId}
                      onChange={(e) => setSelectedUserId(e.target.value)}
                    >
                      {users.map((u) => (
                        <option key={u.id} value={u.id}>
                          {u.full_name || u.email} ({u.primary_role})
                        </option>
                      ))}
                    </select>
                  </div>
                )}
              </div>

              {/* Summary Cards */}
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '12px', marginBottom: '20px' }}>
                <div className="feas-stat-card">
                  <div className="stat-label">Total Logged This Week</div>
                  <div className="stat-value" style={{ color: '#38bdf8' }}>
                    {timesheet?.total_hours ?? 0} hrs
                  </div>
                  <div className="stat-sub">Across {timesheet?.entries?.length || 0} active tasks</div>
                </div>
                <div className="feas-stat-card">
                  <div className="stat-label">Weekly Target</div>
                  <div className="stat-value" style={{ color: '#10b981' }}>
                    40.0 hrs
                  </div>
                  <div className="stat-sub">
                    {Math.round(((timesheet?.total_hours || 0) / 40.0) * 100)}% of target met
                  </div>
                </div>
                <div className="feas-stat-card">
                  <div className="stat-label">Daily Average</div>
                  <div className="stat-value" style={{ color: '#f59e0b' }}>
                    {timesheet ? (timesheet.total_hours / 5.0).toFixed(1) : 0} hrs/day
                  </div>
                  <div className="stat-sub">Based on 5 working days</div>
                </div>
              </div>

              {/* Timesheet Matrix Table */}
              {loading ? (
                <div style={{ textAlign: 'center', padding: '40px', color: 'var(--text-muted)' }}>
                  Loading timesheet details...
                </div>
              ) : !timesheet?.entries || timesheet.entries.length === 0 ? (
                <div style={{ textAlign: 'center', padding: '40px 20px', background: 'rgba(255, 255, 255, 0.02)', borderRadius: '12px', border: '1px dashed var(--border)' }}>
                  <Clock size={32} color="var(--text-muted)" style={{ margin: '0 auto 12px' }} />
                  <div style={{ fontWeight: 600, color: 'var(--text-main)', marginBottom: '4px' }}>
                    No time logged for this week yet
                  </div>
                  <div style={{ fontSize: '0.85rem', color: 'var(--text-muted)' }}>
                    Use the live timer or manual log button on any task card to log hours.
                  </div>
                </div>
              ) : (
                <div style={{ overflowX: 'auto', border: '1px solid var(--border)', borderRadius: '10px' }}>
                  <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '0.85rem' }}>
                    <thead>
                      <tr style={{ background: 'rgba(255, 255, 255, 0.04)', borderBottom: '1px solid var(--border)', textAlign: 'left' }}>
                        <th style={{ padding: '10px 14px', width: '30%' }}>Task & Project</th>
                        {weekDays.map((d, i) => (
                          <th key={d} style={{ padding: '10px 8px', textAlign: 'center', width: '8%' }}>
                            <div>{d}</div>
                            <div style={{ fontSize: '0.7rem', color: 'var(--text-muted)', fontWeight: 400 }}>
                              {weekDates[i]?.slice(5)}
                            </div>
                          </th>
                        ))}
                        <th style={{ padding: '10px 14px', textAlign: 'right', width: '10%' }}>Total</th>
                      </tr>
                    </thead>
                    <tbody>
                      {timesheet.entries.map((entry) => (
                        <tr key={entry.task_id} style={{ borderBottom: '1px solid rgba(255, 255, 255, 0.03)' }}>
                          <td style={{ padding: '10px 14px' }}>
                            <div style={{ fontWeight: 600, color: 'var(--text-main)' }}>{entry.task_title}</div>
                            <div style={{ fontSize: '0.72rem', color: 'var(--text-muted)' }}>{entry.project_name}</div>
                          </td>
                          {weekDates.map((dateStr) => {
                            const hrs = entry.daily_hours?.[dateStr] || 0;
                            return (
                              <td key={dateStr} style={{ padding: '10px 8px', textAlign: 'center', color: hrs > 0 ? '#38bdf8' : 'rgba(255, 255, 255, 0.2)', fontWeight: hrs > 0 ? 700 : 400 }}>
                                {hrs > 0 ? `${hrs}h` : '—'}
                              </td>
                            );
                          })}
                          <td style={{ padding: '10px 14px', textAlign: 'right', fontWeight: 700, color: '#10b981' }}>
                            {entry.total_hours}h
                          </td>
                        </tr>
                      ))}
                      {/* Daily Totals Footer */}
                      <tr style={{ background: 'rgba(255, 255, 255, 0.05)', fontWeight: 700 }}>
                        <td style={{ padding: '12px 14px' }}>Daily Total</td>
                        {weekDates.map((dateStr) => {
                          const total = timesheet.daily_totals?.[dateStr] || 0;
                          return (
                            <td key={dateStr} style={{ padding: '12px 8px', textAlign: 'center', color: total > 8 ? '#f59e0b' : total > 0 ? '#38bdf8' : 'var(--text-muted)' }}>
                              {total > 0 ? `${total}h` : '0h'}
                            </td>
                          );
                        })}
                        <td style={{ padding: '12px 14px', textAlign: 'right', color: '#10b981', fontSize: '1rem' }}>
                          {timesheet.total_hours}h
                        </td>
                      </tr>
                    </tbody>
                  </table>
                </div>
              )}
            </div>
          )}

          {activeTab === 'burndown' && (
            <div>
              {/* Burndown Summary Metrics */}
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 1fr))', gap: '12px', marginBottom: '20px' }}>
                <div className="feas-stat-card">
                  <div className="stat-label">Total Estimated Effort</div>
                  <div className="stat-value" style={{ color: '#38bdf8' }}>
                    {totalEst} hrs
                  </div>
                  <div className="stat-sub">Across all project tasks</div>
                </div>
                <div className="feas-stat-card">
                  <div className="stat-label">Total Logged Effort</div>
                  <div className="stat-value" style={{ color: '#10b981' }}>
                    {totalLogged} hrs
                  </div>
                  <div className="stat-sub">
                    {totalEst > 0 ? Math.round((totalLogged / totalEst) * 100) : 0}% work completed
                  </div>
                </div>
                <div className="feas-stat-card">
                  <div className="stat-label">Remaining Effort</div>
                  <div className="stat-value" style={{ color: '#f59e0b' }}>
                    {Math.max(0, totalEst - totalLogged)} hrs
                  </div>
                  <div className="stat-sub">To target project completion</div>
                </div>
              </div>

              {/* Burndown SVG Chart */}
              {loading ? (
                <div style={{ textAlign: 'center', padding: '40px', color: 'var(--text-muted)' }}>
                  Calculating burndown trajectory...
                </div>
              ) : points.length === 0 ? (
                <div style={{ textAlign: 'center', padding: '40px', color: 'var(--text-muted)' }}>
                  No burndown points available for this timeline.
                </div>
              ) : (
                <div
                  style={{
                    background: 'rgba(255, 255, 255, 0.02)',
                    border: '1px solid var(--border)',
                    borderRadius: '12px',
                    padding: '20px',
                    position: 'relative',
                  }}
                >
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '14px' }}>
                    <div style={{ display: 'flex', alignItems: 'center', gap: '16px', fontSize: '0.8rem' }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                        <span style={{ width: '14px', height: '3px', background: '#38bdf8', borderTop: '2px dashed #38bdf8', display: 'inline-block' }} />
                        <span style={{ color: 'var(--text-muted)' }}>Ideal Burndown (Linear)</span>
                      </div>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                        <span style={{ width: '14px', height: '3px', background: '#10b981', display: 'inline-block' }} />
                        <span style={{ color: 'var(--text-muted)' }}>Actual Remaining Effort</span>
                      </div>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '6px' }}>
                        <span style={{ width: '10px', height: '10px', background: 'rgba(56, 189, 248, 0.3)', display: 'inline-block', borderRadius: '2px' }} />
                        <span style={{ color: 'var(--text-muted)' }}>Daily Logged Effort</span>
                      </div>
                    </div>
                  </div>

                  {/* SVG Canvas */}
                  <div style={{ width: '100%', height: '260px' }}>
                    <svg
                      viewBox="0 0 800 240"
                      style={{ width: '100%', height: '100%', overflow: 'visible' }}
                    >
                      {/* Grid Lines */}
                      <line x1="40" y1="20" x2="780" y2="20" stroke="rgba(255,255,255,0.06)" strokeDasharray="4" />
                      <line x1="40" y1="75" x2="780" y2="75" stroke="rgba(255,255,255,0.06)" strokeDasharray="4" />
                      <line x1="40" y1="130" x2="780" y2="130" stroke="rgba(255,255,255,0.06)" strokeDasharray="4" />
                      <line x1="40" y1="185" x2="780" y2="185" stroke="rgba(255,255,255,0.06)" strokeDasharray="4" />
                      <line x1="40" y1="200" x2="780" y2="200" stroke="rgba(255,255,255,0.2)" />

                      {/* Y-Axis Labels */}
                      <text x="32" y="24" fill="rgba(255,255,255,0.4)" fontSize="10" textAnchor="end">{totalEst}h</text>
                      <text x="32" y="112" fill="rgba(255,255,255,0.4)" fontSize="10" textAnchor="end">{Math.round(totalEst / 2)}h</text>
                      <text x="32" y="204" fill="rgba(255,255,255,0.4)" fontSize="10" textAnchor="end">0h</text>

                      {/* Daily Logged Effort Bars */}
                      {points.map((pt, i) => {
                        const x = 50 + (i / Math.max(1, points.length - 1)) * 710;
                        const barHeight = totalEst > 0 ? (pt.logged_hours_on_day / totalEst) * 160 : 0;
                        const y = 200 - barHeight;
                        return (
                          <rect
                            key={`bar-${i}`}
                            x={x - 4}
                            y={y}
                            width="8"
                            height={Math.max(2, barHeight)}
                            fill="rgba(56, 189, 248, 0.3)"
                            rx="2"
                          />
                        );
                      })}

                      {/* Ideal Line (Dashed Cyan) */}
                      <polyline
                        fill="none"
                        stroke="#38bdf8"
                        strokeWidth="2"
                        strokeDasharray="5,5"
                        points={points
                          .map((pt, i) => {
                            const x = 50 + (i / Math.max(1, points.length - 1)) * 710;
                            const y = totalEst > 0 ? 20 + ((totalEst - pt.ideal_remaining_hours) / totalEst) * 180 : 200;
                            return `${x},${y}`;
                          })
                          .join(' ')}
                      />

                      {/* Actual Line (Solid Emerald) */}
                      <polyline
                        fill="none"
                        stroke="#10b981"
                        strokeWidth="3"
                        strokeLinecap="round"
                        points={points
                          .map((pt, i) => {
                            const x = 50 + (i / Math.max(1, points.length - 1)) * 710;
                            const y = totalEst > 0 ? 20 + ((totalEst - pt.actual_remaining_hours) / totalEst) * 180 : 200;
                            return `${x},${y}`;
                          })
                          .join(' ')}
                      />

                      {/* Interactive Point Nodes */}
                      {points.map((pt, i) => {
                        const x = 50 + (i / Math.max(1, points.length - 1)) * 710;
                        const y = totalEst > 0 ? 20 + ((totalEst - pt.actual_remaining_hours) / totalEst) * 180 : 200;
                        return (
                          <circle
                            key={`node-${i}`}
                            cx={x}
                            cy={y}
                            r="4"
                            fill="#10b981"
                            stroke="#0f172a"
                            strokeWidth="2"
                            style={{ cursor: 'pointer' }}
                            onMouseEnter={() => setHoveredPoint(pt)}
                            onMouseLeave={() => setHoveredPoint(null)}
                          />
                        );
                      })}
                    </svg>
                  </div>

                  {/* Tooltip on Hover */}
                  {hoveredPoint && (
                    <div
                      style={{
                        position: 'absolute',
                        top: '20px',
                        right: '30px',
                        background: '#1e293b',
                        border: '1px solid #38bdf8',
                        borderRadius: '8px',
                        padding: '10px 14px',
                        fontSize: '0.8rem',
                        boxShadow: '0 4px 12px rgba(0,0,0,0.4)',
                        zIndex: 10,
                      }}
                    >
                      <div style={{ fontWeight: 700, color: 'var(--text-main)', marginBottom: '4px' }}>
                        Date: {hoveredPoint.date} (Day {hoveredPoint.day_index})
                      </div>
                      <div style={{ color: '#10b981' }}>
                        Actual Remaining: <b>{hoveredPoint.actual_remaining_hours}h</b>
                      </div>
                      <div style={{ color: '#38bdf8' }}>
                        Ideal Remaining: <b>{hoveredPoint.ideal_remaining_hours}h</b>
                      </div>
                      <div style={{ color: '#f59e0b', marginTop: '2px' }}>
                        Logged on this day: <b>{hoveredPoint.logged_hours_on_day}h</b>
                      </div>
                    </div>
                  )}

                  {/* X-Axis Dates */}
                  <div style={{ display: 'flex', justifyContent: 'space-between', marginTop: '10px', fontSize: '0.75rem', color: 'var(--text-muted)' }}>
                    <span>Start: {burndown.start_date}</span>
                    <span>Target Deadline: {burndown.target_end_date}</span>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
