export interface RepoSummary {
  id: string;
  full_name: string;
  open_findings: number;
  high_severity: number;
  last_scanned_at: string;
}

export interface TrendPoint {
  date: string;
  high: number;
  medium: number;
  low: number;
}

export interface Finding {
  id: string;
  file: string;
  line: number;
  algorithm: string;
  severity: string;
  category: string;
  suggested_replacement: string;
}