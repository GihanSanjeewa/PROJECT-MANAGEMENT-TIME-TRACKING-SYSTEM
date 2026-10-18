import React, { useState } from 'react';
import { Layers, Lock, Mail, ArrowRight, ShieldCheck, UserCheck, AlertCircle, Sparkles } from 'lucide-react';
import { login } from '../services/api';

export default function LoginPage({ onLoginSuccess }) {
  const [email, setEmail] = useState('alex.pm@pulseflow.io');
  const [password, setPassword] = useState('SecurePass123!');
  const [isLoading, setIsLoading] = useState(false);
  const [errorMsg, setErrorMsg] = useState('');

  const demoAccounts = [
    {
      roleTitle: 'Project Manager',
      email: 'alex.pm@pulseflow.io',
      name: 'Alex Morgan',
      badge: 'PROJECT_MANAGER',
      badgeColor: '#a855f7',
      desc: 'Full project timeline, budget & team allocation control',
    },
    {
      roleTitle: 'Developer',
      email: 'jordan.dev@pulseflow.io',
      name: 'Jordan Lee',
      badge: 'DEVELOPER',
      badgeColor: '#38bdf8',
      desc: 'Task execution, Kanban board & live time tracking',
    },
    {
      roleTitle: 'QA Engineer',
      email: 'maya.qa@pulseflow.io',
      name: 'Maya Patel',
      badge: 'QA_ENGINEER',
      badgeColor: '#10b981',
      desc: 'Quality assurance, verification & test execution',
    },
    {
      roleTitle: 'Business Analyst',
      email: 'taylor.ba@pulseflow.io',
      name: 'Taylor Kim',
      badge: 'BUSINESS_ANALYST',
      badgeColor: '#f59e0b',
      desc: 'Requirements, feasibility analysis & backlog grooming',
    },
  ];

  const handleQuickSelect = (demoEmail) => {
    setEmail(demoEmail);
    setPassword('SecurePass123!');
    setErrorMsg('');
  };

  const handleSubmit = async (e) => {
    e.preventDefault();
    setErrorMsg('');
    setIsLoading(true);

    try {
      const user = await login(email, password);
      onLoginSuccess(user);
    } catch (err) {
      setErrorMsg(err.message || 'Authentication failed. Please check credentials.');
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <div className="login-page-container">
      <div className="login-backdrop-glow" />

      <div className="login-card-wrapper">
        <div className="login-brand-header">
          <div className="login-logo-badge">
            <Layers size={24} color="#fff" />
          </div>
          <h1>PulseFlow</h1>
          <p className="login-subtitle">
            Enterprise Project Management & Deadline Feasibility System
          </p>
        </div>

        {errorMsg && (
          <div className="login-error-alert">
            <AlertCircle size={18} />
            <span>{errorMsg}</span>
          </div>
        )}

        <form onSubmit={handleSubmit} className="login-form">
          <div className="form-group">
            <label>
              <Mail size={14} /> Work Email
            </label>
            <input
              type="email"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
              placeholder="e.g. alex.pm@pulseflow.io"
              required
            />
          </div>

          <div className="form-group">
            <label>
              <Lock size={14} /> Password
            </label>
            <input
              type="password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••••••"
              required
            />
          </div>

          <button type="submit" className="login-submit-btn" disabled={isLoading}>
            {isLoading ? (
              <span>Authenticating...</span>
            ) : (
              <>
                <span>Sign In with Credentials</span>
                <ArrowRight size={16} />
              </>
            )}
          </button>
        </form>

        <div className="demo-accounts-section">
          <div className="demo-accounts-title">
            <Sparkles size={14} color="#818cf8" />
            <span>Or Quick-Login as Pre-Seeded Team Role:</span>
          </div>

          <div className="demo-cards-grid">
            {demoAccounts.map((acc) => (
              <div
                key={acc.email}
                className={`demo-role-card ${email === acc.email ? 'active' : ''}`}
                onClick={() => handleQuickSelect(acc.email)}
              >
                <div className="demo-role-header">
                  <span className="demo-role-name">{acc.name}</span>
                  <span
                    className="demo-role-badge"
                    style={{ backgroundColor: `${acc.badgeColor}22`, color: acc.badgeColor, borderColor: `${acc.badgeColor}55` }}
                  >
                    {acc.roleTitle}
                  </span>
                </div>
                <div className="demo-role-email">{acc.email}</div>
                <div className="demo-role-desc">{acc.desc}</div>
              </div>
            ))}
          </div>
        </div>

        <div className="login-security-footer">
          <ShieldCheck size={14} color="#10b981" />
          <span>Protected with Argon2/Bcrypt Cost-12 & 15m Signed JWT Access Tokens</span>
        </div>
      </div>
    </div>
  );
}
