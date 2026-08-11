import { useState, useCallback, useRef } from 'react';
import { fetchSessionMessages } from '../services/api';

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

export type StreamStatus = 'idle' | 'connecting' | 'streaming' | 'retrying' | 'completed' | 'error';

const MAX_RETRIES = 2;
const RETRY_BASE_MS = 700;

function isRetryableError(err: unknown): boolean {
  if (!err || typeof err !== 'object') return false;
  const e = err as { name?: string; message?: string };
  if (e.name === 'AbortError') return false;
  const msg = (e.message || '').toLowerCase();
  return (
    msg.includes('failed to fetch') ||
    msg.includes('network') ||
    msg.includes('interrupted') ||
    msg.includes('http error (502)') ||
    msg.includes('http error (503)') ||
    msg.includes('http error (504)')
  );
}

export function useChatSSE() {
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [status, setStatus] = useState<StreamStatus>('idle');
  const [currentMetrics, setCurrentMetrics] = useState<MetricsData | null>(null);
  const [finalUsage, setFinalUsage] = useState<FinalUsageData | null>(null);
  const [errorMsg, setErrorMsg] = useState<string | null>(null);
  const [retryCount, setRetryCount] = useState(0);

  const abortControllerRef = useRef<AbortController | null>(null);

  const clearMessages = useCallback(() => {
    setMessages([]);
    setCurrentMetrics(null);
    setFinalUsage(null);
    setErrorMsg(null);
    setStatus('idle');
    setRetryCount(0);
  }, []);

  const loadSessionHistory = useCallback(async (sessionID: string) => {
    const history = await fetchSessionMessages(sessionID);
    const mapped: ChatMessage[] = history
      .filter((m) => m.role === 'user' || m.role === 'assistant')
      .map((m, idx) => ({
        id: `${sessionID}-${idx}-${m.timestamp}`,
        role: m.role as 'user' | 'assistant',
        content: m.content,
        timestamp: new Date(m.timestamp),
      }));
    setMessages(mapped);
    setCurrentMetrics(null);
    setErrorMsg(null);
    setStatus('idle');
  }, []);

  const streamOnce = useCallback(
    async (
      prompt: string,
      sessionID: string,
      preferredModel: string,
      assistantMsgId: string,
      signal: AbortSignal
    ) => {
      const resp = await fetch('/api/v1/chat/stream', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          session_id: sessionID,
          prompt,
          model: preferredModel,
        }),
        credentials: 'include',
        signal,
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
      let sawError = false;

      while (true) {
        const { done, value } = await reader.read();
        if (done) break;

        buffer += decoder.decode(value, { stream: true });
        const events = buffer.split('\n\n');
        buffer = events.pop() || '';

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
                prev.map((msg) => (msg.id === assistantMsgId ? { ...msg, metrics: metricsData } : msg))
              );
            } else if (eventName === 'text') {
              const textDelta = parsedData.text_delta || '';
              setMessages((prev) =>
                prev.map((msg) =>
                  msg.id === assistantMsgId ? { ...msg, content: msg.content + textDelta } : msg
                )
              );
            } else if (eventName === 'final_usage') {
              setFinalUsage(parsedData as FinalUsageData);
            } else if (eventName === 'error') {
              sawError = true;
              const message = parsedData.message || 'Provider stream failed';
              setStatus('error');
              setErrorMsg(message);
              setMessages((prev) =>
                prev.map((msg) =>
                  msg.id === assistantMsgId
                    ? { ...msg, content: msg.content || `Error: ${message}` }
                    : msg
                )
              );
            }
          } catch (e) {
            console.warn('Failed to parse SSE JSON payload:', dataStr, e);
          }
        }
      }

      if (!sawError) {
        setStatus('completed');
      }
      return !sawError;
    },
    []
  );

  const sendMessage = useCallback(
    async (prompt: string, sessionID: string = 'default-session', preferredModel: string = '') => {
      if (!prompt.trim()) return;

      setStatus('connecting');
      setErrorMsg(null);
      setCurrentMetrics(null);
      setRetryCount(0);

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
      const signal = abortControllerRef.current.signal;

      let attempt = 0;
      while (attempt <= MAX_RETRIES) {
        try {
          // Reset assistant bubble content on retry
          if (attempt > 0) {
            setStatus('retrying');
            setRetryCount(attempt);
            setMessages((prev) =>
              prev.map((msg) => (msg.id === assistantMsgId ? { ...msg, content: '', metrics: undefined } : msg))
            );
            await new Promise((resolve) => setTimeout(resolve, RETRY_BASE_MS * attempt));
            if (signal.aborted) {
              setStatus('idle');
              return;
            }
          }

          await streamOnce(prompt, sessionID, preferredModel, assistantMsgId, signal);
          return;
        } catch (err: any) {
          if (err?.name === 'AbortError') {
            setStatus('idle');
            return;
          }
          if (attempt < MAX_RETRIES && isRetryableError(err)) {
            attempt += 1;
            continue;
          }
          console.error('SSE Stream error:', err);
          setStatus('error');
          setErrorMsg(err.message || 'Stream connection interrupted.');
          setMessages((prev) =>
            prev.map((msg) =>
              msg.id === assistantMsgId
                ? { ...msg, content: msg.content || `Error: ${err.message || 'stream failed'}` }
                : msg
            )
          );
          return;
        }
      }
    },
    [streamOnce]
  );

  const cancel = useCallback(() => {
    abortControllerRef.current?.abort();
  }, []);

  return {
    messages,
    status,
    currentMetrics,
    finalUsage,
    errorMsg,
    retryCount,
    sendMessage,
    loadSessionHistory,
    clearMessages,
    cancel,
  };
}
