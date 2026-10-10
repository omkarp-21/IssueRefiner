import { BrowserRouter, Navigate, Route, Routes } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import Shell from "./components/layout/Shell";
import { AuthProvider, useAuth } from "./features/auth/AuthContext";
import Login from "./features/auth/Login";
import Overview from "./features/repositories/Overview";
import Repositories from "./features/repositories/Repositories";
import RepositoryIssues from "./features/issues/RepositoryIssues";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      refetchOnWindowFocus: false,
    },
  },
});

function Guard({ children }: { children: React.ReactNode }) {
  const { me, isLoading, isAuthenticated } = useAuth();
  if (isLoading && !me) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <p className="text-sm text-muted">Loading…</p>
      </div>
    );
  }
  if (!isAuthenticated) return <Navigate to="/login" replace />;
  return <>{children}</>;
}

function LoginRoute() {
  const { isAuthenticated, isLoading } = useAuth();
  if (isLoading) return null;
  if (isAuthenticated) return <Navigate to="/" replace />;
  return <Login />;
}

export default function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <BrowserRouter>
        <AuthProvider>
          <Routes>
            <Route path="/login" element={<LoginRoute />} />
            <Route
              element={
                <Guard>
                  <Shell />
                </Guard>
              }
            >
              <Route index element={<Overview />} />
              <Route path="repositories" element={<Repositories />} />
              <Route path="repositories/:id" element={<RepositoryIssues />} />
            </Route>
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </AuthProvider>
      </BrowserRouter>
    </QueryClientProvider>
  );
}
