import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { apiFetch } from "../../lib/api";
import type { MeResponse } from "../../types";

interface AuthState {
  me: MeResponse | null;
  isLoading: boolean;
  isAuthenticated: boolean;
  logout: () => Promise<void>;
}

const AuthContext = createContext<AuthState>({
  me: null,
  isLoading: true,
  isAuthenticated: false,
  logout: async () => {},
});

export function AuthProvider({ children }: { children: ReactNode }) {
  const queryClient = useQueryClient();
  const [logoutPending, setLogoutPending] = useState(false);

  const { data, isLoading, isError } = useQuery({
    queryKey: ["me"],
    queryFn: () => apiFetch<MeResponse>("/api/me"),
    retry: false,
    staleTime: 60_000,
  });

  useEffect(() => {
    if (isError) queryClient.setQueryData(["me"], null);
  }, [isError, queryClient]);

  const logout = async () => {
    setLogoutPending(true);
    try {
      // Requires a valid session; a 401 means the cookie is already dead,
      // which we treat as logged out either way.
      await apiFetch("/api/logout", { method: "POST" });
    } catch {
      // Intentionally ignored: logged out is logged out.
    } finally {
      queryClient.setQueryData(["me"], null);
      await queryClient.invalidateQueries({ queryKey: ["me"] });
      setLogoutPending(false);
    }
  };

  return (
    <AuthContext.Provider
      value={{
        me: data ?? null,
        isLoading: isLoading || logoutPending,
        isAuthenticated: Boolean(data),
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  );
}

export function useAuth(): AuthState {
  return useContext(AuthContext);
}
