import { getStoredSession } from "@/src/auth/storage";
import type { ErrorResponse } from "@/src/types/api";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

interface ApiErrorParams {
  status: number;
  code: string;
  message: string;
}

export class ApiError extends Error {
  public readonly status: number;
  public readonly code: string;

  public constructor(params: ApiErrorParams) {
    super(params.message);
    this.name = "ApiError";
    this.status = params.status;
    this.code = params.code;
  }
}

function absoluteUrl(path: string): string {
  if (path.startsWith("http://") || path.startsWith("https://")) {
    return path;
  }
  return `${API_BASE_URL}${path}`;
}

async function parseJson(response: Response): Promise<unknown> {
  const text = await response.text();
  if (!text) {
    return null;
  }

  try {
    return JSON.parse(text) as unknown;
  } catch {
    return null;
  }
}

function toApiError(response: Response, payload: unknown): ApiError {
  const errorPayload = payload as ErrorResponse | null;
  const code = errorPayload?.error?.code ?? `HTTP_${response.status}`;
  const message =
    errorPayload?.error?.message ?? `Request failed with status ${response.status}`;
  return new ApiError({ status: response.status, code, message });
}

export async function apiFetch<T>(
  path: string,
  options: RequestInit = {},
): Promise<T> {
  const headers = new Headers(options.headers ?? {});
  const session = getStoredSession();
  if (session?.token) {
    headers.set("Authorization", `Bearer ${session.token}`);
  }
  if (options.body && !headers.has("Content-Type")) {
    headers.set("Content-Type", "application/json");
  }

  const response = await fetch(absoluteUrl(path), {
    ...options,
    headers,
    cache: "no-store",
  });

  const payload = await parseJson(response);
  if (!response.ok) {
    throw toApiError(response, payload);
  }

  return payload as T;
}
