import { useState, useEffect, useRef, type FormEvent } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Link, useSearchParams } from "react-router-dom";
import { ApiException, apiFetch } from "../../lib/api";
import { renderShortcodes } from "../../lib/emoji";
import type { ReposResponse, Repository } from "../../types";
import {
  AddIcon,
  ArrowRightIcon,
  BranchIcon,
  CheckCircleIcon,
  CodeIcon,
  LockIcon,
  SearchIcon,
} from "../../components/icons";

function formatRelativeTime(dateStr: string): string {
  try {
    const date = new Date(dateStr);
    const now = new Date();
    const diffMs = now.getTime() - date.getTime();
    if (isNaN(diffMs)) return dateStr;
    const diffSec = Math.floor(diffMs / 1000);
    if (diffSec < 60) return "just now";
    const diffMin = Math.floor(diffSec / 60);
    if (diffMin < 60) return `${diffMin}m ago`;
    const diffHour = Math.floor(diffMin / 60);
    if (diffHour < 24) return `${diffHour}h ago`;
    const diffDay = Math.floor(diffHour / 24);
    if (diffDay < 30) return `${diffDay}d ago`;
    return date.toLocaleDateString(undefined, {
      month: "short",
      day: "numeric",
      year: "numeric",
    });
  } catch {
    return dateStr;
  }
}

function RepoCard({ repo }: { repo: Repository }) {
  return (
    <article className="flex flex-col justify-between gap-4 rounded-lg border border-[#2a2a2a] bg-[#161616] p-5 transition-all duration-150 hover:border-[#3e3e3e] hover:bg-[#1a1a1a] sm:p-6">
      <div className="flex flex-col justify-between gap-3 md:flex-row md:items-start">
        <div className="flex min-w-0 items-start gap-3">
          <div className="mt-0.5 flex size-8 shrink-0 items-center justify-center rounded border border-[#2f2f2f] bg-[#202020] text-[#acc7ff]">
            {repo.private ? <LockIcon /> : <CodeIcon />}
          </div>
          <div className="flex min-w-0 flex-col">
            <div className="flex flex-wrap items-center gap-2.5">
              <Link
                to={`/repositories/${repo.id}`}
                className="truncate font-mono text-base font-semibold text-white transition-colors hover:text-[#acc7ff]"
              >
                {repo.full_name}
              </Link>
              <span className="rounded border border-[#333333] bg-[#222222] px-2 py-0.5 font-mono text-xs text-[#8c8c8c]">
                {repo.private ? "Private" : "Public"}
              </span>
              <span className="inline-flex items-center gap-1.5 rounded border border-[#2b2b2b] bg-[#1c1c1c] px-2 py-0.5 font-mono text-xs text-[#8c8c8c]">
                <span className="text-[13px] text-[#888888]">
                  <BranchIcon />
                </span>
                {repo.default_branch}
              </span>
            </div>
            {repo.description && (
              <p className="mt-2 mb-1 max-w-3xl text-sm leading-relaxed text-[#a0a0a0]">
                {renderShortcodes(repo.description)}
              </p>
            )}
          </div>
        </div>
        <div className="flex shrink-0 items-center self-start pt-0.5 md:self-start">
          <Link
            to={`/repositories/${repo.id}`}
            className="inline-flex cursor-pointer items-center gap-1.5 rounded border border-[#333333] bg-[#222222] px-3.5 py-1.5 text-xs font-medium text-[#ededed] transition-colors hover:border-[#444444] hover:bg-[#2a2a2a]"
          >
            <span>Open</span>
            <span className="text-[14px] text-[#888888]">
              <ArrowRightIcon />
            </span>
          </Link>
        </div>
      </div>
      <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-2 border-t border-[#222222] pt-3 font-mono text-xs text-[#888888]">
        <div className="flex flex-wrap items-center gap-3.5">
          <span className="text-[#808080]">
            Updated {formatRelativeTime(repo.updated_at)}
          </span>
        </div>
        <div className="flex items-center gap-1.5 text-[#808080]">
          <span className="text-[14px] text-[#4edea3]">
            <CheckCircleIcon />
          </span>
          <span>Synced {formatRelativeTime(repo.synced_at)}</span>
        </div>
      </div>
    </article>
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
        className="w-full max-w-sm rounded-md border border-[#222222] bg-[#121212] p-5"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-[14px] font-medium text-[#f0f0f0]">
          Import repository
        </h2>
        <p className="mt-1 font-mono text-[11px] text-[#888888]">
          Public GitHub repositories only.
        </p>
        <form onSubmit={onSubmit} className="mt-4 flex flex-col gap-3">
          <label className="flex flex-col gap-1 text-[12px] text-[#888888]">
            Owner
            <input
              value={owner}
              onChange={(e) => setOwner(e.target.value)}
              placeholder="octo"
              aria-label="Repository owner"
              autoFocus
              className="h-[38px] rounded border border-[#222222] bg-[#080808] px-3 font-mono text-[12px] text-[#f0f0f0] outline-none transition-colors placeholder:text-[#555555] focus:border-accent"
            />
          </label>
          <label className="flex flex-col gap-1 text-[12px] text-[#888888]">
            Repository
            <input
              value={repo}
              onChange={(e) => setRepo(e.target.value)}
              placeholder="demo"
              aria-label="Repository name"
              className="h-[38px] rounded border border-[#222222] bg-[#080808] px-3 font-mono text-[12px] text-[#f0f0f0] outline-none transition-colors placeholder:text-[#555555] focus:border-accent"
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
              className="h-8 rounded border border-[#222222] bg-[#121212] px-3 text-[12px] text-[#f0f0f0] transition-colors hover:border-[#555555] hover:bg-[#161616]"
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

export default function Repositories() {
  const [searchParams] = useSearchParams();
  const [filter, setFilter] = useState(() => searchParams.get("q") ?? "");
  const [importOpen, setImportOpen] = useState(
    () => searchParams.get("import") === "1"
  );
  const searchInputRef = useRef<HTMLInputElement>(null);

  const reposQuery = useQuery({
    queryKey: ["repos"],
    queryFn: () => apiFetch<ReposResponse>("/api/repos"),
    retry: false,
  });

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        searchInputRef.current?.focus();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  const repos = reposQuery.data?.repositories ?? [];
  const q = filter.trim().toLowerCase();
  const visible = q
    ? repos.filter(
        (r) =>
          r.full_name.toLowerCase().includes(q) ||
          (r.description && r.description.toLowerCase().includes(q))
      )
    : repos;

  const connectedText = q
    ? `${visible.length} of ${repos.length} connected`
    : `${repos.length} connected`;

  return (
    <div className="mx-auto flex w-full max-w-6xl flex-col">
      {/* Title & Action Bar */}
      <div className="flex flex-col justify-between gap-4 border-b border-[#222222] pb-6 md:flex-row md:items-end">
        <div>
          <h1 className="text-xl font-medium tracking-tight text-white sm:text-2xl">
            Repositories
          </h1>
          <p className="mt-1 text-sm text-[#888888]">
            Connected repositories available for AI issue triage and investigation.
          </p>
        </div>
        <div>
          <button
            type="button"
            onClick={() => setImportOpen(true)}
            className="inline-flex cursor-pointer items-center gap-2 rounded border border-transparent bg-accent px-3.5 py-1.5 text-sm font-medium text-white shadow-sm transition-colors duration-150 hover:bg-accent-hover"
          >
            <span className="text-[18px]">
              <AddIcon />
            </span>
            <span>Import repository</span>
          </button>
        </div>
      </div>

      {/* Controls: Filter & Counter Segment */}
      <div className="flex flex-col items-stretch justify-between gap-3 pt-6 pb-5 sm:flex-row sm:items-center">
        <div className="relative flex-1 max-w-md">
          <div className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-3 text-[#888888]">
            <span className="text-[18px]">
              <SearchIcon />
            </span>
          </div>
          <input
            ref={searchInputRef}
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            className="w-full rounded border border-[#2a2a2a] bg-[#161616] py-1.5 pr-14 pl-9 font-sans text-sm text-[#f0f0f0] transition-colors placeholder:text-[#888888] focus:border-accent focus:ring-1 focus:ring-accent focus:outline-none"
            placeholder="Filter connected repositories..."
            type="text"
            aria-label="Filter connected repositories"
          />
          <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2.5">
            <kbd className="rounded border border-[#353534] bg-[#201f1f] px-1.5 py-0.5 font-mono text-[10px] text-[#888888]">
              ⌘K
            </kbd>
          </div>
        </div>
        <div className="flex items-center gap-1.5 self-end font-mono text-sm text-[#888888] sm:self-auto">
          {reposQuery.isSuccess && <span>{connectedText}</span>}
        </div>
      </div>

      {/* Repositories List Cards / States */}
      {reposQuery.isLoading && (
        <div
          className="flex flex-col gap-4"
          aria-busy="true"
          aria-label="Loading repositories"
        >
          {[1, 2, 3].map((i) => (
            <div
              key={i}
              className="flex animate-pulse flex-col justify-between gap-4 rounded-lg border border-[#2a2a2a] bg-[#161616] p-5 sm:p-6"
            >
              <div className="flex flex-col justify-between gap-3 md:flex-row md:items-start">
                <div className="flex min-w-0 items-start gap-3">
                  <div className="size-8 shrink-0 rounded border border-[#2f2f2f] bg-[#202020]" />
                  <div className="flex flex-col gap-2">
                    <div className="flex items-center gap-2">
                      <div className="h-5 w-44 rounded bg-[#222222]" />
                      <div className="h-5 w-14 rounded bg-[#222222]" />
                      <div className="h-5 w-16 rounded bg-[#222222]" />
                    </div>
                    <div className="h-4 w-72 rounded bg-[#222222]" />
                  </div>
                </div>
                <div className="h-7 w-20 rounded bg-[#222222]" />
              </div>
              <div className="border-t border-[#222222] pt-3">
                <div className="h-3 w-32 rounded bg-[#222222]" />
              </div>
            </div>
          ))}
        </div>
      )}

      {reposQuery.isError && (
        <div
          role="alert"
          className="rounded-lg border border-[#2a2a2a] bg-[#161616] p-8 text-center"
        >
          <p className="text-sm font-medium text-[#e06c75]">
            {reposQuery.error instanceof ApiException
              ? reposQuery.error.message
              : "Failed to load repositories."}
          </p>
          <p className="mt-1 text-xs text-[#888888]">
            Check your network connection and session status.
          </p>
        </div>
      )}

      {reposQuery.isSuccess && repos.length === 0 && (
        <div className="rounded-lg border border-[#2a2a2a] bg-[#161616] p-12 text-center">
          <div className="mx-auto mb-3 flex size-10 items-center justify-center rounded border border-[#2f2f2f] bg-[#202020] text-[#acc7ff]">
            <CodeIcon />
          </div>
          <p className="text-base font-medium text-white">
            No repositories connected yet
          </p>
          <p className="mx-auto mt-1 max-w-sm text-sm text-[#888888]">
            Connect a public GitHub repository to begin issue triage and investigation.
          </p>
          <div className="mt-5">
            <button
              type="button"
              onClick={() => setImportOpen(true)}
              className="inline-flex cursor-pointer items-center gap-2 rounded bg-accent px-3.5 py-1.5 text-sm font-medium text-white transition-colors hover:bg-accent-hover"
            >
              <AddIcon />
              <span>Import repository</span>
            </button>
          </div>
        </div>
      )}

      {reposQuery.isSuccess && repos.length > 0 && visible.length === 0 && (
        <div className="rounded-lg border border-[#2a2a2a] bg-[#161616] p-12 text-center">
          <p className="text-base font-medium text-white">
            No repositories match “{filter.trim()}”
          </p>
          <p className="mt-1 text-sm text-[#888888]">
            Try searching for a different repository name or clear the filter.
          </p>
          <div className="mt-4">
            <button
              type="button"
              onClick={() => setFilter("")}
              className="inline-flex cursor-pointer items-center gap-1.5 rounded border border-[#333333] bg-[#222222] px-3.5 py-1.5 text-xs font-medium text-[#ededed] transition-colors hover:bg-[#2a2a2a]"
            >
              Clear filter
            </button>
          </div>
        </div>
      )}

      {visible.length > 0 && (
        <div className="flex flex-col gap-4">
          {visible.map((r) => (
            <RepoCard key={r.id} repo={r} />
          ))}
        </div>
      )}

      <ImportDialog open={importOpen} onClose={() => setImportOpen(false)} />
    </div>
  );
}
