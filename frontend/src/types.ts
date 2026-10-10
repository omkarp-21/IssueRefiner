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

export interface ApiError {
  error: string;
}
