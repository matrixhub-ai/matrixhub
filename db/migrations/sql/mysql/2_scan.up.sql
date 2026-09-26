-- Model artifact security scanning (issue #1066): task lifecycle, per-file
-- findings with digest-based reuse, admission policy and audit trail.

CREATE TABLE IF NOT EXISTS `scan_tasks` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `repo_type` VARCHAR(16) NOT NULL,
    `project` VARCHAR(255) NOT NULL,
    `name` VARCHAR(255) NOT NULL,
    `revision` VARCHAR(64) NOT NULL,
    `trigger` VARCHAR(32) NOT NULL DEFAULT 'upload',
    `status` VARCHAR(16) NOT NULL DEFAULT 'pending',
    `verdict` VARCHAR(16) NOT NULL DEFAULT '',
    `force` TINYINT NOT NULL DEFAULT 0,
    `created_by` VARCHAR(255) NOT NULL DEFAULT '',
    `error` TEXT,
    `file_count` INT NOT NULL DEFAULT 0,
    `bytes_total` BIGINT NOT NULL DEFAULT 0,
    `scanner_versions` JSON,
    `started_at` TIMESTAMP NULL DEFAULT NULL,
    `finished_at` TIMESTAMP NULL DEFAULT NULL,
    `claim_owner` VARCHAR(64) NOT NULL DEFAULT '',
    `claim_deadline` TIMESTAMP NULL DEFAULT NULL,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_scan_repo_rev` (`repo_type`,`project`,`name`,`revision`,`id`),
    KEY `idx_scan_claim` (`status`,`claim_deadline`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `scan_file_results` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `task_id` BIGINT NOT NULL,
    `path` VARCHAR(1024) NOT NULL,
    `digest` CHAR(64) NOT NULL,
    `file_type` VARCHAR(64) NOT NULL DEFAULT '',
    `size` BIGINT NOT NULL DEFAULT 0,
    `scanner_id` VARCHAR(32) NOT NULL,
    `scanner_version` VARCHAR(128) NOT NULL DEFAULT '',
    `severity` VARCHAR(16) NOT NULL DEFAULT 'clean',
    `rule` VARCHAR(255) NOT NULL DEFAULT '',
    `detail` VARCHAR(1024) NOT NULL DEFAULT '',
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_scan_file_task` (`task_id`),
    KEY `idx_scan_file_cache` (`digest`,`scanner_id`,`scanner_version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `scan_policies` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `project` VARCHAR(255) DEFAULT NULL,
    `mode` VARCHAR(16) NOT NULL DEFAULT 'enforce',
    `block_severity` VARCHAR(16) NOT NULL DEFAULT 'critical',
    `on_pending` VARCHAR(8) NOT NULL DEFAULT 'block',
    `on_failed` VARCHAR(8) NOT NULL DEFAULT 'block',
    `on_scanner_unavailable` VARCHAR(8) NOT NULL DEFAULT 'block',
    `updated_by` VARCHAR(255) NOT NULL DEFAULT '',
    `updated_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uni_scan_policy_project` (`project`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `scan_audit_events` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `repo_type` VARCHAR(16) NOT NULL DEFAULT '',
    `project` VARCHAR(255) NOT NULL DEFAULT '',
    `name` VARCHAR(255) NOT NULL DEFAULT '',
    `revision` VARCHAR(64) NOT NULL DEFAULT '',
    `path` VARCHAR(1024) DEFAULT NULL,
    `actor` VARCHAR(255) NOT NULL DEFAULT '',
    `action` VARCHAR(32) NOT NULL,
    `decision` VARCHAR(64) NOT NULL DEFAULT '',
    `reason` VARCHAR(1024) NOT NULL DEFAULT '',
    `task_id` BIGINT NOT NULL DEFAULT 0,
    `created_at` TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_scan_audit_repo` (`repo_type`,`project`,`name`,`revision`,`id`),
    KEY `idx_scan_audit_actor` (`actor`,`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
