import { useState, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { ApiException, apiFetch } from "../../lib/api";
import { renderShortcodes } from "../../lib/emoji";
import type { ReposResponse, Repository } from "../../types";
import { AddIcon, ArrowIcon, SearchIcon } from "../../components/icons";

function RepoCard({ repo }: { repo: Repository }) {
  return (
    <div className="flex min-w-0 flex-col justify-between rounded border border-border bg-surface p-4 transition-colors hover:border-subtle">
      <div>
        <div className="mb-2 flex items-center justify-between gap-2">
          <span className="truncate font-mono text-[13px] font-medium text-foreground">
            {repo.full_name}
          </span>
          <span className="shrink-0 rounded border border-border px-1.5 py-0.5 font-mono text-[10px] text-muted">
            {repo.private ? "Private" : "Public"}
          </span>
        </div>
        {repo.description && (
          <p className="text-[12px] leading-relaxed text-muted line-clamp-2">
            {renderShortcodes(repo.description)}
          </p>
        )}
      </div>
      <div className="mt-4 flex items-center justify-between border-t border-border pt-3 font-mono text-[11px] text-muted">
        <span className="truncate">{repo.default_branch}</span>
        <a
          href={repo.html_url}
          target="_blank"
          rel="noreferrer"
          className="font-medium text-white transition-colors hover:text-accent"
        >
          Open
        </a>
      </div>
    </div>
  );
}

function ImportDialog({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const queryClient = useQueryClient();
  const [owner, setOwner] = useState("");
  const [repo, setRepo] = useState("");

  const importMutation = useMutation({
    mutationFn: () =>
      apiFetch<{ repository: Repository }>("/api/repos/import", {
        method: "POST",
        body: JSON.stringify({ owner: owner.trim(), repo: repo.trim() }),
      }),
    onSuccess: () => {
      setOwner("");
      setRepo("");
      void queryClient.invalidateQueries({ queryKey: ["repos"] });
      onClose();
    },
  });

  if (!open) return null;

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (owner.trim() && repo.trim() && !importMutation.isPending) {
      importMutation.mutate();
    }
  };

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 p-4"
      role="dialog"
      aria-modal="true"
      aria-label="Import repository"
      onClick={onClose}
    >
      <div
        className="w-full max-w-sm rounded-md border border-border bg-surface p-5"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-[14px] font-medium text-foreground">
          Import repository
        </h2>
        <p className="mt-1 font-mono text-[11px] text-muted">
          Public GitHub repositories only.
        </p>
        <form onSubmit={onSubmit} className="mt-4 flex flex-col gap-3">
          <label className="flex flex-col gap-1 text-[12px] text-muted">
            Owner
            <input
              value={owner}
              onChange={(e) => setOwner(e.target.value)}
              placeholder="octo"
              aria-label="Repository owner"
              autoFocus
              className="h-[38px] rounded border border-border bg-background px-3 font-mono text-[12px] text-foreground outline-none transition-colors placeholder:text-subtle focus:border-accent"
            />
          </label>
          <label className="flex flex-col gap-1 text-[12px] text-muted">
            Repository
            <input
              value={repo}
              onChange={(e) => setRepo(e.target.value)}
              placeholder="demo"
              aria-label="Repository name"
              className="h-[38px] rounded border border-border bg-background px-3 font-mono text-[12px] text-foreground outline-none transition-colors placeholder:text-subtle focus:border-accent"
            />
          </label>
          {importMutation.isError && (
            <p role="alert" className="text-[12px] text-[#e06c75]">
              {importMutation.error instanceof ApiException
                ? importMutation.error.message
                : "Import failed."}
            </p>
          )}
          <div className="mt-1 flex justify-end gap-2">
            <button
              type="button"
              onClick={onClose}
              className="h-8 rounded border border-border bg-surface px-3 text-[12px] text-foreground transition-colors hover:border-subtle hover:bg-raised"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={
                importMutation.isPending || !owner.trim() || !repo.trim()
              }
              className="h-8 rounded bg-accent px-3 text-[12px] font-medium text-white transition-colors hover:bg-accent-hover disabled:opacity-50"
            >
              {importMutation.isPending ? "Importing…" : "Import"}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}

export default function Overview() {
  const [filter, setFilter] = useState("");
  const [importOpen, setImportOpen] = useState(false);

  const reposQuery = useQuery({
    queryKey: ["repos"],
    queryFn: () => apiFetch<ReposResponse>("/api/repos"),
    retry: false,
  });

  const repos = reposQuery.data?.repositories ?? [];
  const q = filter.trim().toLowerCase();
  const visible = q
    ? repos.filter((r) => r.full_name.toLowerCase().includes(q))
    : repos;

  return (
    <div className="flex flex-col gap-8">
      {/* Page title & actions */}
      <div className="flex flex-col justify-between gap-4 pb-2 sm:flex-row sm:items-center">
        <div>
          <h1 className="text-xl font-semibold tracking-tight text-foreground">
            Overview
          </h1>
          <p className="mt-0.5 text-[13px] text-muted">
            Connected repositories and recent issue investigations across your
            GitHub workspace.
          </p>
        </div>
        <div className="flex items-center gap-2.5">
          <div className="relative flex items-center">
            <span className="pointer-events-none absolute left-2.5 text-[15px] text-muted">
              <SearchIcon />
            </span>
            <input
              value={filter}
              onChange={(e) => setFilter(e.target.value)}
              placeholder="Filter repositories..."
              aria-label="Filter repositories"
              className="h-8 w-56 rounded border border-border bg-surface pl-8 pr-3 font-mono text-[12px] text-foreground outline-none transition-colors placeholder:text-muted focus:border-subtle"
            />
          </div>
          <button
            type="button"
            onClick={() => setImportOpen(true)}
            className="flex h-8 items-center gap-1.5 rounded bg-accent px-3 text-[12px] font-medium text-white transition-colors hover:bg-accent-hover"
          >
            <span className="text-[15px]">
              <AddIcon />
            </span>
            <span>Import repository</span>
          </button>
        </div>
      </div>

      {/* Connected repositories */}
      <section className="flex flex-col gap-3" aria-label="Connected repositories">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <h2 className="text-[14px] font-medium text-foreground">
              Connected repositories
            </h2>
            {reposQuery.isSuccess && (
              <span className="font-mono text-xs text-muted">
                {repos.length}
              </span>
            )}
          </div>
          {/* Label follows the task correction ("Manage repositories"); the
              Stitch source calls this "Manage integrations", but no
              integrations management exists — this navigates to the real
              repository-list route. */}
          <Link
            to="/repositories"
            className="flex items-center gap-1 text-[12px] text-muted transition-colors hover:text-foreground"
          >
            <span>Manage repositories</span>
            <span className="text-[13px]">
              <ArrowIcon />
            </span>
          </Link>
        </div>
        {reposQuery.isLoading && (
          <p className="rounded border border-border bg-surface px-4 py-6 text-[13px] text-muted">
            Loading repositories…
          </p>
        )}
        {reposQuery.isError && (
          <p
            role="alert"
            className="rounded border border-border bg-surface px-4 py-6 text-[13px] text-[#e06c75]"
          >
            {reposQuery.error instanceof ApiException
              ? reposQuery.error.message
              : "Could not load repositories."}
          </p>
        )}
        {reposQuery.isSuccess && repos.length === 0 && (
          <div className="rounded border border-border bg-surface px-4 py-8 text-center">
            <p className="text-[13px] text-foreground">
              No repositories connected yet.
            </p>
            <p className="mt-1 text-[12px] text-muted">
              Use Import repository to connect a public GitHub repository.
            </p>
          </div>
        )}
        {reposQuery.isSuccess && repos.length > 0 && visible.length === 0 && (
          <div className="rounded border border-border bg-surface px-4 py-8 text-center">
            <p className="text-[13px] text-foreground">
              No repositories match “{filter.trim()}”.
            </p>
          </div>
        )}
        {visible.length > 0 && (
          <div className="grid grid-cols-1 gap-3 md:grid-cols-3 2xl:grid-cols-4">
            {visible.map((r) => (
              <RepoCard key={r.id} repo={r} />
            ))}
          </div>
        )}
      </section>

      {/* Recent investigations. The backend exposes investigation history only
          per repository + issue plus single-job lookup — there is no
          cross-workspace listing endpoint — so this section is a deliberate
          empty state, not fabricated history. */}
      <section className="flex flex-col gap-3" aria-label="Recent investigations">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <h2 className="text-[14px] font-medium text-foreground">
              Recent investigations
            </h2>
          </div>
        </div>
        <div className="rounded border border-border bg-surface px-4 py-8 text-center">
          <p className="text-[13px] text-foreground">
            No recent investigations to show.
          </p>
          <p className="mx-auto mt-1 max-w-md text-[12px] text-muted">
            Investigations you run on repository issues will appear here once a
            workspace-wide history is available.
          </p>
        </div>
      </section>

      <ImportDialog open={importOpen} onClose={() => setImportOpen(false)} />
    </div>
  );
}
