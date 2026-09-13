import React, { createContext, useContext, useState, useEffect, useCallback, useMemo } from 'react';

interface FlagContextType {
  getFlag: <T = any>(key: string, fallback: T) => T;
  userId: string;
  setUserId: (id: string) => void;
  lastUpdated: number;
}

const FlagContext = createContext<FlagContextType>({
  getFlag: (_, fallback) => fallback,
  userId: 'user_42',
  setUserId: () => {},
  lastUpdated: Date.now(),
});

const STORAGE_KEY = 'ds_feature_flags_cache';

const getApiUrl = (path: string): string => {
  const isCaddyProxy = window.location.port === '' || window.location.port === '80' || window.location.port === '443';
  const prefix = isCaddyProxy ? '/api' : 'http://localhost:8080/api';
  return `${prefix}${path.startsWith('/') ? path : '/' + path}`;
};

export const FeatureFlagProvider: React.FC<{
  children: React.ReactNode;
}> = ({ children }) => {
  const [userId, setUserId] = useState<string>('user_42');
  const [flags, setFlags] = useState<Record<string, any>>(() => {
    try {
      const cached = localStorage.getItem(`${STORAGE_KEY}_${userId}`);
      if (cached) return JSON.parse(cached);
    } catch {}
    return {};
  });
  const [lastUpdated, setLastUpdated] = useState<number>(Date.now());

  // Fetch flags for current user
  const syncFlags = useCallback(async (targetUid: string) => {
    try {
      const res = await fetch(getApiUrl(`/v1/evaluate?userId=${encodeURIComponent(targetUid)}`));
      if (res.ok) {
        const data = await res.json();
        setFlags(data);
        setLastUpdated(Date.now());
        try {
          localStorage.setItem(`${STORAGE_KEY}_${targetUid}`, JSON.stringify(data));
        } catch {}
      }
    } catch (err) {
      console.warn('Feature flag sync error:', err);
    }
  }, []);

  // Initial sync on userId change
  useEffect(() => {
    syncFlags(userId);
  }, [userId, syncFlags]);

  // Real-time SSE Stream listener for instant 0 CLS updates
  useEffect(() => {
    const sse = new EventSource(getApiUrl('/v1/stream'));

    sse.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        // If override matches active user or is a general flag update, re-sync immediately
        if (!data.userId || data.userId === userId) {
          syncFlags(userId);
        }
      } catch {}
    };

    return () => {
      sse.close();
    };
  }, [userId, syncFlags]);

  const getFlag = useCallback(
    <T,>(key: string, fallback: T): T => {
      return flags[key] !== undefined ? (flags[key] as T) : fallback;
    },
    [flags]
  );

  const value = useMemo(
    () => ({ getFlag, userId, setUserId, lastUpdated }),
    [getFlag, userId, lastUpdated]
  );

  return <FlagContext.Provider value={value}>{children}</FlagContext.Provider>;
};

// Custom Hook
export function useFeatureFlag<T = boolean>(flagKey: string, fallback: T): T {
  const { getFlag } = useContext(FlagContext);
  return getFlag<T>(flagKey, fallback);
}

// Custom Hook for User Context
export function useFlagUser() {
  const { userId, setUserId, lastUpdated } = useContext(FlagContext);
  return { userId, setUserId, lastUpdated };
}

// Declarative Component Wrapper
export const FeatureFlag: React.FC<{
  flag: string;
  fallback?: React.ReactNode;
  children: React.ReactNode;
}> = ({ flag, fallback = null, children }) => {
  const isEnabled = useFeatureFlag(flag, false);
  return <>{isEnabled ? children : fallback}</>;
};
