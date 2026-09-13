import React from 'react';
import { FeatureFlagProvider, useFeatureFlag, useFlagUser } from './sdk/FeatureFlagContext';
import { Sparkles, Layers, ArrowRight, UserCheck, Radio, Zap } from 'lucide-react';

function DesignSystemShowcase() {
  const { userId, setUserId, lastUpdated } = useFlagUser();
  const buttonVariant = useFeatureFlag<string>('ds-button-v2', 'v1');

  return (
    <div style={styles.showcase}>
      {/* Header Info */}
      <header style={styles.header}>
        <div style={styles.badge}>
          <Sparkles size={16} color="#38bdf8" />
          <span>React Design System SDK Demo</span>
        </div>
        <h1 style={styles.title}>Design System Component Sandbox</h1>
        <p style={styles.subtitle}>
          Zero Cumulative Layout Shift (0 CLS) • Live SSE Sync • Evaluation in &lt;1ms
        </p>
      </header>

      {/* User Switcher Control */}
      <div style={styles.userCard}>
        <div style={styles.userCardHeader}>
          <UserCheck size={18} color="#4ade80" />
          <span style={styles.userCardTitle}>Active User Session Context</span>
        </div>
        <div style={styles.userSelector}>
          {['user_42', 'alice@example.com', 'user-beta-01', 'user-regular-01'].map((id) => (
            <button
              key={id}
              onClick={() => setUserId(id)}
              style={{
                ...styles.userBtn,
                ...(userId === id ? styles.activeUserBtn : {}),
              }}
            >
              {id}
            </button>
          ))}
        </div>
        <div style={styles.sseStatus}>
          <Radio size={14} color="#4ade80" className="pulse" />
          <span>Real-time SSE Stream Active (Last Synced: {new Date(lastUpdated).toLocaleTimeString()})</span>
        </div>
      </div>

      {/* Component Variant Showcase */}
      <div style={styles.componentCard}>
        <div style={styles.compHeader}>
          <Layers size={20} color="#f59e0b" />
          <div>
            <h2 style={styles.compTitle}>Component: &lt;Button /&gt;</h2>
            <span style={styles.flagMeta}>Flag Key: <code>ds-button-v2</code></span>
          </div>
          <div style={styles.badgeGroup}>
            <div style={styles.activeVariantBadge}>
              Active Variant: <strong>{buttonVariant}</strong>
            </div>
          </div>
        </div>

        <div style={styles.previewBox}>
          {buttonVariant === 'v2' && (
            <button style={styles.btnV2}>
              <Zap size={18} />
              <span>Redesigned Action Button (v2 Variant)</span>
              <ArrowRight size={16} />
            </button>
          )}

          {buttonVariant === 'compact' && (
            <button style={styles.btnCompact}>
              <span>Compact Button (Dense Variant)</span>
            </button>
          )}

          {buttonVariant === 'v1' && (
            <button style={styles.btnV1}>
              Legacy Action Button (v1 Default)
            </button>
          )}
        </div>

        <div style={styles.evalInfoBox}>
          <Sparkles size={14} color="#38bdf8" />
          <span>
            Evaluation Rule: <strong>{userId}</strong> evaluates to <code>"{buttonVariant}"</code>.
            {buttonVariant !== 'v1' ? ' (Assigned via Specific User Override)' : ' (Default Base Variant)'}
          </span>
        </div>

        <div style={styles.codeSnippet}>
          <code>
            {`// Zero-CLS React Component Usage\nconst variant = useFeatureFlag('ds-button-v2', 'v1');\n<Button variant="${buttonVariant}" />`}
          </code>
        </div>
      </div>
    </div>
  );
}

export default function App() {
  return (
    <FeatureFlagProvider>
      <DesignSystemShowcase />
    </FeatureFlagProvider>
  );
}

const styles: Record<string, React.CSSProperties> = {
  showcase: {
    maxWidth: '900px',
    margin: '0 auto',
    padding: '24px 16px',
    boxSizing: 'border-box',
    width: '100%',
  },
  header: {
    textAlign: 'center',
    marginBottom: '28px',
  },
  badge: {
    display: 'inline-flex',
    alignItems: 'center',
    gap: '8px',
    background: 'rgba(56, 189, 248, 0.1)',
    border: '1px solid rgba(56, 189, 248, 0.2)',
    padding: '6px 14px',
    borderRadius: '20px',
    color: '#38bdf8',
    fontSize: '13px',
    fontWeight: 600,
    marginBottom: '16px',
    maxWidth: '100%',
    boxSizing: 'border-box',
  },
  title: {
    margin: 0,
    fontSize: 'clamp(20px, 5vw, 28px)',
    fontWeight: 700,
    color: '#f8fafc',
    lineHeight: '1.2',
  },
  subtitle: {
    margin: '8px 0 0 0',
    fontSize: 'clamp(12px, 3.5vw, 14px)',
    color: '#94a3b8',
    lineHeight: '1.4',
  },
  userCard: {
    background: '#0f172a',
    border: '1px solid #1e293b',
    borderRadius: '16px',
    padding: '16px',
    marginBottom: '20px',
    boxSizing: 'border-box',
  },
  userCardHeader: {
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    marginBottom: '12px',
  },
  userCardTitle: {
    fontSize: '14px',
    fontWeight: 600,
    color: '#f8fafc',
  },
  userSelector: {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fit, minmax(130px, 1fr))',
    gap: '8px',
    marginBottom: '12px',
  },
  userBtn: {
    background: '#1e293b',
    color: '#94a3b8',
    border: '1px solid #334155',
    borderRadius: '8px',
    padding: '10px 12px',
    cursor: 'pointer',
    fontSize: '13px',
    fontWeight: 500,
    transition: 'all 0.2s ease',
    textOverflow: 'ellipsis',
    overflow: 'hidden',
    whiteSpace: 'nowrap',
    width: '100%',
    boxSizing: 'border-box',
  },
  activeUserBtn: {
    background: '#0284c7',
    color: '#ffffff',
    borderColor: '#38bdf8',
    fontWeight: 600,
  },
  sseStatus: {
    display: 'flex',
    alignItems: 'flex-start',
    gap: '8px',
    fontSize: '12px',
    color: '#94a3b8',
    lineHeight: '1.4',
    wordBreak: 'break-word',
  },
  componentCard: {
    background: '#0f172a',
    border: '1px solid #1e293b',
    borderRadius: '16px',
    padding: '20px',
    boxSizing: 'border-box',
  },
  compHeader: {
    display: 'flex',
    flexDirection: 'column',
    gap: '12px',
    marginBottom: '20px',
  },
  compTitle: {
    margin: 0,
    fontSize: 'clamp(16px, 4vw, 20px)',
    fontWeight: 700,
    color: '#f8fafc',
  },
  flagMeta: {
    fontSize: '12px',
    color: '#94a3b8',
  },
  activeVariantBadge: {
    alignSelf: 'flex-start',
    background: 'rgba(245, 158, 11, 0.15)',
    color: '#fbbf24',
    border: '1px solid rgba(245, 158, 11, 0.3)',
    padding: '6px 14px',
    borderRadius: '20px',
    fontSize: '13px',
  },
  previewBox: {
    background: '#020617',
    border: '1px dashed #334155',
    borderRadius: '12px',
    padding: '32px 16px',
    display: 'flex',
    justifyContent: 'center',
    alignItems: 'center',
    marginBottom: '20px',
    minHeight: '100px',
    boxSizing: 'border-box',
    width: '100%',
    overflowX: 'auto',
  },
  btnV1: {
    background: '#334155',
    color: '#f8fafc',
    border: 'none',
    borderRadius: '6px',
    padding: '12px 20px',
    fontSize: '14px',
    cursor: 'pointer',
    maxWidth: '100%',
    wordBreak: 'break-word',
  },
  btnV2: {
    background: 'linear-gradient(135deg, #0284c7 0%, #2563eb 100%)',
    color: '#ffffff',
    border: 'none',
    borderRadius: '12px',
    padding: '12px 20px',
    fontSize: '14px',
    fontWeight: 600,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    gap: '8px',
    boxShadow: '0 8px 20px -4px rgba(37, 99, 235, 0.5)',
    cursor: 'pointer',
    maxWidth: '100%',
    textAlign: 'center',
  },
  btnCompact: {
    background: '#10b981',
    color: '#ffffff',
    border: 'none',
    borderRadius: '4px',
    padding: '6px 12px',
    fontSize: '12px',
    fontWeight: 600,
    cursor: 'pointer',
  },
  codeSnippet: {
    background: '#020617',
    border: '1px solid #1e293b',
    borderRadius: '8px',
    padding: '14px',
    fontFamily: 'monospace',
    color: '#38bdf8',
    fontSize: '12px',
    whiteSpace: 'pre-wrap',
    wordBreak: 'break-word',
    overflowX: 'auto',
  },
  evalInfoBox: {
    background: 'rgba(15, 23, 42, 0.8)',
    border: '1px solid #334155',
    borderRadius: '8px',
    padding: '10px 14px',
    marginBottom: '16px',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    fontSize: '12px',
    color: '#cbd5e1',
  },
};
