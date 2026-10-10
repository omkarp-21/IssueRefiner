import { useState, useEffect, useRef } from "react";
import { useParams, Link, useSearchParams } from "react-router-dom";
import { useQuery } from "@tanstack/react-query";
import { ApiException, apiFetch } from "../../lib/api";
import { renderShortcodes } from "../../lib/emoji";
import type { RepoResponse, IssuesResponse } from "../../types";
import {
  ArrowLeftIcon,
  BranchIcon,
  ExternalLinkIcon,
  LockIcon,
  SearchIcon,
} from "../../components/icons";

function formatRelativeTime(dateStr?: string): string {
  if (!dateStr) return "";
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

function getIssueLabelClass(label: string): string {
  const l = label.toLowerCase();
  if (l.includes("bug")) {
    return "bg-[#1f1616] text-[#ffb4ab] border border-[#3c2222]";
  }
  if (l.includes("enhancement") || l.includes("feature")) {
    return "bg-[#15231c] text-[#4edea3] border border-[#1f3a2c]";
  }
  return "bg-surface-raised text-muted border border-[#222222]";
}

export default function RepositoryIssues() {
  const { id } = useParams<{ id: string }>();
  const [searchParams] = useSearchParams();
  const [stateFilter, setStateFilter] = useState<"open" | "closed">(
    () => (searchParams.get("state") === "closed" ? "closed" : "open")
  );
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState(() => searchParams.get("q") ?? "");
  const searchInputRef = useRef<HTMLInputElement>(null);

  const repoQuery = useQuery({
    queryKey: ["repo", id],
    queryFn: () => apiFetch<RepoResponse>(`/api/repos/${id}`),
    enabled: Boolean(id),
    retry: false,
  });

  const issuesQuery = useQuery({
    queryKey: ["repo-issues", id, stateFilter, page],
    queryFn: () =>
      apiFetch<IssuesResponse>(
        `/api/repos/${id}/issues?state=${stateFilter}&page=${page}&per_page=30`
      ),
    enabled: Boolean(id),
    retry: false,
  });

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (
        e.key === "/" &&
        document.activeElement?.tagName !== "INPUT" &&
        document.activeElement?.tagName !== "TEXTAREA"
      ) {
        e.preventDefault();
        searchInputRef.current?.focus();
        searchInputRef.current?.select();
      }
    };
    window.addEventListener("keydown", handleKeyDown);
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, []);

  const repo = repoQuery.data?.repository;
  const issues = issuesQuery.data?.issues ?? [];

  const q = search.trim().toLowerCase();
  const visibleIssues = q
    ? issues.filter((issue) => {
        const num = String(issue.Number ?? issue.number ?? "");
        const title = (issue.Title ?? issue.title ?? "").toLowerCase();
        const body = (issue.Body ?? issue.body ?? "").toLowerCase();
        const author = (issue.Author ?? issue.author ?? "").toLowerCase();
        return (
          title.includes(q) ||
          num.includes(q) ||
          `#${num}`.includes(q) ||
          author.includes(q) ||
          body.includes(q)
        );
      })
    : issues;

  if (repoQuery.isError) {
    const err = repoQuery.error;
    return (
      <div className="mx-auto flex w-full max-w-[1280px] flex-col gap-6 px-6 py-6">
        <Link
          to="/repositories"
          className="inline-flex items-center gap-1.5 font-mono text-xs text-muted transition-colors hover:text-foreground"
        >
          <ArrowLeftIcon />
          <span>Back to Repositories</span>
        </Link>
        <div
          role="alert"
          className="rounded-lg border border-[#222222] bg-[#161616] p-8 text-center"
        >
          <p className="text-sm font-medium text-[#e06c75]">
            {err instanceof ApiException ? err.message : "Repository not found."}
          </p>
          <p className="mt-1 text-xs text-muted">
            The requested repository ID may be invalid or inaccessible.
          </p>
          <div className="mt-4">
            <Link
              to="/repositories"
              className="inline-flex items-center gap-1.5 rounded border border-[#333333] bg-[#222222] px-3.5 py-1.5 font-mono text-xs font-medium text-[#ededed] transition-colors hover:bg-[#2a2a2a]"
            >
              Return to Repositories list
            </Link>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto flex w-full max-w-[1280px] flex-col gap-6 px-6 py-6">
      {/* Back Link */}
      <div>
        <Link
          to="/repositories"
          className="inline-flex items-center gap-1.5 font-mono text-xs text-muted transition-colors hover:text-foreground"
        >
          <ArrowLeftIcon />
          <span>Repositories</span>
        </Link>
      </div>

      {/* Repository Identity Row */}
      <section className="flex flex-col justify-between gap-4 border-b border-[#222222] pb-5 md:flex-row md:items-start">
        <div className="flex max-w-3xl flex-col gap-1.5">
          {repoQuery.isLoading ? (
            <div className="flex animate-pulse flex-col gap-2">
              <div className="h-6 w-56 rounded bg-[#222222]" />
              <div className="h-4 w-96 rounded bg-[#222222]" />
            </div>
          ) : repo ? (
            <>
              <div className="flex flex-wrap items-center gap-2.5">
                <span className="font-sans text-xl font-medium tracking-tight text-foreground sm:text-2xl">
                  {repo.full_name}
                </span>
                <span className="flex items-center gap-1 rounded border border-[#222222] bg-surface-raised px-2 py-0.5 font-mono text-[11px] text-muted">
                  {repo.private && (
                    <span className="text-[12px] text-muted">
                      <LockIcon />
                    </span>
                  )}
                  {repo.private ? "Private" : "Public"}
                </span>
                <span className="inline-flex items-center gap-1 rounded border border-[#222222] bg-surface-raised px-2 py-0.5 font-mono text-[11px] text-muted">
                  <span className="text-[12px] text-muted">
                    <BranchIcon />
                  </span>
                  {repo.default_branch}
                </span>
              </div>
              {repo.description && (
                <p className="text-sm leading-relaxed text-muted">
                  {renderShortcodes(repo.description)}
                </p>
              )}
            </>
          ) : null}
        </div>
      </section>

      {/* Issue Controls Bar */}
      <section className="flex flex-col items-stretch justify-between gap-3 md:flex-row md:items-center">
        {/* Search Box */}
        <div className="relative flex-1 max-w-md">
          <span className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-2.5 text-muted">
            <span className="text-[16px]">
              <SearchIcon />
            </span>
          </span>
          <input
            ref={searchInputRef}
            id="issue-search-input"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search issues by title or #number..."
            type="text"
            className="h-8 w-full rounded border border-[#222222] bg-[#0c0c0c] pl-8 pr-8 font-sans text-xs text-foreground transition-colors placeholder:text-subtle focus:border-accent focus:outline-none"
            aria-label="Search issues by title or number"
          />
          <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2">
            <kbd className="flex h-4.5 items-center justify-center rounded border border-[#222222] bg-surface-raised px-1.5 font-mono text-[10px] leading-none text-muted">
              /
            </kbd>
          </div>
        </div>

        {/* State Filter Tabs */}
        <div className="flex shrink-0 flex-wrap items-center gap-3">
          <div className="flex items-center rounded border border-[#222222] bg-[#0c0c0c] p-0.5">
            <button
              type="button"
              id="tab-open"
              onClick={() => {
                setStateFilter("open");
                setPage(1);
              }}
              className={`flex cursor-pointer items-center gap-1.5 rounded px-2.5 py-1 font-mono text-xs font-medium transition-colors ${
                stateFilter === "open"
                  ? "bg-surface-raised text-foreground"
                  : "text-muted hover:text-foreground"
              }`}
            >
              <span className="size-1.5 rounded-full bg-[#10b981]" />
              <span>Open</span>
              {stateFilter === "open" && issuesQuery.isSuccess && (
                <span className="ml-0.5 text-muted">{issues.length}</span>
              )}
            </button>
            <button
              type="button"
              id="tab-closed"
              onClick={() => {
                setStateFilter("closed");
                setPage(1);
              }}
              className={`flex cursor-pointer items-center gap-1.5 rounded px-2.5 py-1 font-mono text-xs font-medium transition-colors ${
                stateFilter === "closed"
                  ? "bg-surface-raised text-foreground"
                  : "text-muted hover:text-foreground"
              }`}
            >
              <span className="size-1.5 rounded-full bg-subtle" />
              <span>Closed</span>
              {stateFilter === "closed" && issuesQuery.isSuccess && (
                <span className="ml-0.5 text-subtle">{issues.length}</span>
              )}
            </button>
          </div>
        </div>
      </section>

      {/* Issues Table */}
      <div className="overflow-hidden rounded border border-[#222222] bg-surface">
        {/* Table Header */}
        <div className="grid select-none grid-cols-12 gap-4 border-b border-[#222222] bg-[#0e0e0e] px-4 py-2.5 font-mono text-xs text-muted">
          <div className="col-span-1"># ID</div>
          <div className="col-span-6 md:col-span-7">Title &amp; Context</div>
          <div className="col-span-2 hidden md:block">State</div>
          <div className="col-span-5 text-right md:col-span-2">Updated / Action</div>
        </div>

        {/* Loading State */}
        {issuesQuery.isLoading && (
          <div className="divide-y divide-[#222222]" aria-busy="true" aria-label="Loading issues">
            {[1, 2, 3, 4, 5].map((i) => (
              <div key={i} className="grid animate-pulse grid-cols-12 gap-4 px-4 py-3 items-center">
                <div className="col-span-1">
                  <div className="h-4 w-8 rounded bg-[#222222]" />
                </div>
                <div className="col-span-6 md:col-span-7 flex flex-col gap-1.5">
                  <div className="h-4 w-3/4 rounded bg-[#222222]" />
                  <div className="h-3 w-1/3 rounded bg-[#222222]" />
                </div>
                <div className="col-span-2 hidden md:block">
                  <div className="h-3 w-12 rounded bg-[#222222]" />
                </div>
                <div className="col-span-5 md:col-span-2 flex justify-end">
                  <div className="h-6 w-16 rounded bg-[#222222]" />
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Error State */}
        {issuesQuery.isError && (
          <div role="alert" className="p-8 text-center">
            <p className="text-sm font-medium text-[#e06c75]">
              {issuesQuery.error instanceof ApiException
                ? issuesQuery.error.message
                : "Failed to load issues from GitHub."}
            </p>
            <p className="mt-1 text-xs text-muted">
              Verify your GitHub connection or try refreshing.
            </p>
          </div>
        )}

        {/* Empty State (0 issues for this filter) */}
        {issuesQuery.isSuccess && issues.length === 0 && (
          <div className="p-12 text-center">
            <p className="text-sm font-medium text-foreground">
              No {stateFilter} issues found
            </p>
            <p className="mt-1 text-xs text-muted">
              There are no {stateFilter} issues in this repository.
            </p>
          </div>
        )}

        {/* No search results State */}
        {issuesQuery.isSuccess && issues.length > 0 && visibleIssues.length === 0 && (
          <div className="p-12 text-center">
            <p className="text-sm font-medium text-foreground">
              No issues match “{search.trim()}”
            </p>
            <p className="mt-1 text-xs text-muted">
              Try searching by another title or issue number, or clear your search.
            </p>
            <div className="mt-4">
              <button
                type="button"
                onClick={() => setSearch("")}
                className="inline-flex cursor-pointer items-center gap-1.5 rounded border border-[#333333] bg-[#222222] px-3.5 py-1.5 font-mono text-xs font-medium text-[#ededed] transition-colors hover:bg-[#2a2a2a]"
              >
                Clear search
              </button>
            </div>
          </div>
        )}

        {/* Issues List Rows */}
        {visibleIssues.length > 0 && (
          <div className="divide-y divide-[#222222]">
            {visibleIssues.map((issue) => {
              const num = issue.Number ?? issue.number ?? 0;
              const title = issue.Title ?? issue.title ?? "";
              const state = (issue.State ?? issue.state ?? "open").toLowerCase();
              const author = issue.Author ?? issue.author ?? "";
              const labels = issue.Labels ?? issue.labels ?? [];
              const updatedAt = issue.UpdatedAt ?? issue.updated_at ?? "";
              const htmlUrl = issue.HTMLURL ?? issue.html_url ?? "";

              return (
                <div
                  key={num}
                  className="group grid grid-cols-12 items-center gap-4 px-4 py-3 transition-colors hover:bg-surface-raised"
                >
                  {/* Issue # ID */}
                  <div className="col-span-1">
                    <span className="font-mono text-xs font-medium text-accent">
                      #{num}
                    </span>
                  </div>

                  {/* Title & Context */}
                  <div className="col-span-6 min-w-0 pr-2 md:col-span-7">
                    <div className="flex flex-wrap items-center gap-2">
                      {htmlUrl ? (
                        <a
                          href={htmlUrl}
                          target="_blank"
                          rel="noreferrer"
                          className="truncate text-sm font-medium text-foreground transition-colors hover:text-accent"
                          title={title}
                        >
                          {title}
                        </a>
                      ) : (
                        <span className="truncate text-sm font-medium text-foreground">
                          {title}
                        </span>
                      )}
                      {labels.length > 0 && (
                        <div className="flex shrink-0 items-center gap-1.5">
                          {labels.slice(0, 3).map((lbl) => (
                            <span
                              key={lbl}
                              className={`rounded px-1.5 py-0.5 font-mono text-[11px] ${getIssueLabelClass(
                                lbl
                              )}`}
                            >
                              {lbl}
                            </span>
                          ))}
                          {labels.length > 3 && (
                            <span className="rounded border border-[#222222] bg-surface-raised px-1.5 py-0.5 font-mono text-[11px] text-muted">
                              +{labels.length - 3}
                            </span>
                          )}
                        </div>
                      )}
                    </div>
                    <div className="mt-0.5 flex items-center gap-2 font-mono text-[11px] text-subtle">
                      <span>Updated {formatRelativeTime(updatedAt)}</span>
                      {author && (
                        <>
                          <span>·</span>
                          <span className="text-muted">by @{author}</span>
                        </>
                      )}
                    </div>
                  </div>

                  {/* State */}
                  <div className="col-span-2 hidden items-center gap-1.5 md:flex">
                    <span
                      className={`size-1.5 rounded-full ${
                        state === "open" ? "bg-[#10b981]" : "bg-subtle"
                      }`}
                    />
                    <span className="font-mono text-xs text-muted capitalize">
                      {state}
                    </span>
                  </div>

                  {/* Action Link to GitHub */}
                  <div className="col-span-5 flex items-center justify-end gap-3 md:col-span-2">
                    {htmlUrl ? (
                      <a
                        href={htmlUrl}
                        target="_blank"
                        rel="noreferrer"
                        className="flex h-7 shrink-0 items-center gap-1 rounded border border-[#222222] bg-surface-raised px-2.5 font-mono text-xs text-muted transition-colors hover:border-subtle hover:text-foreground"
                      >
                        <span className="text-[13px]">
                          <ExternalLinkIcon />
                        </span>
                        <span>GitHub</span>
                      </a>
                    ) : null}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>

      {/* Pagination Controls */}
      {(page > 1 || issues.length === 30) && (
        <div className="flex items-center justify-between border-t border-[#222222] pt-4 font-mono text-xs">
          <button
            type="button"
            disabled={page <= 1}
            onClick={() => setPage((p) => Math.max(1, p - 1))}
            className="flex items-center gap-1.5 rounded border border-[#222222] bg-surface px-3 py-1.5 text-muted transition-colors hover:border-subtle hover:text-foreground disabled:opacity-40 disabled:hover:border-[#222222] disabled:hover:text-muted cursor-pointer"
          >
            Previous
          </button>
          <span className="text-muted">Page {page}</span>
          <button
            type="button"
            disabled={issues.length < 30}
            onClick={() => setPage((p) => p + 1)}
            className="flex items-center gap-1.5 rounded border border-[#222222] bg-surface px-3 py-1.5 text-muted transition-colors hover:border-subtle hover:text-foreground disabled:opacity-40 disabled:hover:border-[#222222] disabled:hover:text-muted cursor-pointer"
          >
            Next
          </button>
        </div>
      )}

      {/* Keyboard Shortcuts Footer */}
      <footer className="mt-2 flex select-none items-center justify-between border-t border-[#222222] pt-3 pb-8 font-mono text-[11px] text-subtle">
        <div className="flex items-center gap-3">
          <span className="flex items-center gap-1.5">
            <kbd className="rounded border border-[#222222] bg-surface-raised px-1 py-0.5 font-mono text-[10px] text-muted">
              /
            </kbd>
            <span>to search</span>
          </span>
        </div>
        <div className="flex items-center gap-2 text-muted">
          <span>Issue Sync: Active</span>
          <span className="size-1.5 rounded-full bg-[#10b981]" />
        </div>
      </footer>
    </div>
  );
}
