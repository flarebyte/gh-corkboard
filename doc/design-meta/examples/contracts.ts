type Status = "draft" | "active" | "review" | "resolved" | "obsolete";
type Format = "json" | "csv" | "md";

interface Index {
  version: 1;
  documents: IndexEntry[];
}

interface IndexEntry {
  path: string;
  description: string;
  status: Status;
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
  path: string;
  format: Format;
  schemaId?: string;
  state: "valid" | "invalid" | "schemaless";
  diagnostics: Diagnostic[];
}
