ALTER TABLE `sync_jobs`
    ADD COLUMN `job_type` varchar(32) NOT NULL DEFAULT 'sync' COMMENT 'Job type: sync or model_cache'
        AFTER `id`,
    ADD COLUMN `params` text NULL COMMENT 'Parameters of the job type, as JSON'
        AFTER `sync_type`;
