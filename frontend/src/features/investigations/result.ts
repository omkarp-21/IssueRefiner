// Structured investigation result: mirrors internal/investigation/model.go.
// job.result arrives as a JSON *string*; parse with parseResultString() —
// never trust the raw payload shape in production rendering.

export type Readiness = "READY" | "NEEDS_INFORMATION";
export type Confidence = "HIGH" | "MEDIUM" | "LOW";

export interface ResultHypothesis {
  statement: string;
  confidence: Confidence;
  reasoning: string;
  evidence: string[];
}

export interface InvestigationResult {
  summary: string;
  readiness: Readiness;
  missing_information: string[];
  signals: string[];
  observed_behavior: string;
  expected_behavior: string;
  environment: string;
  reproduction_steps: string[];
  hypotheses: ResultHypothesis[];
}

export type ParsedResult =
  | { ok: true; result: InvestigationResult }
  | { ok: false; reason: string };

function asStringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value.filter((v): v is string => typeof v === "string");
}

function asString(value: unknown): string {
  return typeof value === "string" ? value : "";
}

function isConfidence(value: unknown): value is Confidence {
  return value === "HIGH" || value === "MEDIUM" || value === "LOW";
}

/**
 * Safely parse a job.result JSON string, applying the same structural rules
 * as the backend validator: summary required, readiness enum, hypotheses
 * fully populated, NEEDS_INFORMATION requires missing_information. Returns a
 * controlled failure instead of throwing.
 */
export function parseResultString(raw: string | null | undefined): ParsedResult {
  if (raw == null || raw.trim() === "") {
    return { ok: false, reason: "empty" };
  }
  let parsed: unknown;
  try {
    parsed = JSON.parse(raw);
  } catch {
    return { ok: false, reason: "malformed" };
  }
  if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
    return { ok: false, reason: "malformed" };
  }
  const o = parsed as Record<string, unknown>;
  const summary = asString(o.summary).trim();
  if (!summary) return { ok: false, reason: "missing-summary" };
  const readiness = o.readiness;
  if (readiness !== "READY" && readiness !== "NEEDS_INFORMATION") {
    return { ok: false, reason: "bad-readiness" };
  }
  const missing = asStringArray(o.missing_information);
  if (readiness === "NEEDS_INFORMATION" && missing.length === 0) {
    return { ok: false, reason: "needs-info-without-items" };
  }
  const rawHyps = Array.isArray(o.hypotheses) ? o.hypotheses : [];
  const hypotheses: ResultHypothesis[] = [];
  for (const h of rawHyps) {
    if (typeof h !== "object" || h === null) return { ok: false, reason: "bad-hypothesis" };
    const ho = h as Record<string, unknown>;
    if (!asString(ho.statement).trim()) return { ok: false, reason: "bad-hypothesis" };
    if (!isConfidence(ho.confidence)) return { ok: false, reason: "bad-hypothesis" };
    if (!asString(ho.reasoning).trim()) return { ok: false, reason: "bad-hypothesis" };
    const evidence = asStringArray(ho.evidence);
    if (evidence.length === 0) return { ok: false, reason: "bad-hypothesis" };
    hypotheses.push({
      statement: asString(ho.statement),
      confidence: ho.confidence,
      reasoning: asString(ho.reasoning),
      evidence,
    });
  }
  return {
    ok: true,
    result: {
      summary,
      readiness,
      missing_information: missing,
      signals: asStringArray(o.signals),
      observed_behavior: asString(o.observed_behavior),
      expected_behavior: asString(o.expected_behavior),
      environment: asString(o.environment),
      reproduction_steps: asStringArray(o.reproduction_steps),
      hypotheses,
    },
  };
}

/** Split an evidence ref into a displayable source label. No links invented. */
export function evidenceLabel(ref: string): { kind: "issue" | "comment" | "unknown"; id: string } {
  if (ref === "issue") return { kind: "issue", id: "issue" };
  const m = /^comment:(\d+)$/.exec(ref);
  if (m) return { kind: "comment", id: m[1] };
  return { kind: "unknown", id: ref };
}
