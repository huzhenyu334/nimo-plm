import { useEffect, useRef, useCallback } from 'react';
import { useQueryClient } from '@tanstack/react-query';

export interface SSETaskEvent {
  project_id: string;
  task_id: string;
  action: string;
}

interface UseSSEOptions {
  onTaskUpdate?: (event: SSETaskEvent) => void;
  onProjectUpdate?: (event: SSETaskEvent) => void;
  onMyTaskUpdate?: (event: SSETaskEvent) => void;
  enabled?: boolean;
}

export function useSSE({ onTaskUpdate, onProjectUpdate, onMyTaskUpdate, enabled = true }: UseSSEOptions) {
  const eventSourceRef = useRef<EventSource | null>(null);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout>>(undefined);
  const onTaskUpdateRef = useRef(onTaskUpdate);
  const onProjectUpdateRef = useRef(onProjectUpdate);
  const onMyTaskUpdateRef = useRef(onMyTaskUpdate);
  const queryClient = useQueryClient();

  // Keep callback refs up to date without causing reconnections
  useEffect(() => {
    onTaskUpdateRef.current = onTaskUpdate;
  }, [onTaskUpdate]);

  useEffect(() => {
    onProjectUpdateRef.current = onProjectUpdate;
  }, [onProjectUpdate]);

  useEffect(() => {
    onMyTaskUpdateRef.current = onMyTaskUpdate;
  }, [onMyTaskUpdate]);

  const connect = useCallback(() => {
    const token = localStorage.getItem('access_token');
    if (!token) return;

    // Close existing connection
    if (eventSourceRef.current) {
      eventSourceRef.current.close();
    }

    const baseUrl = window.location.origin;
    const url = `${baseUrl}/api/v1/sse/events?token=${encodeURIComponent(token)}`;
    const es = new EventSource(url);
    eventSourceRef.current = es;

    es.addEventListener('connected', () => {
      console.log('[SSE] Connected');
    });

    es.addEventListener('task_update', (e: MessageEvent) => {
      try {
        const data: SSETaskEvent = JSON.parse(e.data);
        onTaskUpdateRef.current?.(data);
      } catch (err) {
        console.error('[SSE] Failed to parse task_update event:', err);
      }
    });

    es.addEventListener('project_update', (e: MessageEvent) => {
      try {
        const data: SSETaskEvent = JSON.parse(e.data);
        onProjectUpdateRef.current?.(data);
      } catch (err) {
        console.error('[SSE] Failed to parse project_update event:', err);
      }
    });

    es.addEventListener('my_task_update', (e: MessageEvent) => {
      try {
        const data: SSETaskEvent = JSON.parse(e.data);
        onMyTaskUpdateRef.current?.(data);
      } catch (err) {
        console.error('[SSE] Failed to parse my_task_update event:', err);
      }
    });

    es.onerror = () => {
      console.warn('[SSE] Connection error, reconnecting in 3s...');
      es.close();
      eventSourceRef.current = null;
      // Auto-reconnect after 3 seconds
      reconnectTimerRef.current = setTimeout(() => {
        if (enabled) connect();
      }, 3000);
    };
  }, [enabled]);

  // Refresh data when page becomes visible again
  useEffect(() => {
    const handleVisibilityChange = () => {
      if (document.visibilityState === 'visible') {
        console.log('[SSE] Page visible, invalidating queries');
        queryClient.invalidateQueries();
      }
    };

    const handleFocus = () => {
      console.log('[SSE] Window focused, invalidating queries');
      queryClient.invalidateQueries();
    };

    document.addEventListener('visibilitychange', handleVisibilityChange);
    window.addEventListener('focus', handleFocus);

    return () => {
      document.removeEventListener('visibilitychange', handleVisibilityChange);
      window.removeEventListener('focus', handleFocus);
    };
  }, [queryClient]);

  useEffect(() => {
    if (enabled) {
      connect();
    }
    return () => {
      if (eventSourceRef.current) {
        eventSourceRef.current.close();
        eventSourceRef.current = null;
      }
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current);
      }
    };
  }, [connect, enabled]);
}
