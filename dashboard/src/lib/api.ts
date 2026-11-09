const CONTROL_PLANE_URL = process.env.NEXT_PUBLIC_CONTROL_PLANE_URL || "http://localhost:8081";
export const GATEWAY_URL = process.env.NEXT_PUBLIC_GATEWAY_URL || "http://localhost:8000";

export function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("gateway_auth_token");
}

export function setToken(token: string) {
  if (typeof window !== "undefined") {
    localStorage.setItem("gateway_auth_token", token);
  }
}

export function clearToken() {
  if (typeof window !== "undefined") {
    localStorage.removeItem("gateway_auth_token");
    localStorage.removeItem("gateway_user");
  }
}

export async function apiRequest<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
  const token = getToken();
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string> || {}),
  };

  if (token) {
    headers["Authorization"] = `Bearer ${token}`;
  }

  const res = await fetch(`${CONTROL_PLANE_URL}${endpoint}`, {
    ...options,
    headers,
  });

  if (!res.ok) {
    const errorBody = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(errorBody.error || errorBody.message || `Request failed with status ${res.status}`);
  }

  return res.json();
}

