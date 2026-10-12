import { useState, useEffect, useRef, useCallback, type FormEvent } from "react";
import { useParams, Link } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiException, apiFetch } from "../../lib/api";
import { renderShortcodes } from "../../lib/emoji";
import type {
  RepoResponse,
  IssuesResponse,
  Issue,
  ProvidersResponse,
  ProviderConfig,
  JobResponse,
  InvestigationsResponse,
  InvestigationRecord,
} from "../../types";
import {
  AddIcon,
  ArrowLeftIcon,
  ArrowRightIcon,
  BranchIcon,
  ExternalLinkIcon,
  LockIcon,
  SearchIcon,
} from "../../components/icons";
import { useJobPoll } from "../investigations/useJobPoll";

// ─── constants ───────────────────────────────────────────────────────────────

const PER_PAGE = 7;
const SEARCH_DEBOUNCE_MS = 300;

// ─── helpers ─────────────────────────────────────────────────────────────────

export function formatRelativeTime(dateStr?: string): string {
  if (!dateStr) return "";
  try {
    const date = new Date(dateStr);
    const diffMs = Date.now() - date.getTime();
    if (isNaN(diffMs)) return dateStr;
    const s = Math.floor(diffMs / 1000);
    if (s < 60) return "just now";
    const m = Math.floor(s / 60);
    if (m < 60) return `${m}m ago`;
    const h = Math.floor(m / 60);
    if (h < 24) return `${h}h ago`;
    const d = Math.floor(h / 24);
    if (d < 30) return `${d}d ago`;
    return date.toLocaleDateString(undefined, { month: "short", day: "numeric", year: "numeric" });
  } catch {
    return dateStr;
  }
}

// Dual-casing accessors: Go struct has no JSON tags so keys are PascalCase;
// keeping snake_case fallbacks for any future serialisation change.
function issueNum(i: Issue): number    { return i.Number ?? i.number ?? 0; }
function issueTitle(i: Issue): string  { return i.Title ?? i.title ?? ""; }
function issueState(i: Issue): string  { return (i.State ?? i.state ?? "open").toLowerCase(); }
function issueAuthor(i: Issue): string { return i.Author ?? i.author ?? ""; }
function issueLabels(i: Issue): string[] {
  const raw = i.Labels ?? i.labels;
  if (!Array.isArray(raw)) return [];
  return raw.filter((l) => typeof l === "string" && l.length > 0);
}
function issueUpdatedAt(i: Issue): string { return i.UpdatedAt ?? i.updated_at ?? ""; }
function issueHtmlUrl(i: Issue): string   { return i.HTMLURL ?? i.html_url ?? ""; }

// ─── label chip ──────────────────────────────────────────────────────────────

// Neutral consistent style — backend does not expose label colors.
function LabelChip({ label }: { label: string }) {
  return (
    <span className="max-w-[120px] truncate rounded border border-[#222222] bg-surface-raised px-1.5 py-0.5 font-mono text-[11px] text-muted">
      {label}
    </span>
  );
}

// ─── provider / model picker ──────────────────────────────────────────────────

interface PickerProps {
  providers: ProviderConfig[];
  selectedConfigId: string;
  selectedModel: string;
  onConfigChange: (id: string) => void;
  onModelChange: (m: string) => void;
}

export function ProviderPicker({
  providers,
  selectedConfigId,
  selectedModel,
  onConfigChange,
  onModelChange,
}: PickerProps) {
  const cfg = providers.find((p) => p.id === selectedConfigId);
  const eligibleModels = cfg?.models.filter((m) => m.enabled) ?? [];

  return (
    <div className="flex flex-col gap-3">
      <label className="flex flex-col gap-1 text-[12px] text-muted">
        Provider
        <select
          value={selectedConfigId}
          onChange={(e) => {
            onConfigChange(e.target.value);
            onModelChange("");
          }}
          className="h-[34px] rounded border border-[#222222] bg-[#080808] px-2 font-mono text-[12px] text-foreground outline-none focus:border-accent"
        >
          <option value="">Select provider…</option>
          {providers
            .filter((p) => p.enabled)
            .map((p) => (
              <option key={p.id} value={p.id}>
                {p.display_name}
              </option>
            ))}
        </select>
      </label>
      <label className="flex flex-col gap-1 text-[12px] text-muted">
        Model
        <select
          value={selectedModel}
          onChange={(e) => onModelChange(e.target.value)}
          disabled={!cfg}
          className="h-[34px] rounded border border-[#222222] bg-[#080808] px-2 font-mono text-[12px] text-foreground outline-none focus:border-accent disabled:opacity-50"
        >
          <option value="">Select model…</option>
          {eligibleModels.map((m) => (
            <option key={m.id} value={m.id}>
              {m.name}
            </option>
          ))}
        </select>
      </label>
    </div>
  );
}

// ─── job poller ───────────────────────────────────────────────────────────────

/* useJobPoll lives in features/investigations/useJobPoll.ts (imported above). */

// ─── per-issue investigation status ──────────────────────────────────────────

/**
 * GET /api/repos/:id/issues/:number/investigations
 * Returns newest-first, kind='investigate' only, LIMIT 50.
 *
 * Limitation: if a repo has >50 investigate jobs for this issue and the most
 * recent done result is beyond that window, this hook will not find it and
 * will show Launch instead of Brief. This is documented rather than silently
 * hidden. Creating a new investigation in that case is safe.
 */
function useIssueInvestigations(repoId: string, issueNumber: number, enabled: boolean) {
  return useQuery({
    queryKey: ["issue-investigations", repoId, issueNumber],
    queryFn: () =>
      apiFetch<InvestigationsResponse>(
        `/api/repos/${repoId}/issues/${issueNumber}/investigations`
      ),
    enabled: enabled && Boolean(repoId) && issueNumber > 0,
    retry: false,
    staleTime: 30_000,
  });
}

// ─── Launch form (shared) ────────────────────────────────────────────────────

interface LaunchFormProps {
  repoId: string;
  issueNumber: number;
  onLaunched: (jobId: string) => void;
}

/** Provider picker + POST /api/repos/:id/investigate. No new job until submit. */
function LaunchForm({ repoId, issueNumber, onLaunched }: LaunchFormProps) {
  const queryClient = useQueryClient();
  const [configId, setConfigId] = useState("");
  const [model, setModel] = useState("");

  const providersQuery = useQuery({
    queryKey: ["providers"],
    queryFn: () => apiFetch<ProvidersResponse>("/api/providers"),
    retry: false,
  });

  const providers = providersQuery.data?.providers ?? [];
  const hasEligible = providers.some((p) => p.enabled && p.models.some((m) => m.enabled));

  const launchMutation = useMutation({
    mutationFn: () =>
      apiFetch<{ job: { id: string } }>(`/api/repos/${repoId}/investigate`, {
        method: "POST",
        body: JSON.stringify({ provider_config_id: configId, model, issue_number: issueNumber }),
      }),
    onSuccess: (data) => {
      void queryClient.invalidateQueries({ queryKey: ["issue-investigations", repoId, issueNumber] });
      onLaunched(data.job.id);
    },
  });

  const canSubmit = Boolean(configId) && Boolean(model) && !launchMutation.isPending;

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (canSubmit) launchMutation.mutate();
  };

  if (providersQuery.isLoading) {
    return <p className="mt-4 text-[12px] text-muted">Loading providers…</p>;
  }
  if (providersQuery.isError) {
    return (
      <p role="alert" className="mt-4 text-[12px] text-[#e06c75]">
        Failed to load providers. Check your configuration.
      </p>
    );
  }
  if (!hasEligible) {
    return (
      <p role="alert" className="mt-4 text-[12px] text-muted">
        No enabled providers or models are configured. Contact your org admin.
      </p>
    );
  }
  return (
    <form onSubmit={onSubmit} className="mt-4 flex flex-col gap-4">
      <ProviderPicker
        providers={providers}
        selectedConfigId={configId}
        selectedModel={model}
        onConfigChange={setConfigId}
        onModelChange={setModel}
      />
      {launchMutation.isError && (
        <p role="alert" className="text-[12px] text-[#e06c75]">
          {launchMutation.error instanceof ApiException
            ? launchMutation.error.message
            : "Launch failed. Try again."}
        </p>
      )}
      <div className="flex justify-end gap-2">
        <button
          type="submit"
          disabled={!canSubmit}
          className="h-8 rounded bg-[#f0f0f0] px-3 text-[12px] font-medium text-[#080808] transition-opacity hover:opacity-90 disabled:opacity-50"
        >
          {launchMutation.isPending ? "Launching…" : "Launch"}
        </button>
      </div>
    </form>
  );
}

// ─── Per-issue action dialog (on-demand history) ─────────────────────────────

/**
 * Single entry point for an issue's investigation action. History is fetched
 * only when the user opens this dialog — never fanned out per row on page
 * load. If the latest done investigation exists → Brief (opens the existing
 * result, creates nothing). Otherwise → Launch form.
 *
 * Known limitation: history is capped at 50 records server-side. A missing
 * record does not prove the issue was never investigated; launching again in
 * that case is safe.
 */
function IssueActionDialog({
  repoId,
  issue,
  onClose,
  onLaunched,
  onBriefJobId,
}: {
  repoId: string;
  issue: Issue;
  onClose: () => void;
  onLaunched: (jobId: string) => void;
  onBriefJobId: (jobId: string) => void;
}) {
  const num = issueNum(issue);
  const histQuery = useIssueInvestigations(repoId, num, true);
  const latestDone: InvestigationRecord | undefined =
    histQuery.data?.investigations.find((r) => r.status === "done");

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 p-4"
      role="dialog"
      aria-modal="true"
      aria-label={`Investigation for #${num}`}
      onClick={onClose}
    >
      <div
        className="w-full max-w-sm rounded border border-[#222222] bg-[#121212] p-5"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-[14px] font-medium text-foreground">Investigation</h2>
        <p className="mt-0.5 font-mono text-[11px] text-muted">
          #{num} — {issueTitle(issue)}
        </p>

        {histQuery.isLoading && (
          <p className="mt-4 text-[12px] text-muted">Checking for existing investigations…</p>
        )}
        {histQuery.isSuccess && latestDone && (
          <div className="mt-4 flex flex-col gap-4">
            <p className="text-[12px] text-muted">
              This issue already has a completed investigation
              {latestDone.model ? ` (${latestDone.model})` : ""}. Opening the
              brief creates nothing new.
            </p>
            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={onClose}
                className="h-8 rounded border border-[#222222] bg-[#121212] px-3 text-[12px] text-foreground transition-colors hover:border-subtle hover:bg-[#161616]"
              >
                Cancel
              </button>
              <button
                type="button"
                id={`brief-btn-${num}`}
                onClick={() => { onBriefJobId(latestDone.id); onClose(); }}
                className="h-8 rounded border border-[#203a5c] bg-[#162234] px-3 text-[12px] font-medium text-accent transition-colors hover:bg-[#1f304a]"
              >
                Brief
              </button>
            </div>
          </div>
        )}
        {(histQuery.isError || (histQuery.isSuccess && !latestDone)) && (
          <>
            {histQuery.isError && (
              <p role="alert" className="mt-4 text-[12px] text-muted">
                Could not check investigation history — you can still launch a
                new investigation below.
              </p>
            )}
            <LaunchForm
              repoId={repoId}
              issueNumber={num}
              onLaunched={(jobId) => { onLaunched(jobId); onClose(); }}
            />
          </>
        )}
        <div className="mt-4 flex justify-start">
          <button
            type="button"
            onClick={onClose}
            className="text-[12px] text-muted transition-colors hover:text-foreground"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
}

// ─── New Investigation dialog (repo-level) ─────────────────────────────────────

interface NewInvestigationDialogProps {
  repoId: string;
  issues: Issue[];
  onClose: () => void;
  onLaunched: (jobId: string) => void;
}

function NewInvestigationDialog({ repoId, issues, onClose, onLaunched }: NewInvestigationDialogProps) {
  const queryClient = useQueryClient();
  const [selectedIssueNum, setSelectedIssueNum] = useState<number | "">(
    issues.length > 0 ? issueNum(issues[0]) : ""
  );
  const [configId, setConfigId] = useState("");
  const [model, setModel] = useState("");

  const providersQuery = useQuery({
    queryKey: ["providers"],
    queryFn: () => apiFetch<ProvidersResponse>("/api/providers"),
    retry: false,
  });

  const providers = providersQuery.data?.providers ?? [];
  const hasEligible = providers.some((p) => p.enabled && p.models.some((m) => m.enabled));

  const launchMutation = useMutation({
    mutationFn: () =>
      apiFetch<{ job: { id: string } }>(`/api/repos/${repoId}/investigate`, {
        method: "POST",
        body: JSON.stringify({
          provider_config_id: configId,
          model,
          issue_number: selectedIssueNum,
        }),
      }),
    onSuccess: (data) => {
      void queryClient.invalidateQueries({
        queryKey: ["issue-investigations", repoId, selectedIssueNum],
      });
      onLaunched(data.job.id);
      onClose();
    },
  });

  const canSubmit =
    Boolean(selectedIssueNum) && Boolean(configId) && Boolean(model) && !launchMutation.isPending;

  const onSubmit = (e: FormEvent) => {
    e.preventDefault();
    if (canSubmit) launchMutation.mutate();
  };

  return (
    <div
      className="fixed inset-0 z-[60] flex items-center justify-center bg-black/60 p-4"
      role="dialog"
      aria-modal="true"
      aria-label="New Investigation"
      onClick={onClose}
    >
      <div
        className="w-full max-w-sm rounded border border-[#222222] bg-[#121212] p-5"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="text-[14px] font-medium text-foreground">New Investigation</h2>
        <p className="mt-0.5 font-mono text-[11px] text-muted">
          Select an issue and a provider to begin.
        </p>

        {providersQuery.isLoading && (
          <p className="mt-4 text-[12px] text-muted">Loading providers…</p>
        )}
        {providersQuery.isError && (
          <p role="alert" className="mt-4 text-[12px] text-[#e06c75]">
            Failed to load providers.
          </p>
        )}
        {providersQuery.isSuccess && !hasEligible && (
          <p role="alert" className="mt-4 text-[12px] text-muted">
            No enabled providers configured. Contact your org admin.
          </p>
        )}
        {providersQuery.isSuccess && hasEligible && (
          <form onSubmit={onSubmit} className="mt-4 flex flex-col gap-4">
            <label className="flex flex-col gap-1 text-[12px] text-muted">
              Issue
              <select
                value={selectedIssueNum}
                onChange={(e) => setSelectedIssueNum(e.target.value ? Number(e.target.value) : "")}
                className="h-[34px] rounded border border-[#222222] bg-[#080808] px-2 font-mono text-[12px] text-foreground outline-none focus:border-accent"
              >
                <option value="">Select issue…</option>
                {issues.map((iss) => {
                  const n = issueNum(iss);
                  return (
                    <option key={n} value={n}>
                      #{n} — {issueTitle(iss)}
                    </option>
                  );
                })}
              </select>
            </label>
            <ProviderPicker
              providers={providers}
              selectedConfigId={configId}
              selectedModel={model}
              onConfigChange={setConfigId}
              onModelChange={setModel}
            />
            {launchMutation.isError && (
              <p role="alert" className="text-[12px] text-[#e06c75]">
                {launchMutation.error instanceof ApiException
                  ? launchMutation.error.message
                  : "Launch failed."}
              </p>
            )}
            <div className="flex justify-end gap-2">
              <button
                type="button"
                onClick={onClose}
                className="h-8 rounded border border-[#222222] bg-[#121212] px-3 text-[12px] text-foreground transition-colors hover:border-subtle hover:bg-[#161616]"
              >
                Cancel
              </button>
              <button
                type="submit"
                disabled={!canSubmit}
                className="h-8 rounded bg-[#f0f0f0] px-3 text-[12px] font-medium text-[#080808] transition-opacity hover:opacity-90 disabled:opacity-50"
              >
                {launchMutation.isPending ? "Launching…" : "Launch"}
              </button>
            </div>
          </form>
        )}
        {!providersQuery.isSuccess && (
          <div className="mt-4 flex justify-end">
            <button
              type="button"
              onClick={onClose}
              className="h-8 rounded border border-[#222222] bg-[#121212] px-3 text-[12px] text-foreground transition-colors hover:border-subtle"
            >
              Close
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

// ─── Issue row ────────────────────────────────────────────────────────────────

function IssueRow({
  issue,
  onInvestigate,
}: {
  issue: Issue;
  onInvestigate: (issue: Issue) => void;
}) {
  const num    = issueNum(issue);
  const title  = issueTitle(issue);
  const state  = issueState(issue);
  const author = issueAuthor(issue);
  const labels = issueLabels(issue);
  const upd    = issueUpdatedAt(issue);
  const url    = issueHtmlUrl(issue);

  return (
    <div className="group grid grid-cols-12 items-center gap-4 px-4 py-3 transition-colors hover:bg-surface-raised">
      {/* # ID */}
      <div className="col-span-1">
        <span className="font-mono text-xs font-medium text-accent">#{num}</span>
      </div>

      {/* Title & Context */}
      <div className="col-span-6 min-w-0 pr-2 md:col-span-7">
        <div className="flex flex-wrap items-center gap-2">
          {url ? (
            <a
              href={url}
              target="_blank"
              rel="noreferrer"
              className="truncate text-sm font-medium text-foreground transition-colors hover:text-accent"
              title={title}
            >
              {title}
            </a>
          ) : (
            <span className="truncate text-sm font-medium text-foreground">{title}</span>
          )}
          {labels.length > 0 && (
            <div className="flex shrink-0 flex-wrap items-center gap-1.5">
              {labels.slice(0, 3).map((lbl) => (
                <LabelChip key={lbl} label={lbl} />
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
          <span>Updated {formatRelativeTime(upd)}</span>
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
        <span className="font-mono text-xs capitalize text-muted">{state}</span>
      </div>

      {/* Actions */}
      <div className="col-span-3 flex items-center justify-end gap-2 md:col-span-2">
        {/* GitHub link — always present when URL is available */}
        {url && (
          <a
            href={url}
            target="_blank"
            rel="noreferrer"
            title="Open on GitHub"
            className="flex h-7 shrink-0 items-center gap-1 rounded border border-[#222222] bg-surface-raised px-2.5 font-mono text-xs text-muted transition-colors hover:border-subtle hover:text-foreground"
          >
            <span className="text-[13px]">
              <ExternalLinkIcon />
            </span>
            <span>GitHub</span>
          </a>
        )}

        {/* Investigation action: exactly one per issue. History is checked
            on demand inside the dialog — never fanned out per row. */}
        <button
          type="button"
          id={`investigate-btn-${num}`}
          onClick={() => onInvestigate(issue)}
          title="Check history, view brief, or launch an investigation"
          className="flex h-7 shrink-0 items-center gap-1 rounded border border-[#222222] bg-surface-raised px-2.5 font-mono text-xs text-foreground transition-colors hover:border-accent hover:text-accent"
        >
          <span>Investigate</span>
        </button>
      </div>
    </div>
  );
}

// ─── Job status banner ────────────────────────────────────────────────────────

function JobStatusBanner({ jobId, onDismiss }: { jobId: string; onDismiss: () => void }) {
  const jobQuery = useJobPoll(jobId);
  const job = jobQuery.data?.job;
  if (!job) return null;

  if (job.status === "queued" || job.status === "running") {
    return (
      <div
        role="status"
        className="flex items-center gap-4 rounded border border-[#222222] bg-surface-raised px-4 py-2.5 text-xs text-muted"
      >
        <span className="inline-block h-2 w-2 animate-pulse rounded-full bg-[#10b981]" />
        Investigation {job.status}… (job {job.id.slice(0, 8)})
      </div>
    );
  }
  if (job.status === "done") {
    return (
      <div className="flex items-center justify-between gap-4 rounded border border-[#1f3a2c] bg-[#15231c] px-4 py-2.5 text-xs text-[#4edea3]">
        <span>Investigation complete. Open Investigate on the issue row to view the brief.</span>
        <button type="button" onClick={onDismiss} className="text-muted hover:text-foreground">
          Dismiss
        </button>
      </div>
    );
  }
  if (job.status === "failed") {
    return (
      <div className="flex items-center justify-between gap-4 rounded border border-[#3c2222] bg-[#1f1616] px-4 py-2.5 text-xs text-[#e06c75]">
        <span>Investigation failed: {job.error ?? "unknown error"}</span>
        <button type="button" onClick={onDismiss} className="text-muted hover:text-foreground">
          Dismiss
        </button>
      </div>
    );
  }
  return null;
}

// ─── Brief modal ──────────────────────────────────────────────────────────────

/**
 * Fetches GET /api/jobs/:id and renders the job result string.
 * This is how Brief opens the existing investigation result — no new job is created.
 */
function BriefModal({ jobId, onClose }: { jobId: string; onClose: () => void }) {
  const jobQuery = useQuery({
    queryKey: ["job", jobId],
    queryFn: () => apiFetch<JobResponse>(`/api/jobs/${jobId}`),
    retry: false,
  });

  return (
    <div
      className="fixed inset-0 z-[60] flex items-start justify-center overflow-y-auto bg-black/70 p-4 pt-12"
      role="dialog"
      aria-modal="true"
      aria-label="Investigation Brief"
      onClick={onClose}
    >
      <div
        className="w-full max-w-2xl rounded border border-[#222222] bg-[#121212] p-6"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-[14px] font-medium text-foreground">Investigation Brief</h2>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close brief"
            className="text-muted transition-colors hover:text-foreground"
          >
            ✕
          </button>
        </div>

        {jobQuery.isLoading && (
          <p className="text-[12px] text-muted">Loading brief…</p>
        )}
        {jobQuery.isError && (
          <p role="alert" className="text-[12px] text-[#e06c75]">
            {jobQuery.error instanceof ApiException
              ? jobQuery.error.message
              : "Failed to load brief."}
          </p>
        )}
        {jobQuery.isSuccess && (
          <>
            <div className="mb-3 flex flex-wrap items-center gap-3 font-mono text-[11px] text-muted">
              <span>Job: {jobQuery.data.job.id.slice(0, 8)}…</span>
              <span>·</span>
              <span
                className={
                  jobQuery.data.job.status === "done"
                    ? "text-[#4edea3]"
                    : jobQuery.data.job.status === "failed"
                    ? "text-[#e06c75]"
                    : "text-muted"
                }
              >
                {jobQuery.data.job.status}
              </span>
              <span>·</span>
              <span>{jobQuery.data.job.model}</span>
            </div>
            {jobQuery.data.job.result ? (
              <pre className="max-h-[60vh] overflow-y-auto whitespace-pre-wrap rounded border border-[#1a1a1a] bg-[#0c0c0c] p-4 font-mono text-[12px] leading-relaxed text-foreground">
                {jobQuery.data.job.result}
              </pre>
            ) : jobQuery.data.job.status === "failed" ? (
              <p className="text-[12px] text-[#e06c75]">
                Investigation failed: {jobQuery.data.job.error ?? "unknown error"}
              </p>
            ) : (
              <p className="text-[12px] text-muted">
                No result yet ({jobQuery.data.job.status}).
              </p>
            )}
            <div className="mt-4 flex justify-end">
              <Link
                to={`/investigations/${jobId}`}
                className="font-mono text-[11px] text-muted transition-colors hover:text-foreground"
              >
                Open full investigation page →
              </Link>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

// ─── Main component ───────────────────────────────────────────────────────────

export default function RepositoryIssues() {
  const { id } = useParams<{ id: string }>();

  const [stateFilter, setStateFilter] = useState<"open" | "closed">("open");
  const [page, setPage] = useState(1);
  const [search, setSearch] = useState("");
  const searchInputRef = useRef<HTMLInputElement>(null);

  const [launchIssue, setLaunchIssue] = useState<Issue | null>(null);
  const [newInvOpen, setNewInvOpen] = useState(false);
  const [activePollJobId, setActivePollJobId] = useState<string | null>(null);
  const [briefJobId, setBriefJobId] = useState<string | null>(null);

  // '/' keyboard shortcut to focus search.
  useEffect(() => {
    const handler = (e: KeyboardEvent) => {
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
    window.addEventListener("keydown", handler);
    return () => window.removeEventListener("keydown", handler);
  }, []);

  const repoQuery = useQuery({
    queryKey: ["repo", id],
    queryFn: () => apiFetch<RepoResponse>(`/api/repos/${id}`),
    enabled: Boolean(id),
    retry: false,
  });

  // Repository-wide server search, debounced so typing does not fire a
  // request per keystroke. The trimmed query is the source of truth for
  // fetching; a blank query omits `q` and returns the recent listing.
  const [debouncedSearch, setDebouncedSearch] = useState("");
  useEffect(() => {
    const timer = setTimeout(() => setDebouncedSearch(search), SEARCH_DEBOUNCE_MS);
    return () => clearTimeout(timer);
  }, [search]);
  const activeQuery = debouncedSearch.trim();

  // React Query key includes id, state, page, and normalized query — cached
  // data cannot leak across repositories, states, pages, or searches.
  const issuesQuery = useQuery({
    queryKey: ["repo-issues", id, stateFilter, page, activeQuery],
    queryFn: () => {
      const params = new URLSearchParams({
        state: stateFilter,
        page: String(page),
        per_page: String(PER_PAGE),
      });
      if (activeQuery) params.set("q", activeQuery);
      return apiFetch<IssuesResponse>(`/api/repos/${id}/issues?${params.toString()}`);
    },
    enabled: Boolean(id),
    retry: false,
  });

  const repo    = repoQuery.data?.repository;
  const issues  = issuesQuery.data?.issues ?? [];
  // Next is driven solely by the envelope's has_more flag — never by page
  // fullness, and no totals are fabricated.
  const hasMore = issuesQuery.data?.has_more ?? false;
  const handleLaunch    = useCallback((issue: Issue) => setLaunchIssue(issue), []);
  const handleLaunched  = useCallback((jobId: string) => setActivePollJobId(jobId), []);
  const handleBriefJobId = useCallback((jobId: string) => setBriefJobId(jobId), []);

  // ── repo error ────────────────────────────────────────────────────────────

  if (repoQuery.isError) {
    const err = repoQuery.error;
    return (
      <div className="mx-auto flex w-full max-w-[1280px] flex-col gap-6 px-6 py-6">
        <Link
          to="/repositories"
          className="inline-flex items-center gap-1.5 font-mono text-xs text-muted transition-colors hover:text-foreground"
        >
          <ArrowLeftIcon />
          <span>Repositories</span>
        </Link>
        <div role="alert" className="rounded border border-[#222222] bg-[#161616] p-8 text-center">
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
              Return to Repositories
            </Link>
          </div>
        </div>
      </div>
    );
  }

  // ── main render ───────────────────────────────────────────────────────────

  return (
    <div className="mx-auto flex w-full max-w-[1280px] flex-col gap-6 px-6 py-6">
      {/* Back link */}
      <div>
        <Link
          to="/repositories"
          className="inline-flex items-center gap-1.5 font-mono text-xs text-muted transition-colors hover:text-foreground"
        >
          <ArrowLeftIcon />
          <span>Repositories</span>
        </Link>
      </div>

      {/* Repository identity row */}
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

        {/* New Investigation (repo-level) */}
        <div className="flex shrink-0 items-center self-start">
          <button
            type="button"
            id="new-investigation-btn"
            onClick={() => setNewInvOpen(true)}
            disabled={!issuesQuery.isSuccess || issues.length === 0}
            className="flex h-8 cursor-pointer items-center gap-1.5 rounded bg-[#f0f0f0] px-3 font-sans text-[12px] font-medium text-[#080808] transition-opacity hover:opacity-90 disabled:cursor-not-allowed disabled:opacity-40"
          >
            <span className="text-[15px]">
              <AddIcon />
            </span>
            <span>New Investigation</span>
          </button>
        </div>
      </section>

      {/* Job status banner */}
      {activePollJobId && (
        <JobStatusBanner
          jobId={activePollJobId}
          onDismiss={() => setActivePollJobId(null)}
        />
      )}

      {/* Controls bar */}
      <section className="flex flex-col items-stretch justify-between gap-3 md:flex-row md:items-center">
        {/* Search — repository-wide via the backend `q` parameter */}
        <div className="relative max-w-md flex-1">
          <span className="pointer-events-none absolute inset-y-0 left-0 flex items-center pl-2.5 text-muted">
            <span className="text-[16px]">
              <SearchIcon />
            </span>
          </span>
          <input
            ref={searchInputRef}
            id="issue-search-input"
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(1); }}
            placeholder="Search all repository issues…"
            type="text"
            className="h-8 w-full rounded border border-[#222222] bg-[#0c0c0c] pl-8 pr-8 font-sans text-xs text-foreground transition-colors placeholder:text-subtle focus:border-accent focus:outline-none"
            aria-label="Search all repository issues by text or issue number"
          />
          <div className="pointer-events-none absolute inset-y-0 right-0 flex items-center pr-2">
            <kbd className="flex items-center justify-center rounded border border-[#222222] bg-surface-raised px-1.5 font-mono text-[10px] leading-none text-muted">
              /
            </kbd>
          </div>
        </div>

        {/* State filter tabs — each triggers a new server request */}
        <div className="flex shrink-0 flex-wrap items-center gap-3">
          <div className="flex items-center rounded border border-[#222222] bg-[#0c0c0c] p-0.5">
            <button
              type="button"
              id="tab-open"
              onClick={() => { setStateFilter("open"); setPage(1); setSearch(""); }}
              className={`flex cursor-pointer items-center gap-1.5 rounded px-2.5 py-1 font-mono text-xs font-medium transition-colors ${
                stateFilter === "open"
                  ? "bg-surface-raised text-foreground"
                  : "text-muted hover:text-foreground"
              }`}
            >
              <span className="size-1.5 rounded-full bg-[#10b981]" />
              <span>Open</span>
            </button>
            <button
              type="button"
              id="tab-closed"
              onClick={() => { setStateFilter("closed"); setPage(1); setSearch(""); }}
              className={`flex cursor-pointer items-center gap-1.5 rounded px-2.5 py-1 font-mono text-xs font-medium transition-colors ${
                stateFilter === "closed"
                  ? "bg-surface-raised text-foreground"
                  : "text-muted hover:text-foreground"
              }`}
            >
              <span className="size-1.5 rounded-full bg-subtle" />
              <span>Closed</span>
            </button>
          </div>
        </div>
      </section>

      {/* Issue table */}
      <div className="overflow-hidden rounded border border-[#222222] bg-surface">
        {/* Table header */}
        <div className="grid select-none grid-cols-12 gap-4 border-b border-[#222222] bg-[#0e0e0e] px-4 py-2.5 font-mono text-xs text-muted">
          <div className="col-span-1"># ID</div>
          <div className="col-span-6 md:col-span-7">Title &amp; Context</div>
          <div className="col-span-2 hidden md:block">State</div>
          <div className="col-span-3 text-right md:col-span-2">Actions</div>
        </div>

        {/* Loading */}
        {issuesQuery.isLoading && (
          <div className="divide-y divide-[#222222]" aria-busy="true" aria-label="Loading issues">
            {Array.from({ length: 5 }).map((_, i) => (
              <div key={i} className="grid animate-pulse grid-cols-12 items-center gap-4 px-4 py-3">
                <div className="col-span-1"><div className="h-4 w-8 rounded bg-[#222222]" /></div>
                <div className="col-span-6 flex flex-col gap-1.5 md:col-span-7">
                  <div className="h-4 w-3/4 rounded bg-[#222222]" />
                  <div className="h-3 w-1/3 rounded bg-[#222222]" />
                </div>
                <div className="col-span-2 hidden md:block"><div className="h-3 w-12 rounded bg-[#222222]" /></div>
                <div className="col-span-3 flex justify-end gap-2 md:col-span-2">
                  <div className="h-7 w-16 rounded bg-[#222222]" />
                  <div className="h-7 w-14 rounded bg-[#222222]" />
                </div>
              </div>
            ))}
          </div>
        )}

        {/* Error */}
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
            <div className="mt-4">
              <button
                type="button"
                onClick={() => void issuesQuery.refetch()}
                className="inline-flex items-center gap-1.5 rounded border border-[#333333] bg-[#222222] px-3.5 py-1.5 font-mono text-xs font-medium text-[#ededed] transition-colors hover:bg-[#2a2a2a]"
              >
                Retry
              </button>
            </div>
          </div>
        )}

        {/* Empty listing — the server returned no issues for this state
            and page. has_more=false here is a genuine end. */}
        {issuesQuery.isSuccess && issues.length === 0 && !activeQuery && (
          <div className="p-12 text-center">
            <p className="text-sm font-medium text-foreground">
              No {stateFilter} issues on this page
            </p>
            <p className="mt-1 text-xs text-muted">
              There are no further {stateFilter} issues in this repository.
            </p>
          </div>
        )}

        {/* No repository-wide matches for the search query */}
        {issuesQuery.isSuccess && issues.length === 0 && activeQuery && (
          <div className="p-12 text-center">
            <p className="text-sm font-medium text-foreground">
              No issues match "{activeQuery}"
            </p>
            <p className="mt-1 text-xs text-muted">
              No {stateFilter} issues in this repository match your search.
              Try a different term or clear the search to return to recent issues.
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

        {/* Issue rows — rendered exactly as returned by the server */}
        {issues.length > 0 && (
          <div className="divide-y divide-[#222222]">
            {issues.map((issue) => (
              <IssueRow
                key={issueNum(issue)}
                issue={issue}
                onInvestigate={handleLaunch}
              />
            ))}
          </div>
        )}
      </div>

      {/* Pagination — compact centered group. Next follows the envelope's
          has_more flag; Previous is disabled on page 1. No totals shown. */}
      <div className="mt-4 flex flex-wrap items-center justify-center gap-3 font-mono text-xs">
        <button
          type="button"
          id="pagination-prev"
          disabled={page <= 1 || issuesQuery.isLoading}
          onClick={() => setPage((p) => Math.max(1, p - 1))}
          className="flex h-8 cursor-pointer items-center gap-1.5 rounded border border-[#222222] bg-surface px-3 text-muted transition-colors hover:border-subtle hover:text-foreground disabled:cursor-default disabled:opacity-40 disabled:hover:border-[#222222] disabled:hover:text-muted"
        >
          <span className="text-[13px]">
            <ArrowLeftIcon />
          </span>
          <span>Previous</span>
        </button>
        <span className="min-w-16 text-center text-muted">Page {page}</span>
        <button
          type="button"
          id="pagination-next"
          disabled={!hasMore || issuesQuery.isLoading}
          onClick={() => setPage((p) => p + 1)}
          className="flex h-8 cursor-pointer items-center gap-1.5 rounded border border-[#222222] bg-surface px-3 text-muted transition-colors hover:border-accent hover:text-accent disabled:cursor-default disabled:opacity-40 disabled:hover:border-[#222222] disabled:hover:text-muted"
        >
          <span>Next</span>
          <span className="text-[13px]">
            <ArrowRightIcon />
          </span>
        </button>
      </div>

      {/* Footer */}
      <footer className="mt-2 flex select-none items-center justify-between border-t border-[#222222] pb-8 pt-3 font-mono text-[11px] text-subtle">
        <div className="flex items-center gap-3">
          <span className="flex items-center gap-1.5">
            <kbd className="rounded border border-[#222222] bg-surface-raised px-1 py-0.5 font-mono text-[10px] text-muted">
              /
            </kbd>
            <span>to search all issues</span>
          </span>
        </div>
        <div className="flex items-center gap-2 text-muted">
          <span>Issue Sync: Active</span>
          <span className="size-1.5 rounded-full bg-[#10b981]" />
        </div>
      </footer>

      {/* Dialogs */}
      {launchIssue && id && (
        <IssueActionDialog
          repoId={id}
          issue={launchIssue}
          onClose={() => setLaunchIssue(null)}
          onLaunched={handleLaunched}
          onBriefJobId={handleBriefJobId}
        />
      )}
      {newInvOpen && id && (
        <NewInvestigationDialog
          repoId={id}
          issues={issues}
          onClose={() => setNewInvOpen(false)}
          onLaunched={handleLaunched}
        />
      )}
      {briefJobId && (
        <BriefModal jobId={briefJobId} onClose={() => setBriefJobId(null)} />
      )}
    </div>
  );
}
