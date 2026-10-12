export interface User {
  id: string;
  github_id: number;
  username: string;
  display_name: string;
  email: string | null;
}

export interface Organization {
  id: string;
  name: string;
}

export interface MeResponse {
  user: User;
  organization: Organization;
  role: "owner" | "member";
}

export interface Repository {
  id: string;
  organization_id: string;
  github_id: number;
  owner: string;
  name: string;
  full_name: string;
  description: string | null;
  default_branch: string;
  html_url: string;
  private: boolean;
  created_at: string;
  updated_at: string;
  synced_at: string;
}

export interface ReposResponse {
  repositories: Repository[];
}

export interface RepoResponse {
  repository: Repository;
}

export interface Issue {
  Number?: number;
  number?: number;
  Title?: string;
  title?: string;
  Body?: string;
  body?: string;
  State?: string;
  state?: string;
  Author?: string;
  author?: string;
  Labels?: string[];
  labels?: string[];
  CreatedAt?: string;
  created_at?: string;
  UpdatedAt?: string;
  updated_at?: string;
  HTMLURL?: string;
  html_url?: string;
}

export interface IssuesResponse {
  issues: Issue[];
  page: number;
  per_page: number;
  has_more: boolean;
}

export interface ApiError {
  error: string;
}

// Job model (internal/jobs/model.go). result is a JSON string when present.
export interface Job {
  id: string;
  organization_id: string;
  repository_id: string;
  provider_config_id: string;
  provider_type: string;
  model: string;
  created_by: string;
  kind: "investigate" | "refine";
  issue_number: number;
  status: "queued" | "running" | "done" | "failed";
  input_title: string | null;
  input_body: string | null;
  result: string | null;
  error: string | null;
  input_tokens: number | null;
  output_tokens: number | null;
  attempts: number;
  created_at: string;
  updated_at: string;
}

export interface JobResponse {
  job: Job;
  repository: Repository | null;
}

// Provider model returned by GET /api/providers.
export interface ProviderModel {
  id: string;
  name: string;
  enabled: boolean;
}

export interface ProviderConfig {
  id: string;
  provider_type: string;
  display_name: string;
  enabled: boolean;
  models: ProviderModel[];
}

export interface ProvidersResponse {
  providers: ProviderConfig[];
}

// Investigation history item from GET /api/repos/:id/issues/:number/investigations.
// The backend returns kind='investigate' only, newest-first, LIMIT 50.
export interface InvestigationRecord {
  id: string;           // job id
  status: "queued" | "running" | "done" | "failed";
  model: string;
  created_at: string;
  updated_at: string;
}

export interface InvestigationsResponse {
  investigations: InvestigationRecord[];
}
