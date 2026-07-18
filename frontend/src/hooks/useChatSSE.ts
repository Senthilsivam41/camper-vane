import { useState, useCallback, useRef } from 'react';

export interface MetricsData {
  selected_model: string;
  routing_rationale: string;
  estimated_cost_delta: string;
  budget_throttled?: boolean;
  complexity_score?: number;
}

export interface FinalUsageData {
  input_tokens_consumed: number;
  output_tokens_consumed: number;
  updated_daily_total: number;
}

export interface ChatMessage {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  timestamp: Date;
  metrics?: MetricsData;
}

export type StreamStatus = 'idle' | 'connecting' | 'streaming' | 'completed' | 'error';

export function useChatSSE() {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [status, setStatus] = useState<StreamStatus>('idle');
  const [currentMetrics, setCurrentMetrics] = useState<MetricsData | null>(null);
  const [finalUsage, setFinalUsage] = useState<FinalUsageData | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);

  const abortControllerRef = useRef<AbortController | null>(null);

  const sendMessage = useCallback(async (prompt: string, sessionID: string = 'default-session', preferredModel: string = '') => {
    if (!prompt.trim()) return;

    // Reset stream state
    setStatus('connecting');
    setErrorMsg(null);
    setCurrentMetrics(null);

    const userMsg: ChatMessage = {
      id: `user-${Date.now()}`,
      role: 'user',
      content: prompt,
      timestamp: new Date(),
    };

    const assistantMsgId = `assistant-${Date.now()}`;
    const initialAssistantMsg: ChatMessage = {
      id: assistantMsgId,
      role: 'assistant',
      content: '',
      timestamp: new Date(),
    };

    setMessages((prev) => [...prev, userMsg, initialAssistantMsg]);

    if (abortControllerRef.current) {
      abortControllerRef.current.abort();
    }
    abortControllerRef.current = new AbortController();

    try {
      const resp = await fetch('/api/v1/chat/stream', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          session_id: sessionID,
          prompt,
          model: preferredModel,
        }),
        credentials: 'include',
        signal: abortControllerRef.current.signal,
      });

      if (!resp.ok) {
        throw new Error(`HTTP Error (${resp.status}): Failed to establish chat stream`);
      }

      if (!resp.body) {
        throw new Error('Streaming response body empty');
      }

      setStatus('streaming');
      const reader = resp.body.getReader();
      const decoder = new TextDecoder();
      let buffer = '';

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const events = buffer.split('\n\n');
        buffer = events.pop() || ''; // Keep incomplete trailing chunk in buffer

        for (const evtStr of events) {
          if (!evtStr.trim()) continue;

          const lines = evtStr.split('\n');
          let eventName = 'message';
          let dataStr = '';

          for (const l of lines) {
            if (l.startsWith('event: ')) {
              eventName = l.substring(7).trim();
            } else if (l.startsWith('data: ')) {
              dataStr = l.substring(6).trim();
            }
          }

          if (!dataStr) continue;

          try {
            const parsedData = JSON.parse(dataStr);

            if (eventName === 'metrics') {
              const metricsData = parsedData as MetricsData;
              setCurrentMetrics(metricsData);
              setMessages((prev) =>
                prev.map((msg) =>
                  msg.id === assistantMsgId ? { ...msg, metrics: metricsData } : msg
                )
              );
            } else if (eventName === 'text') {
              const textDelta = parsedData.text_delta || '';
              setMessages((prev) =>
                prev.map((msg) =>
                  msg.id === assistantMsgId
                    ? { ...msg, content: msg.content + textDelta }
                    : msg
                )
              );
            } else if (eventName === 'final_usage') {
              setFinalUsage(parsedData as FinalUsageData);
            }
          } catch (e) {
            console.warn('Failed to parse SSE JSON payload:', dataStr, e);
          }
        }
      }

      setStatus('completed');
    } catch (err: any) {
      if (err.name === 'AbortError') {
        setStatus('idle');
        return;
      }
      console.error('SSE Stream error:', err);
      setStatus('error');
      setErrorMsg(err.message || 'Stream connection interrupted. Retry requested.');
    }
  }, []);

  return {
    messages,
    status,
    currentMetrics,
    finalUsage,
    errorMsg,
    sendMessage,
  };
}
