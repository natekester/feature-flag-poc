import React, { useState, useEffect } from 'react';
import { Sliders, UserCheck, RefreshCw, Zap, Shield, CheckCircle, Server, Trash2, AlertTriangle, X } from 'lucide-react';

interface FeatureFlag {
  key: string;
  enabled: boolean;
  defaultValue: any;
  userOverrides?: Record<string, any>;
  rollout?: Array<{ variation: any; bucketPercentage: number }>;
  updatedAt: number;
}

const getApiUrl = (path: string): string => {
  const isCaddyProxy = window.location.port === '' || window.location.port === '80' || window.location.port === '443';
  const prefix = isCaddyProxy ? '/api' : 'http://localhost:8080/api';
  return `${prefix}${path.startsWith('/') ? path : '/' + path}`;
};

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

  // Modal confirmation state
  const [confirmModal, setConfirmModal] = useState<{
    isOpen: boolean;
    flagKey: string;
    userId: string;
  }>({ isOpen: false, flagKey: '', userId: '' });

  const fetchFlags = async () => {
    setLoading(true);
    setError(null);
    try {
      const res = await fetch(getApiUrl('/v1/admin/flags'));
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = await res.json();
      const loadedFlags: FeatureFlag[] = data || [];
      setFlags(loadedFlags);
      if (loadedFlags.length > 0 && (!selectedFlag || !loadedFlags.some((f) => f.key === selectedFlag))) {
        setSelectedFlag(loadedFlags[0].key);
      }
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
    if (!targetUserId.trim() || !selectedFlag) {
      setError('Please select a valid feature flag and enter a target user ID.');
      return;
    }

    setSaving(true);
    setSuccessMessage(null);
    setError(null);
    try {
      const endpoint = getApiUrl(`/v1/admin/flags/${encodeURIComponent(selectedFlag)}/overrides/users/${encodeURIComponent(targetUserId.trim())}`);
      const res = await fetch(endpoint, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ variation: overrideVariation }),
      });

      if (!res.ok) {
        const errData = await res.json().catch(() => ({}));
        throw new Error(errData.error || `HTTP ${res.status}`);
      }

      setSuccessMessage(`Override saved! User '${targetUserId.trim()}' is now assigned to '${overrideVariation}'.`);
      fetchFlags(); // refresh list
    } catch (err: any) {
      setError(`Error saving override: ${err.message}`);
    } finally {
      setSaving(false);
    }
  };

  const openDeleteModal = (flagKey: string, userId: string) => {
    setConfirmModal({ isOpen: true, flagKey, userId });
  };

  const closeDeleteModal = () => {
    setConfirmModal({ isOpen: false, flagKey: '', userId: '' });
  };

  const confirmRemoveOverride = async () => {
    const { flagKey, userId } = confirmModal;
    if (!flagKey || !userId) return;

    closeDeleteModal();
    setSuccessMessage(null);
    setError(null);
    try {
      const endpoint = getApiUrl(`/v1/admin/flags/${encodeURIComponent(flagKey)}/overrides/users/${encodeURIComponent(userId)}`);
      const res = await fetch(endpoint, { method: 'DELETE' });

      if (!res.ok) {
        const errData = await res.json().catch(() => ({}));
        throw new Error(errData.error || `HTTP ${res.status}`);
      }

      setSuccessMessage(`Override removed for user '${userId}'.`);
      fetchFlags(); // refresh list
    } catch (err: any) {
      setError(`Error removing override: ${err.message}`);
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
                              <button
                                onClick={() => openDeleteModal(flag.key, uid)}
                                title={`Remove override for ${uid}`}
                                style={styles.removeBtn}
                              >
                                <Trash2 size={12} />
                                <span>Delete</span>
                              </button>
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

      {/* Confirmation Modal */}
      {confirmModal.isOpen && (
        <div style={styles.modalOverlay} onClick={closeDeleteModal}>
          <div style={styles.modalContent} onClick={(e) => e.stopPropagation()}>
            <div style={styles.modalHeader}>
              <div style={styles.modalTitleGroup}>
                <AlertTriangle size={20} color="#f87171" />
                <h3 style={styles.modalTitle}>Remove User Override?</h3>
              </div>
              <button onClick={closeDeleteModal} style={styles.modalCloseBtn}>
                <X size={18} />
              </button>
            </div>

            <p style={styles.modalText}>
              Are you sure you want to remove the override for user <code style={styles.code}>{confirmModal.userId}</code> on flag <code style={styles.code}>{confirmModal.flagKey}</code>? This user will revert to default percentage rollout bucketing.
            </p>

            <div style={styles.modalActions}>
              <button onClick={closeDeleteModal} style={styles.cancelBtn}>
                Cancel
              </button>
              <button onClick={confirmRemoveOverride} style={styles.confirmDeleteBtn}>
                <Trash2 size={16} />
                <span>Remove Override</span>
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

const styles: Record<string, React.CSSProperties> = {
  container: {
    maxWidth: '1200px',
    margin: '0 auto',
    padding: '20px 16px',
    boxSizing: 'border-box',
    width: '100%',
  },
  header: {
    display: 'flex',
    flexDirection: 'row',
    justifyContent: 'space-between',
    alignItems: 'center',
    flexWrap: 'wrap',
    gap: '16px',
    marginBottom: '24px',
    borderBottom: '1px solid #1e293b',
    paddingBottom: '20px',
  },
  logoGroup: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
    flex: '1 1 280px',
  },
  iconBadge: {
    width: '44px',
    height: '44px',
    borderRadius: '12px',
    background: 'rgba(56, 189, 248, 0.1)',
    border: '1px solid rgba(56, 189, 248, 0.2)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    flexShrink: 0,
  },
  title: {
    margin: 0,
    fontSize: 'clamp(18px, 4.5vw, 24px)',
    fontWeight: 700,
    color: '#f8fafc',
    lineHeight: '1.2',
  },
  subtitle: {
    margin: '4px 0 0 0',
    fontSize: 'clamp(12px, 3.5vw, 14px)',
    color: '#94a3b8',
    lineHeight: '1.4',
  },
  refreshBtn: {
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: '8px',
    background: '#1e293b',
    color: '#f8fafc',
    border: '1px solid #334155',
    borderRadius: '8px',
    padding: '10px 16px',
    cursor: 'pointer',
    fontWeight: 500,
    fontSize: '14px',
  },
  grid: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(min(100%, 540px), 1fr))',
    gap: '20px',
  },
  card: {
    background: '#1e293b',
    borderRadius: '16px',
    border: '1px solid #334155',
    padding: '20px',
    boxSizing: 'border-box',
    width: '100%',
    overflowX: 'auto',
  },
  cardHeader: {
    display: 'flex',
    alignItems: 'center',
    gap: '12px',
    marginBottom: '8px',
  },
  cardTitle: {
    margin: 0,
    fontSize: 'clamp(16px, 4vw, 18px)',
    fontWeight: 600,
    color: '#f8fafc',
  },
  cardSubtitle: {
    margin: '0 0 20px 0',
    fontSize: '13px',
    color: '#94a3b8',
    lineHeight: '1.5',
  },
  table: {
    width: '100%',
    borderCollapse: 'collapse',
    marginTop: '16px',
    minWidth: '280px',
  },
  thRow: {
    borderBottom: '1px solid #334155',
  },
  th: {
    textAlign: 'left',
    padding: '10px 12px',
    color: '#94a3b8',
    fontSize: '12px',
    fontWeight: 600,
    textTransform: 'uppercase',
    whiteSpace: 'nowrap',
  },
  tr: {
    borderBottom: '1px solid rgba(51, 65, 85, 0.5)',
  },
  tdKey: {
    padding: '12px',
    fontWeight: 600,
    color: '#f8fafc',
    fontFamily: 'monospace',
    fontSize: '13px',
    wordBreak: 'break-all',
  },
  td: {
    padding: '12px',
    color: '#cbd5e1',
    fontSize: '13px',
  },
  enabledBadge: {
    background: 'rgba(74, 222, 128, 0.15)',
    color: '#4ade80',
    padding: '4px 10px',
    borderRadius: '20px',
    fontSize: '12px',
    fontWeight: 600,
    display: 'inline-block',
  },
  disabledBadge: {
    background: 'rgba(248, 113, 113, 0.15)',
    color: '#f87171',
    padding: '4px 10px',
    borderRadius: '20px',
    fontSize: '12px',
    fontWeight: 600,
    display: 'inline-block',
  },
  code: {
    background: '#0f172a',
    padding: '3px 8px',
    borderRadius: '6px',
    color: '#38bdf8',
    fontFamily: 'monospace',
    fontSize: '12px',
    wordBreak: 'break-all',
    display: 'inline-block',
  },
  overridesList: {
    display: 'flex',
    flexDirection: 'column',
    gap: '6px',
    maxWidth: '100%',
  },
  overrideTag: {
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'space-between',
    gap: '6px',
    background: '#0f172a',
    border: '1px solid #334155',
    padding: '6px 10px',
    borderRadius: '6px',
    fontSize: '12px',
    color: '#38bdf8',
    wordBreak: 'break-word',
    maxWidth: '100%',
    boxSizing: 'border-box',
    flexWrap: 'wrap',
  },
  removeBtn: {
    background: 'rgba(239, 68, 68, 0.15)',
    border: '1px solid rgba(239, 68, 68, 0.3)',
    color: '#f87171',
    cursor: 'pointer',
    padding: '4px 8px',
    display: 'inline-flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: '4px',
    borderRadius: '4px',
    fontSize: '11px',
    fontWeight: 600,
    flexShrink: 0,
    marginLeft: 'auto',
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
    width: '100%',
    boxSizing: 'border-box',
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
    width: '100%',
    boxSizing: 'border-box',
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
    lineHeight: '1.4',
  },
  infoBox: {
    marginTop: '20px',
    padding: '12px',
    background: '#0f172a',
    borderRadius: '8px',
    display: 'flex',
    alignItems: 'flex-start',
    gap: '10px',
    color: '#94a3b8',
    fontSize: '12px',
    lineHeight: '1.4',
  },
  placeholder: {
    padding: '32px 16px',
    textAlign: 'center',
    color: '#64748b',
    fontSize: '14px',
  },
  errorBox: {
    background: 'rgba(239, 68, 68, 0.1)',
    border: '1px solid rgba(239, 68, 68, 0.3)',
    color: '#f87171',
    padding: '12px',
    borderRadius: '8px',
    marginBottom: '16px',
    fontSize: '13px',
  },
  rolloutList: {
    display: 'flex',
    flexDirection: 'row',
    flexWrap: 'wrap',
    gap: '6px',
  },
  rolloutBadge: {
    background: '#0f172a',
    border: '1px solid #334155',
    padding: '4px 8px',
    borderRadius: '6px',
    fontSize: '12px',
    color: '#f59e0b',
  },
  modalOverlay: {
    position: 'fixed',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    background: 'rgba(2, 6, 23, 0.8)',
    backdropFilter: 'blur(4px)',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    zIndex: 1000,
    padding: '16px',
  },
  modalContent: {
    background: '#1e293b',
    border: '1px solid #334155',
    borderRadius: '16px',
    padding: '24px',
    maxWidth: '440px',
    width: '100%',
    boxSizing: 'border-box',
    boxShadow: '0 20px 25px -5px rgba(0, 0, 0, 0.5), 0 8px 10px -6px rgba(0, 0, 0, 0.5)',
  },
  modalHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '16px',
  },
  modalTitleGroup: {
    display: 'flex',
    alignItems: 'center',
    gap: '10px',
  },
  modalTitle: {
    margin: 0,
    fontSize: '18px',
    fontWeight: 600,
    color: '#f8fafc',
  },
  modalCloseBtn: {
    background: 'none',
    border: 'none',
    color: '#94a3b8',
    cursor: 'pointer',
    padding: '4px',
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    borderRadius: '6px',
  },
  modalText: {
    margin: '0 0 24px 0',
    fontSize: '14px',
    color: '#cbd5e1',
    lineHeight: '1.5',
  },
  modalActions: {
    display: 'flex',
    justifyContent: 'flex-end',
    gap: '12px',
  },
  cancelBtn: {
    background: '#0f172a',
    color: '#94a3b8',
    border: '1px solid #334155',
    borderRadius: '8px',
    padding: '10px 16px',
    cursor: 'pointer',
    fontSize: '14px',
    fontWeight: 500,
  },
  confirmDeleteBtn: {
    background: '#dc2626',
    color: '#ffffff',
    border: 'none',
    borderRadius: '8px',
    padding: '10px 16px',
    cursor: 'pointer',
    fontSize: '14px',
    fontWeight: 600,
    display: 'inline-flex',
    alignItems: 'center',
    gap: '8px',
  },
};

if (import.meta.vitest) {
  const { describe, it, expect, vi, beforeEach } = import.meta.vitest;
  const { render, screen, fireEvent, waitFor } = await import('@testing-library/react');

  describe('Admin Portal Console', () => {
    beforeEach(() => {
      vi.restoreAllMocks();
    });

    it('renders admin console title and flag details', async () => {
      const mockFlags = [
        {
          key: 'ds-button-v2',
          enabled: true,
          defaultValue: 'v1',
          userOverrides: { user_42: 'v2' },
          updatedAt: Date.now(),
        },
      ];

      global.fetch = vi.fn().mockResolvedValue({
        ok: true,
        json: async () => mockFlags,
      });

      render(<App />);

      expect(screen.getByText('Feature Flag Control Console')).toBeDefined();

      await waitFor(() => {
        expect(screen.getByText('ds-button-v2')).toBeDefined();
        expect(screen.getByText(/user_42/)).toBeDefined();
      });
    });

    it('calls DELETE endpoint when user confirms removal in modal dialog', async () => {
      const mockFlags = [
        {
          key: 'ds-button-v2',
          enabled: true,
          defaultValue: 'v1',
          userOverrides: { user_42: 'v2' },
          updatedAt: Date.now(),
        },
      ];

      global.fetch = vi.fn().mockImplementation((url, options) => {
        if (options?.method === 'DELETE') {
          return Promise.resolve({
            ok: true,
            json: async () => ({ status: 'success', flagKey: 'ds-button-v2', userId: 'user_42' }),
          });
        }
        return Promise.resolve({
          ok: true,
          json: async () => mockFlags,
        });
      });

      render(<App />);

      await waitFor(() => {
        expect(screen.getByText(/user_42/)).toBeDefined();
      });

      const removeBtn = screen.getByTitle('Remove override for user_42');
      fireEvent.click(removeBtn);

      await waitFor(() => {
        expect(screen.getByText('Remove User Override?')).toBeDefined();
      });

      const confirmBtn = screen.getByText('Remove Override');
      fireEvent.click(confirmBtn);

      await waitFor(() => {
        expect(global.fetch).toHaveBeenCalledWith(
          expect.stringContaining('/v1/admin/flags/ds-button-v2/overrides/users/user_42'),
          expect.objectContaining({ method: 'DELETE' })
        );
      });
    });
  });
}
