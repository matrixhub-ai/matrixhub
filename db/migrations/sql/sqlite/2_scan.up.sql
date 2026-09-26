-- Model artifact security scanning (issue #1066): task lifecycle, per-file
-- findings with digest-based reuse, admission policy and audit trail.

CREATE TABLE IF NOT EXISTS scan_tasks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_type TEXT NOT NULL,
    project TEXT NOT NULL,
    name TEXT NOT NULL,
    revision TEXT NOT NULL,
    trigger TEXT NOT NULL DEFAULT 'upload',
    status TEXT NOT NULL DEFAULT 'pending',
    verdict TEXT NOT NULL DEFAULT '',
    force INTEGER NOT NULL DEFAULT 0,
    created_by TEXT NOT NULL DEFAULT '',
    error TEXT,
    file_count INTEGER NOT NULL DEFAULT 0,
    bytes_total INTEGER NOT NULL DEFAULT 0,
    scanner_versions TEXT,
    started_at DATETIME,
    finished_at DATETIME,
    claim_owner TEXT NOT NULL DEFAULT '',
    claim_deadline DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_scan_repo_rev ON scan_tasks (repo_type, project, name, revision, id);
CREATE INDEX IF NOT EXISTS idx_scan_claim ON scan_tasks (status, claim_deadline);

CREATE TABLE IF NOT EXISTS scan_file_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id INTEGER NOT NULL,
    path TEXT NOT NULL,
    digest TEXT NOT NULL,
    file_type TEXT NOT NULL DEFAULT '',
    size INTEGER NOT NULL DEFAULT 0,
    scanner_id TEXT NOT NULL,
    scanner_version TEXT NOT NULL DEFAULT '',
    severity TEXT NOT NULL DEFAULT 'clean',
    rule TEXT NOT NULL DEFAULT '',
    detail TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_scan_file_task ON scan_file_results (task_id);
CREATE INDEX IF NOT EXISTS idx_scan_file_cache ON scan_file_results (digest, scanner_id, scanner_version);

CREATE TABLE IF NOT EXISTS scan_policies (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    project TEXT DEFAULT NULL,
    mode TEXT NOT NULL DEFAULT 'enforce',
    block_severity TEXT NOT NULL DEFAULT 'critical',
    on_pending TEXT NOT NULL DEFAULT 'block',
    on_failed TEXT NOT NULL DEFAULT 'block',
    on_scanner_unavailable TEXT NOT NULL DEFAULT 'block',
    updated_by TEXT NOT NULL DEFAULT '',
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS uni_scan_policy_project ON scan_policies (project);

CREATE TABLE IF NOT EXISTS scan_audit_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_type TEXT NOT NULL DEFAULT '',
    project TEXT NOT NULL DEFAULT '',
    name TEXT NOT NULL DEFAULT '',
    revision TEXT NOT NULL DEFAULT '',
    path TEXT,
    actor TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL,
    decision TEXT NOT NULL DEFAULT '',
    reason TEXT NOT NULL DEFAULT '',
    task_id INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_scan_audit_repo ON scan_audit_events (repo_type, project, name, revision, id);
CREATE INDEX IF NOT EXISTS idx_scan_audit_actor ON scan_audit_events (actor, id);
