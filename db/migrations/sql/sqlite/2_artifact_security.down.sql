-- Copyright The MatrixHub Authors. Licensed under Apache-2.0.
-- Explicit rollback deletes scan evidence. Back up these tables before downgrading.
DROP TABLE IF EXISTS mh_prototype_scan_cache;
DROP TABLE IF EXISTS mh_prototype_scan_policies;
DROP TABLE IF EXISTS mh_prototype_scan_audit;
DROP TABLE IF EXISTS mh_prototype_scan_attempts;
DROP TABLE IF EXISTS mh_prototype_artifact_scans;
