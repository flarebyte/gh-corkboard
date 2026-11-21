type Status = "draft" | "active" | "review" | "resolved" | "obsolete";
type Format = "json" | "csv" | "md";
type PrincipalId = `${"human" | "agent" | "skill"}:${string}`;
type Scope = `${"agent" | "skill" | "topic"}:${string}`;
type Permission = "read" | "write";
type DocumentId = string; // Canonical lowercase UUIDv4

interface AccessPolicy {
  version: 1;
  principals: Principal[];
  topics: string[];
  grants: ScopeGrant[];
}

interface Principal {
  id: PrincipalId;
  role: "member" | "topic-admin" | "admin";
  state: "active" | "disabled";
}

interface ScopeGrant {
  scope: Scope;
  principal: PrincipalId;
  permission: Permission;
}

interface DocumentMetadata {
  id: DocumentId;
  scope: Scope;
  name: string;
  format: Format;
  schemaId?: string;
  description: string;
  status: Status;
}

interface Index {
  version: 1;
  documents: IndexEntry[];
}

interface IndexEntry extends DocumentMetadata {
  path: string;
  sha256: string;
}

interface CsvProjection {
  columns: string[];
  rows: Record<string, string>[];
}

interface MarkdownProjection {
  headings: { level: 1 | 2 | 3 | 4 | 5 | 6; text: string }[];
}

interface Diagnostic {
  code: string;
  message: string;
  instancePath?: string; // JSON Pointer into the original JSON or projection
  line?: number; // One-based coordinates when available
  column?: number;
}

interface ValidationResult {
  id: DocumentId;
  format: Format;
  schemaId?: string;
  state: "valid" | "invalid" | "schemaless";
  diagnostics: Diagnostic[];
}
