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
    let bg = '#89b4fa'; // default blue
    let fg = '#11111b';

    if (m.includes('claude')) {
      bg = '#cba6f7'; // mauve / purple
    } else if (m.includes('gpt-4o-mini') || m.includes('mini')) {
      bg = '#fab387'; // amber
    } else if (m.includes('gpt-4o') || m.includes('openai')) {
      bg = '#a6e3a1'; // green
    } else if (m.includes('gemini')) {
      bg = '#89b4fa'; // blue
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
    if (pct >= 85) return '#f38ba8'; // red alert
    if (pct >= 70) return '#f9e2af'; // yellow warning
    return '#a6e3a1'; // green nominal
  };

  return (
    <div style={styles.container}>
      <div style={styles.topRow}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
          <span style={{ fontSize: '0.85rem', color: '#a6adc8', fontWeight: 600 }}>Active Model:</span>
          {metrics?.selected_model ? (
            <span style={getModelBadgeStyle(metrics.selected_model)}>
              ● {metrics.selected_model}
            </span>
          ) : (
            <span style={{ ...getModelBadgeStyle(''), opacity: 0.6 }}>
              ○ Auto-Negotiating...
            </span>
          )}
        </div>

        {metrics?.estimated_cost_delta && (
          <span style={styles.costBadge}>
            Est. Cost Savings: {metrics.estimated_cost_delta}
          </span>
        )}
      </div>

      {/* Volumetric Daily Usage Gauge */}
      <div style={styles.gaugeSection}>
        <div style={styles.gaugeHeader}>
          <span style={styles.gaugeLabel}>Daily Token Consumption Bar:</span>
          <span style={styles.gaugeValue}>
            {currentTotal.toLocaleString()} / {dailyCap.toLocaleString()} tokens ({usagePercentage}%)
          </span>
        </div>
        <div style={styles.trackBackground}>
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
        <div style={styles.rationaleBox}>
          <span style={styles.rationaleTitle}>Optimization Engine Insight:</span>
          <p style={styles.rationaleText}>{metrics.routing_rationale}</p>
          {metrics.complexity_score !== undefined && (
            <div style={styles.scoreRow}>
              <span>Analyzed Complexity Score: <strong>{(metrics.complexity_score * 100).toFixed(0)}%</strong></span>
              {metrics.budget_throttled && (
                <span style={styles.throttleAlert}>⚠️ Volumetric Throttled (≥85% Budget Cap)</span>
              )}
            </div>
          )}
        </div>
      )}

      {status === 'streaming' && (
        <div style={styles.streamingIndicator}>
          <span style={styles.dot}></span> SSE Word-by-Word Stream Active...
        </div>
      )}
    </div>
  );
};

const styles: Record<string, React.CSSProperties> = {
  container: {
    backgroundColor: '#181825',
    border: '1px solid #313244',
    borderRadius: '10px',
    padding: '16px',
    marginTop: '12px',
    fontFamily: 'system-ui, -apple-system, sans-serif',
    color: '#cdd6f4',
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
    backgroundColor: '#313244',
    color: '#a6e3a1',
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
    marginBottom: '4px',
    color: '#a6adc8',
  },
  gaugeLabel: {
    fontWeight: 600,
  },
  gaugeValue: {
    fontWeight: 600,
  },
  trackBackground: {
    backgroundColor: '#313244',
    borderRadius: '6px',
    height: '8px',
    width: '100%',
    overflow: 'hidden',
  },
  trackFill: {
    height: '100%',
    transition: 'width 0.4s ease-in-out, background-color 0.4s ease',
  },
  rationaleBox: {
    backgroundColor: '#1e1e2e',
    borderLeft: '4px solid #89b4fa',
    padding: '10px 14px',
    borderRadius: '0 6px 6px 0',
    fontSize: '0.85rem',
  },
  rationaleTitle: {
    fontWeight: 'bold',
    color: '#89b4fa',
    display: 'block',
    marginBottom: '2px',
  },
  rationaleText: {
    margin: 0,
    color: '#bac2de',
    lineHeight: 1.4,
  },
  scoreRow: {
    display: 'flex',
    justifyContent: 'space-between',
    marginTop: '6px',
    fontSize: '0.75rem',
    color: '#a6adc8',
  },
  throttleAlert: {
    color: '#f38ba8',
    fontWeight: 'bold',
  },
  streamingIndicator: {
    marginTop: '10px',
    fontSize: '0.8rem',
    color: '#89b4fa',
    display: 'flex',
    alignItems: 'center',
    gap: '6px',
  },
  dot: {
    width: '8px',
    height: '8px',
    borderRadius: '50%',
    backgroundColor: '#89b4fa',
    display: 'inline-block',
  },
};
