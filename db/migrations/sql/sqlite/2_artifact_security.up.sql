-- Copyright The MatrixHub Authors. Licensed under Apache-2.0.
-- Existing prototype names are retained to preserve reports and operator history.
CREATE TABLE IF NOT EXISTS mh_prototype_artifact_scans (
    repo VARCHAR(255) NOT NULL, revision VARCHAR(40) NOT NULL,
    status VARCHAR(32) NOT NULL, report TEXT,
    updated_at DATETIME, attempt BIGINT NOT NULL DEFAULT 1, force BOOLEAN DEFAULT FALSE,
    PRIMARY KEY (repo, revision)
);
CREATE TABLE IF NOT EXISTS mh_prototype_scan_attempts (
    repo VARCHAR(255) NOT NULL, revision VARCHAR(40) NOT NULL, attempt BIGINT NOT NULL,
    report TEXT, updated_at DATETIME, PRIMARY KEY (repo, revision, attempt)
);
CREATE TABLE IF NOT EXISTS mh_prototype_scan_audit (
    id INTEGER PRIMARY KEY AUTOINCREMENT, repo VARCHAR(255), revision VARCHAR(40),
    attempt BIGINT, actor TEXT, action TEXT, detail TEXT, at DATETIME
);
CREATE INDEX IF NOT EXISTS idx_mh_prototype_scan_audit_repo ON mh_prototype_scan_audit(repo);
CREATE TABLE IF NOT EXISTS mh_prototype_scan_policies (repo VARCHAR(255) PRIMARY KEY, policy TEXT);
CREATE TABLE IF NOT EXISTS mh_prototype_scan_cache (
    repo VARCHAR(255) NOT NULL, context VARCHAR(64) NOT NULL,
    digest VARCHAR(64) NOT NULL, ruleset VARCHAR(64) NOT NULL, result TEXT,
    PRIMARY KEY (repo, context, digest, ruleset)
);
