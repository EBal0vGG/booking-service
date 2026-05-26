"use client";

import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
} from "react";

import type { AuthSession } from "@/src/types/api";
import {
  clearStoredSession,
  getStoredSession,
  sessionFromToken,
  setStoredSession,
} from "@/src/auth/storage";

interface AuthContextValue {
  ready: boolean;
  session: AuthSession | null;
  applyToken: (token: string) => boolean;
  logout: () => void;
}

const AuthContext = createContext<AuthContextValue | undefined>(undefined);

export function AuthProvider({ children }: { children: React.ReactNode }) {
  const [session, setSession] = useState<AuthSession | null>(() => getStoredSession());

  const applyToken = useCallback((token: string) => {
    const parsed = sessionFromToken(token);
    if (!parsed) {
      return false;
    }
    setStoredSession(parsed);
    setSession(parsed);
    return true;
  }, []);

  const logout = useCallback(() => {
    clearStoredSession();
    setSession(null);
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({ ready: true, session, applyToken, logout }),
    [applyToken, logout, session],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const value = useContext(AuthContext);
  if (!value) {
    throw new Error("useAuth must be used inside AuthProvider");
  }
  return value;
}
