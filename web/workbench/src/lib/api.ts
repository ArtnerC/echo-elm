// lib/api.ts — thin fetch client over echo-elm local API
// All paths are relative so they work both in dev (proxy) and embedded (same origin).

export interface Library {
  name: string;
  version: string;
  path: string;
  includes: string[];
}

export interface LibraryListResult {
  libraries: Library[];
  total: number;
}

export interface Diagnostic {
  severity: 'Error' | 'Warning' | 'Info' | 'Trace';
  message: string;
  errorType: string;
  startLine: number;
  startChar: number;
  endLine: number;
  endChar: number;
}

export interface TranslateRequest {
  content?: string;
  path?: string;
  format?: 'xml' | 'json' | 'both';
  annotations?: boolean;
  locators?: boolean;
  signatureLevel?: 'None' | 'Differing' | 'Overloads' | 'All';
}

export interface TranslateResult {
  parsedName: string;
  parsedVersion: string;
  elmXml?: string;
  elmJson?: string;
  hasErrors: boolean;
  diagnostics: Diagnostic[];
}

export interface ParityRun {
  id: string;
  createdAt: string;
  fixtureCount: number;
  deltaCount: number;
}

export interface ParityRunListResult {
  runs: ParityRun[];
  total: number;
}

export interface VersionInfo {
  version: string;
  goVersion: string;
  os: string;
  arch: string;
  cqlVersion: string;
  elmVersion: string;
}

export interface WorkspaceInfo {
  path: string;
  libraryCount: number;
  parityRunCount: number;
}

async function get<T>(path: string): Promise<T> {
  const res = await fetch(path);
  if (!res.ok) throw new Error(`GET ${path} → ${res.status}`);
  return res.json();
}

async function post<T>(path: string, body: unknown): Promise<T> {
  const res = await fetch(path, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body)
  });
  if (!res.ok) {
    const text = await res.text().catch(() => res.statusText);
    throw new Error(`POST ${path} → ${res.status}: ${text}`);
  }
  return res.json();
}

export const api = {
  version: () => get<VersionInfo>('/api/version'),
  workspace: () => get<WorkspaceInfo>('/api/workspace'),
  libraries: () => get<LibraryListResult>('/api/libraries'),
  library: (path: string) => get<{ content: string; name: string; version: string }>(`/api/libraries/${encodeURIComponent(path)}`),
  translate: (req: TranslateRequest) => post<TranslateResult>('/api/translate', req),
  parityRuns: () => get<ParityRunListResult>('/api/parity/runs'),
  parityRun: (id: string) => get<{ run: ParityRun; reportMarkdown?: string }>(`/api/parity/runs/${encodeURIComponent(id)}`),
  parityReportMd: (id: string) => fetch(`/api/parity/runs/${encodeURIComponent(id)}/report.md`).then(r => r.text()),
};
