import React, { useState, useEffect } from 'react';
import { Sliders, UserCheck, RefreshCw, Zap, Shield, CheckCircle, Server } from 'lucide-react';

interface FeatureFlag {
  key: string;
  enabled: boolean;
  defaultValue: any;
  userOverrides?: Record<string, any>;
  rollout?: Array<{ variation: any; bucketPercentage: number }>;
  updatedAt: number;
}

export default function App() {
  const [flags, setFlags] = useState<FeatureFlag[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // User override form state
  const [selectedFlag, setSelectedFlag] = useState<string>('ds-button-v2');
  const [targetUserId, setTargetUserId] = useState<string>('user_42');
  const [overrideVariation, setOverrideVariation] = useState<string>('v2');
  const [saving, setSaving] = useState(false);
  const [successMessage, setSuccessMessage] = useState<string | null>(null);

  const fetchFlags = async () => {
    setLoading(true);
    setError(null);
    try {
      // Calls /api/v1/admin/flags via Caddy gateway or fallback
      const apiHost = window.location.port === '3000' ? 'http://localhost:8080/api' : '/api';
      const res = await fetch(`${apiHost}/v1/admin/flags`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      setFlags(data || []);
    } catch (err: any) {
      setError(`Failed to fetch flags: ${err.message}`);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchFlags();
  }, []);

  const handleSaveOverride = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!targetUserId.trim() || !selectedFlag) return;

    setSaving(true);
    setSuccessMessage(null);
    try {
      const apiHost = window.location.port === '3000' ? 'http://localhost:8080/api' : '/api';
      const res = await fetch(`${apiHost}/v1/admin/flags/${selectedFlag}/overrides/users/${encodeURIComponent(targetUserId.trim())}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ variation: overrideVariation }),
      });

      if (!res.ok) throw new Error(`HTTP ${res.status}`);

      setSuccessMessage(`Override saved! User '${targetUserId.trim()}' is now assigned to '${overrideVariation}'.`);
      fetchFlags(); // refresh list
    } catch (err: any) {
      alert(`Error saving override: ${err.message}`);
    } finally {
      setSaving(false);
    }
  };

  return (
    <div style={styles.container}>
      <header style={styles.header}>
        <div style={styles.logoGroup}>
          <div style={styles.iconBadge}>
            <Sliders size={24} color="#38bdf8" />
          </div>
          <div>
            <h1 style={styles.title}>Feature Flag Control Console</h1>
            <p style={styles.subtitle}>Design System Management • Local Dev Skaffold POC</p>
          </div>
        </div>
        <button onClick={fetchFlags} style={styles.refreshBtn} disabled={loading}>
          <RefreshCw size={16} className={loading ? 'spin' : ''} />
          {loading ? 'Refreshing...' : 'Refresh'}
        </button>
      </header>

      <div style={styles.grid}>
        {/* Main Flag Management Table */}
        <section style={styles.card}>
          <div style={styles.cardHeader}>
            <Zap size={20} color="#f59e0b" />
            <h2 style={styles.cardTitle}>Active Design System Flags</h2>
          </div>

          {error && <div style={styles.errorBox}>{error}</div>}

          {loading ? (
            <div style={styles.placeholder}>Loading flags from Go backend...</div>
          ) : flags.length === 0 ? (
            <div style={styles.placeholder}>No flags found. Check DynamoDB Local initialization.</div>
          ) : (
            <table style={styles.table}>
              <thead>
                <tr style={styles.thRow}>
                  <th style={styles.th}>Flag Key</th>
                  <th style={styles.th}>Status</th>
                  <th style={styles.th}>Default Variant</th>
                  <th style={styles.th}>Active User Overrides</th>
                </tr>
              </thead>
              <tbody>
                {flags.map((flag) => (
                  <tr key={flag.key} style={styles.tr}>
                    <td style={styles.tdKey}>{flag.key}</td>
                    <td style={styles.td}>
                      <span style={flag.enabled ? styles.enabledBadge : styles.disabledBadge}>
                        {flag.enabled ? 'Enabled' : 'Disabled'}
                      </span>
                    </td>
                    <td style={styles.td}>
                      <code style={styles.code}>{JSON.stringify(flag.defaultValue)}</code>
                    </td>
                    <td style={styles.td}>
                      {flag.userOverrides && Object.keys(flag.userOverrides).length > 0 ? (
                        <div style={styles.overridesList}>
                          {Object.entries(flag.userOverrides).map(([uid, val]) => (
                            <span key={uid} style={styles.overrideTag}>
                              <UserCheck size={12} /> {uid}: <strong>{String(val)}</strong>
                            </span>
                          ))}
                        </div>
                      ) : (
                        <span style={{ color: '#64748b' }}>None</span>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          )}
        </section>

        {/* User Override Control Panel */}
        <section style={styles.card}>
          <div style={styles.cardHeader}>
            <UserCheck size={20} color="#38bdf8" />
            <h2 style={styles.cardTitle}>User-Specific Override POC</h2>
          </div>
          <p style={styles.cardSubtitle}>
            Target a specific test user (e.g. <code style={styles.code}>user_42</code>) to instantly force a component variant with <strong>0 CLS</strong>.
          </p>

          <form onSubmit={handleSaveOverride} style={styles.form}>
            <div style={styles.formGroup}>
              <label style={styles.label}>Select Feature Flag</label>
              <select
                style={styles.input}
                value={selectedFlag}
                onChange={(e) => setSelectedFlag(e.target.value)}
              >
                {flags.map((f) => (
                  <option key={f.key} value={f.key}>
                    {f.key}
                  </option>
                ))}
                {flags.length === 0 && <option value="ds-button-v2">ds-button-v2</option>}
              </select>
            </div>

            <div style={styles.formGroup}>
              <label style={styles.label}>Target User ID</label>
              <input
                type="text"
                style={styles.input}
                value={targetUserId}
                onChange={(e) => setTargetUserId(e.target.value)}
                placeholder="e.g. user_42, alice@example.com"
                required
              />
            </div>

            <div style={styles.formGroup}>
              <label style={styles.label}>Forced Variation</label>
              <select
                style={styles.input}
                value={overrideVariation}
                onChange={(e) => setOverrideVariation(e.target.value)}
              >
                <option value="v1">v1 (Legacy Design)</option>
                <option value="v2">v2 (New Redesign)</option>
                <option value="compact">compact (Dense Variant)</option>
              </select>
            </div>

            <button type="submit" style={styles.submitBtn} disabled={saving}>
              {saving ? 'Saving to DynamoDB & Redis...' : 'Save User Override'}
            </button>
          </form>

          {successMessage && (
            <div style={styles.successBox}>
              <CheckCircle size={18} color="#4ade80" />
              <span>{successMessage}</span>
            </div>
          )}

          <div style={styles.infoBox}>
            <Server size={16} color="#94a3b8" />
            <span>Updates trigger atomic DynamoDB write, Redis cache purge, and real-time SSE stream event.</span>
          </div>
        </section>
      </div>
    </div>
  );
}

const styles: Record<string, React.CSSProperties> = {
  container: {
    maxWidth: '1200px',
    margin: '0 auto',
    padding: '32px 24px',
  },
  header: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '32px',
    borderBottom: '1px solid #1e293b',
    paddingBottom: '24px',
  },
  logoGroup: {
    display: 'flex',
    alignItems: 'center',
    gap: '16px',
  },
  iconBadge: {
    width: '48px',
    height: '48px',
    borderRadius: '12px',
    background: 'rgba(56, 189, 248, 0.1)',
    border: '1px solid rgba(56, 189, 248, 0.2)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
  },
  title: {
    margin: 0,
    fontSize: '24px',
    fontWeight: 700,
    color: '#f8fafc',
  },
  subtitle: {
    margin: '4px 0 0 0',
    fontSize: '14px',
    color: '#94a3b8',
  },
  refreshBtn: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    background: '#1e293b',
    color: '#f8fafc',
    border: '1px solid #334155',
    borderRadius: '8px',
    padding: '10px 18px',
    cursor: 'pointer',
    fontWeight: 500,
  },
  grid: {
    display: 'grid',
    gridTemplateColumns: '1fr 400px',
    gap: '24px',
  },
  card: {
    background: '#1e293b',
    borderRadius: '16px',
    border: '1px solid #334155',
    padding: '24px',
  },
  cardHeader: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
    marginBottom: '8px',
  },
  cardTitle: {
    margin: 0,
    fontSize: '18px',
    fontWeight: 600,
    color: '#f8fafc',
  },
  cardSubtitle: {
    margin: '0 0 20px 0',
    fontSize: '14px',
    color: '#94a3b8',
    lineHeight: '1.5',
  },
  table: {
    width: '100%',
    borderCollapse: 'collapse',
    marginTop: '16px',
  },
  thRow: {
    borderBottom: '1px solid #334155',
  },
  th: {
    textAlign: 'left',
    padding: '12px 16px',
    color: '#94a3b8',
    fontSize: '13px',
    fontWeight: 600,
    textTransform: 'uppercase',
  },
  tr: {
    borderBottom: '1px solid rgba(51, 65, 85, 0.5)',
  },
  tdKey: {
    padding: '16px',
    fontWeight: 600,
    color: '#f8fafc',
    fontFamily: 'monospace',
  },
  td: {
    padding: '16px',
    color: '#cbd5e1',
    fontSize: '14px',
  },
  enabledBadge: {
    background: 'rgba(74, 222, 128, 0.15)',
    color: '#4ade80',
    padding: '4px 10px',
    borderRadius: '20px',
    fontSize: '12px',
    fontWeight: 600,
  },
  disabledBadge: {
    background: 'rgba(248, 113, 113, 0.15)',
    color: '#f87171',
    padding: '4px 10px',
    borderRadius: '20px',
    fontSize: '12px',
    fontWeight: 600,
  },
  code: {
    background: '#0f172a',
    padding: '3px 8px',
    borderRadius: '6px',
    color: '#38bdf8',
    fontFamily: 'monospace',
    fontSize: '13px',
  },
  overridesList: {
    display: 'flex',
    flexDirection: 'column',
    gap: '6px',
  },
  overrideTag: {
    display: 'inline-flex',
    alignItems: 'center',
    gap: '6px',
    background: '#0f172a',
    border: '1px solid #334155',
    padding: '4px 8px',
    borderRadius: '6px',
    fontSize: '12px',
    color: '#38bdf8',
  },
  form: {
    display: 'flex',
    flexDirection: 'column',
    gap: '16px',
  },
  formGroup: {
    display: 'flex',
    flexDirection: 'column',
    gap: '6px',
  },
  label: {
    fontSize: '13px',
    fontWeight: 600,
    color: '#cbd5e1',
  },
  input: {
    background: '#0f172a',
    border: '1px solid #334155',
    borderRadius: '8px',
    padding: '10px 14px',
    color: '#f8fafc',
    fontSize: '14px',
    outline: 'none',
  },
  submitBtn: {
    background: '#0284c7',
    color: '#ffffff',
    border: 'none',
    borderRadius: '8px',
    padding: '12px',
    fontSize: '14px',
    fontWeight: 600,
    cursor: 'pointer',
    marginTop: '8px',
  },
  successBox: {
    marginTop: '16px',
    padding: '12px 16px',
    background: 'rgba(74, 222, 128, 0.1)',
    border: '1px solid rgba(74, 222, 128, 0.3)',
    borderRadius: '8px',
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
    color: '#4ade80',
    fontSize: '13px',
  },
  infoBox: {
    marginTop: '20px',
    padding: '12px',
    background: '#0f172a',
    borderRadius: '8px',
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
    color: '#94a3b8',
    fontSize: '12px',
    lineHeight: '1.4',
  },
  placeholder: {
    padding: '32px',
    textAlign: 'center',
    color: '#64748b',
  },
  errorBox: {
    background: 'rgba(239, 68, 68, 0.1)',
    border: '1px solid rgba(239, 68, 68, 0.3)',
    color: '#f87171',
    padding: '12px',
    borderRadius: '8px',
    marginBottom: '16px',
  },
};
