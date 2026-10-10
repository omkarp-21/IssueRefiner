import { NavLink, Outlet, useLocation } from "react-router-dom";
import { useAuth } from "../../features/auth/AuthContext";
import {
  DomainIcon,
  GridIcon,
  LogoMark,
  LogoutIcon,
  RepoIcon,
} from "../icons";

function initials(name: string): string {
  const parts = name.trim().split(/\s+/);
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

export default function Shell() {
  const { me, logout } = useAuth();
  const location = useLocation();
  const onRepositories = location.pathname.startsWith("/repositories");

  const displayName = me ? me.user.display_name || me.user.username : "";
  const orgName = me?.organization.name ?? "";

  return (
    <div className="min-h-screen bg-background text-foreground antialiased">
      {/* Sidebar */}
      <aside className="fixed left-0 top-0 z-50 flex h-full w-60 select-none flex-col justify-between border-r border-border bg-surface">
        <div className="flex flex-col">
          <div className="border-b border-border p-4">
            <div className="mb-3 flex items-center gap-2">
              <div className="flex h-5 w-5 items-center justify-center rounded bg-foreground text-[14px] text-background">
                <LogoMark />
              </div>
              <span className="text-[14px] font-medium tracking-tight text-foreground">
                Issue Refiner
              </span>
            </div>
            {/* Workspace indicator. The backend pins a session to one
                organization with no switching endpoint, so this is a static
                indicator, not a switcher. */}
            <div
              className="flex w-full items-center justify-between rounded border border-border bg-raised px-2.5 py-1.5 text-left"
              title="Organization (switching is not supported)"
            >
              <div className="flex min-w-0 items-center gap-2">
                <span className="text-[15px] text-muted">
                  <DomainIcon />
                </span>
                <span className="truncate font-mono text-[12px] text-foreground">
                  {orgName}
                </span>
              </div>
            </div>
          </div>
          <div className="px-3 py-4">
            <nav className="space-y-1" aria-label="Primary">
              <NavLink
                to="/"
                end
                className={({ isActive }) =>
                  isActive || (!onRepositories && location.pathname === "/")
                    ? "flex items-center gap-2.5 rounded px-2.5 py-1.5 text-[13px] font-medium text-accent transition-colors bg-navactive"
                    : "flex items-center gap-2.5 rounded px-2.5 py-1.5 text-[13px] text-muted transition-colors hover:bg-raised hover:text-foreground"
                }
              >
                <span className="text-[17px]">
                  <GridIcon />
                </span>
                <span>Overview</span>
              </NavLink>
              <NavLink
                to="/repositories"
                className={({ isActive }) =>
                  isActive || onRepositories
                    ? "flex items-center gap-2.5 rounded px-2.5 py-1.5 text-[13px] font-medium text-accent transition-colors bg-navactive"
                    : "flex items-center gap-2.5 rounded px-2.5 py-1.5 text-[13px] text-muted transition-colors hover:bg-raised hover:text-foreground"
                }
              >
                <span className="text-[17px]">
                  <RepoIcon />
                </span>
                <span>Repositories</span>
              </NavLink>
            </nav>
          </div>
        </div>
        <div className="border-t border-border p-3">
          <div className="flex items-center justify-between px-2 py-2">
            <div className="flex min-w-0 items-center gap-2.5">
              <div className="flex h-7 w-7 shrink-0 items-center justify-center rounded border border-border bg-raised font-mono text-[11px] font-medium text-foreground">
                {initials(displayName || "?")}
              </div>
              <div className="flex min-w-0 flex-col">
                <span className="truncate text-[13px] font-medium leading-tight text-foreground">
                  {displayName}
                </span>
              </div>
            </div>
            <button
              type="button"
              aria-label="Sign out"
              title="Sign out"
              onClick={logout}
              className="shrink-0 p-1 text-muted transition-colors hover:text-foreground"
            >
              <span className="block text-[17px]">
                <LogoutIcon />
              </span>
            </button>
          </div>
        </div>
      </aside>

      {/* Main column */}
      <div className="pl-60">
        {/* Top header frame. The Stitch breadcrumb ("Repositories /
            acme/payment-service / #142") and its right-side search,
            notifications, and avatar controls have no Overview data source or
            backend behavior behind them, so the header renders as an empty
            frame with the approved dimensions until those controls are
            specified. No inert controls are added. */}
        <header className="fixed left-60 right-0 top-0 z-40 flex h-10 select-none items-center justify-between border-b border-border bg-surface/80 px-6 backdrop-blur-md" />
        <main className="min-h-screen pt-10">
          <div className="px-8 py-8">
            <Outlet />
          </div>
        </main>
      </div>
    </div>
  );
}
