import React from 'react';
import type { MetricsData, FinalUsageData } from '../hooks/useChatSSE';
import type { UserConfig } from '../services/api';

interface OptimizationMetricsPanelProps {
  metrics: MetricsData | null;
  finalUsage: FinalUsageData | null;
  userConfig: UserConfig | null;
  status: string;
}

export const OptimizationMetricsPanel: React.FC<OptimizationMetricsPanelProps> = ({
  metrics,
  finalUsage,
  userConfig,
  status,
}) => {
  const dailyCap = userConfig?.daily_token_cap || 50000;
  const currentTotal = finalUsage?.updated_daily_total || 0;
  const usagePercentage = Math.min(100, Math.round((currentTotal / dailyCap) * 100));

  const getModelBadgeStyle = (modelName?: string): React.CSSProperties => {
    const m = (modelName || '').toLowerCase();
    let bg = 'var(--primary)';
    let fg = '#0f172a';

    if (m.includes('claude')) {
      bg = 'var(--purple)';
    } else if (m.includes('gpt-4o-mini') || m.includes('mini')) {
      bg = 'var(--amber)';
    } else if (m.includes('gpt-4o') || m.includes('openai')) {
      bg = 'var(--success)';
    } else if (m.includes('gemini')) {
      bg = 'var(--primary)';
    }

    return {
      backgroundColor: bg,
      color: fg,
      padding: '4px 12px',
      borderRadius: '12px',
      fontWeight: 'bold',
      fontSize: '0.85rem',
      display: 'inline-flex',
      alignItems: 'center',
      gap: '6px',
      textTransform: 'uppercase',
      letterSpacing: '0.5px',
    };
  };

  const getGaugeColor = (pct: number) => {
    if (pct >= 85) return 'var(--danger)';
    if (pct >= 70) return 'var(--warning)';
    return 'var(--success)';
  };

  return (
    <section
      role="region"
      aria-label="Optimization & Metrics Area"
      style={styles.container}
    >
      <div style={styles.topRow}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <span style={{ fontSize: '0.85rem', color: 'var(--text-muted)', fontWeight: 600 }}>Active Model:</span>
          {metrics?.selected_model ? (
            <span style={getModelBadgeStyle(metrics.selected_model)} aria-label={`Negotiated model: ${metrics.selected_model}`}>
              ● {metrics.selected_model}
            </span>
          ) : (
            <span style={{ ...getModelBadgeStyle(''), opacity: 0.6 }} aria-label="Negotiating model">
              ○ Auto-Negotiating...
            </span>
          )}
        </div>

        {metrics?.estimated_cost_delta && (
          <span style={styles.costBadge} aria-label={`Estimated cost savings: ${metrics.estimated_cost_delta}`}>
            Est. Cost Savings: {metrics.estimated_cost_delta}
          </span>
        )}
      </div>

      {/* Volumetric Daily Usage Gauge */}
      <div style={styles.gaugeSection}>
        <div style={styles.gaugeHeader}>
          <span style={styles.gaugeLabel}>Daily Token Consumption Bar:</span>
          <span style={styles.gaugeValue} aria-label={`Token consumption: ${currentTotal} of ${dailyCap} tokens, ${usagePercentage} percent`}>
            {currentTotal.toLocaleString()} / {dailyCap.toLocaleString()} tokens ({usagePercentage}%)
          </span>
        </div>
        <div
          role="progressbar"
          aria-valuenow={usagePercentage}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-label="Daily token budget usage"
          style={styles.trackBackground}
        >
          <div
            style={{
              ...styles.trackFill,
              width: `${usagePercentage}%`,
              backgroundColor: getGaugeColor(usagePercentage),
            }}
          />
        </div>
      </div>

      {/* Rationale & Optimization Readout */}
      {metrics?.routing_rationale && (
        <div style={styles.rationaleBox} aria-live="polite">
          <span style={styles.rationaleTitle}>Optimization Engine Insight:</span>
          <p style={styles.rationaleText}>{metrics.routing_rationale}</p>
          {metrics.complexity_score !== undefined && (
            <div style={styles.scoreRow}>
              <span>Analyzed Complexity Score: <strong>{(metrics.complexity_score * 100).toFixed(0)}%</strong></span>
              {metrics.budget_throttled && (
                <span style={styles.throttleAlert} role="alert">
                  ⚠️ Volumetric Throttled (≥85% Budget Cap)
                </span>
              )}
            </div>
          )}
        </div>
      )}

      {status === 'streaming' && (
        <div style={styles.streamingIndicator} aria-live="polite" aria-busy="true">
          <span style={styles.dot} className="pulse-dot"></span> SSE Word-by-Word Stream Active...
        </div>
      )}
    </section>
  );
};

const styles: Record<string, React.CSSProperties> = {
  container: {
    backgroundColor: 'var(--bg-card)',
    border: '1px solid var(--border-color)',
    borderRadius: '12px',
    padding: '18px',
    marginTop: '14px',
    color: 'var(--text-main)',
  },
  topRow: {
    display: 'flex',
    justifyContent: 'space-between',
    alignItems: 'center',
    marginBottom: '14px',
    flexWrap: 'wrap',
    gap: '8px',
  },
  costBadge: {
    backgroundColor: 'rgba(74, 222, 128, 0.15)',
    color: 'var(--success)',
    border: '1px solid rgba(74, 222, 128, 0.3)',
    padding: '4px 10px',
    borderRadius: '6px',
    fontSize: '0.8rem',
    fontWeight: 600,
  },
  gaugeSection: {
    marginBottom: '12px',
  },
  gaugeHeader: {
    display: 'flex',
    justifyContent: 'space-between',
    fontSize: '0.8rem',
    marginBottom: '6px',
    color: 'var(--text-muted)',
  },
  gaugeLabel: {
    fontWeight: 600,
  },
  gaugeValue: {
    fontWeight: 600,
  },
  trackBackground: {
    backgroundColor: 'var(--bg-app)',
    borderRadius: '6px',
    height: '10px',
    width: '100%',
    overflow: 'hidden',
  },
  trackFill: {
    height: '100%',
    transition: 'width 0.4s ease-in-out, background-color 0.4s ease',
  },
  rationaleBox: {
    backgroundColor: 'var(--bg-app)',
    borderLeft: '4px solid var(--primary)',
    padding: '12px 14px',
    borderRadius: '0 8px 8px 0',
    fontSize: '0.85rem',
  },
  rationaleTitle: {
    fontWeight: 'bold',
    color: 'var(--primary)',
    display: 'block',
    marginBottom: '2px',
  },
  rationaleText: {
    margin: 0,
    color: 'var(--text-main)',
    lineHeight: 1.4,
  },
  scoreRow: {
    display: 'flex',
    justifyContent: 'space-between',
    marginTop: '8px',
    fontSize: '0.75rem',
    color: 'var(--text-muted)',
  },
  throttleAlert: {
    color: 'var(--danger)',
    fontWeight: 'bold',
  },
  streamingIndicator: {
    marginTop: '12px',
    fontSize: '0.8rem',
    color: 'var(--primary)',
    display: 'flex',
    alignItems: 'center',
    gap: '8px',
    fontWeight: 600,
  },
  dot: {
    width: '8px',
    height: '8px',
  },
};
