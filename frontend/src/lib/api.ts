import type { ApiError } from "../types";

export class ApiException extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

async function parseError(res: Response): Promise<never> {
  let message = `Request failed (${res.status})`;
  try {
    const body = (await res.json()) as ApiError;
    if (body && typeof body.error === "string" && body.error.length > 0) {
      message = body.error;
    }
  } catch {
    // Keep the default message when the envelope is unreadable.
  }
  throw new ApiException(res.status, message);
}

// All /api/* routes require the HttpOnly session cookie, so every request
// must use credentials: "include" and relative URLs (Vite dev proxy).
export async function apiFetch<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
  });
  if (!res.ok) await parseError(res);
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}
