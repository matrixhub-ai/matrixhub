-- Copyright The MatrixHub Authors. Licensed under Apache-2.0.
-- Existing prototype names are retained to preserve reports and operator history.
CREATE TABLE IF NOT EXISTS mh_prototype_artifact_scans (
    repo VARCHAR(255) NOT NULL, revision VARCHAR(40) NOT NULL,
    status VARCHAR(32) NOT NULL, report LONGTEXT,
    updated_at DATETIME(3), attempt BIGINT NOT NULL DEFAULT 1, force BOOLEAN DEFAULT FALSE,
    PRIMARY KEY (repo, revision)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
CREATE TABLE IF NOT EXISTS mh_prototype_scan_attempts (
    repo VARCHAR(255) NOT NULL, revision VARCHAR(40) NOT NULL, attempt BIGINT NOT NULL,
    report LONGTEXT, updated_at DATETIME(3), PRIMARY KEY (repo, revision, attempt)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
CREATE TABLE IF NOT EXISTS mh_prototype_scan_audit (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT, repo VARCHAR(255), revision VARCHAR(40),
    attempt BIGINT, actor TEXT, action TEXT, detail LONGTEXT, at DATETIME(3),
    INDEX idx_mh_prototype_scan_audit_repo (repo)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
CREATE TABLE IF NOT EXISTS mh_prototype_scan_policies (repo VARCHAR(255) PRIMARY KEY, policy TEXT)
    ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
CREATE TABLE IF NOT EXISTS mh_prototype_scan_cache (
    repo VARCHAR(255) NOT NULL, context VARCHAR(64) NOT NULL,
    digest VARCHAR(64) NOT NULL, ruleset VARCHAR(64) NOT NULL, result LONGTEXT,
    PRIMARY KEY (repo, context, digest, ruleset)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
