-- Enforce unique name on sync_policies
ALTER TABLE `sync_policies` DROP INDEX `idx_name`;
ALTER TABLE `sync_policies` ADD UNIQUE INDEX `idx_name` (`name`);
