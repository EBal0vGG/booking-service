import { apiFetch } from "@/src/api/client";
import type { UserRole } from "@/src/types/api";

interface TokenResponse {
  token: string;
}

interface RegisterResponse {
  user: {
    id: string;
    email: string;
    role: UserRole;
    createdAt?: string;
  };
}

export function login(email: string, password: string): Promise<TokenResponse> {
  return apiFetch<TokenResponse>("/login", {
    method: "POST",
    body: JSON.stringify({ email, password }),
  });
}

export function register(
  email: string,
  password: string,
  role: UserRole,
): Promise<RegisterResponse> {
  return apiFetch<RegisterResponse>("/register", {
    method: "POST",
    body: JSON.stringify({ email, password, role }),
  });
}

export function dummyLogin(role: UserRole): Promise<TokenResponse> {
  return apiFetch<TokenResponse>("/dummyLogin", {
    method: "POST",
    body: JSON.stringify({ role }),
  });
}
