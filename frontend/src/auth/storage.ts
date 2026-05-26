import type { AuthSession, UserRole } from "@/src/types/api";

const SESSION_KEY = "booking.frontend.session";

interface JwtPayload {
  user_id?: string;
  role?: string;
  exp?: number;
}

function isBrowser(): boolean {
  return typeof window !== "undefined";
}

function decodeBase64Url(value: string): string | null {
  try {
    const normalized = value.replace(/-/g, "+").replace(/_/g, "/");
    const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, "=");
    return atob(padded);
  } catch {
    return null;
  }
}

function parsePayload(token: string): JwtPayload | null {
  const parts = token.split(".");
  if (parts.length !== 3) {
    return null;
  }
  const decoded = decodeBase64Url(parts[1]);
  if (!decoded) {
    return null;
  }
  try {
    return JSON.parse(decoded) as JwtPayload;
  } catch {
    return null;
  }
}

export function sessionFromToken(token: string): AuthSession | null {
  const payload = parsePayload(token);
  if (!payload?.user_id || !payload?.role) {
    return null;
  }
  if (payload.exp && payload.exp * 1000 <= Date.now()) {
    return null;
  }
  if (payload.role !== "admin" && payload.role !== "user") {
    return null;
  }

  return {
    token,
    role: payload.role as UserRole,
    userId: payload.user_id,
  };
}

export function getStoredSession(): AuthSession | null {
  if (!isBrowser()) {
    return null;
  }
  const raw = window.sessionStorage.getItem(SESSION_KEY);
  if (!raw) {
    return null;
  }
  try {
    const parsed = JSON.parse(raw) as AuthSession;
    const validated = sessionFromToken(parsed.token);
    if (!validated) {
      clearStoredSession();
      return null;
    }
    return validated;
  } catch {
    return null;
  }
}

export function setStoredSession(session: AuthSession): void {
  if (!isBrowser()) {
    return;
  }
  window.sessionStorage.setItem(SESSION_KEY, JSON.stringify(session));
}

export function clearStoredSession(): void {
  if (!isBrowser()) {
    return;
  }
  window.sessionStorage.removeItem(SESSION_KEY);
}
