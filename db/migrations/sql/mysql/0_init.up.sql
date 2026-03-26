CREATE TABLE IF NOT EXISTS `registries`
(
    `id`              int          NOT NULL AUTO_INCREMENT,
    `name`            varchar(64)  NOT NULL,
    `description`     text         NOT NULL,
    `type`            varchar(64)  NOT NULL,              -- registry provider (huggingface, etc.)
    `url`             varchar(255) NOT NULL,
    `credential_type` varchar(255)          DEFAULT NULL, -- ('basic', 'oauth', 'secret')
    `auth_info`       text,
    `insecure`        tinyint(1)   NOT NULL DEFAULT '0',  -- skip SSL verification (0 for False, 1 for True)
    `status`          int          NOT NULL,              -- status (healthy, unhealthy, unknown)
    `created_at`      timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`      timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `name` (`name`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE IF NOT EXISTS `projects`
(
    `id`           int       NOT NULL AUTO_INCREMENT,
    `name`         varchar(64)        DEFAULT "",
    `type`         tinyint   NOT NULL DEFAULT 0,
    `registry_id`  int                DEFAULT NULL,
    `organization` varchar(64)        DEFAULT "",
    `created_at`   timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`   timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `name` (`name`),
    CONSTRAINT `fk_projects_registry_id` FOREIGN KEY (`registry_id`) REFERENCES `registries` (`id`)
) ENGINE = InnoDb DEFAULT CHARSET = utf8mb4;

CREATE TABLE IF NOT EXISTS `users`
(
    `id`         bigint       NOT NULL AUTO_INCREMENT,
    `username`   varchar(255) NOT NULL,
    `password`   varchar(255) NOT NULL default "",
    `email`      varchar(255) NOT NULL default "",
    `created_at` timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `username` (`username`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE IF NOT EXISTS `roles`
(
    `id`          int         NOT NULL AUTO_INCREMENT,
    `name`        varchar(64) NOT NULL,
    `permissions` text        NOT NULL,
    `scope`       varchar(64) NOT NULL,
    `created_at`  timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`  timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `name` (`name`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE IF NOT EXISTS `members_roles_projects`
(
    `id`          int         NOT NULL AUTO_INCREMENT,
    `member_id`   varchar(64) NOT NULL,
    `member_type` varchar(64) NOT NULL,
    `role_id`     int                  DEFAULT NULL,
    `project_id`  int                  DEFAULT NULL,
    `created_at`  timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`  timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    INDEX `project_id_index` (`project_id`),
    UNIQUE KEY `composite_index` (`member_id`, `member_type`, `role_id`, `project_id`),
    CONSTRAINT `fk_members_roles_projects_project_id` FOREIGN KEY (`project_id`) REFERENCES `projects` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_members_roles_projects_role_id` FOREIGN KEY (`role_id`) REFERENCES `roles` (`id`) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE IF NOT EXISTS `models`
(
    `id`              bigint       NOT NULL AUTO_INCREMENT COMMENT 'Model ID',
    `name`            varchar(255) NOT NULL COMMENT 'Model name (e.g., Llama-2-7b-hf)',
    `project_id`      int          NOT NULL COMMENT 'Reference to projects.id',
    `size`            bigint       NOT NULL COMMENT 'Model size in Bytes',
    `default_branch`  varchar(255) NOT NULL COMMENT 'Model branch (e.g., main)',
    `parameter_count` bigint       NOT NULL COMMENT 'Number of model parameters',
    `readme_content`  longtext     NOT NULL COMMENT 'Model README content',
    `is_popular`      tinyint(1)   NOT NULL DEFAULT 0 COMMENT 'popular model flag',
    `created_at`      timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`      timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uniq_project_name` (`project_id`, `name`),
    KEY `idx_updated_at` (`updated_at`),
    CONSTRAINT `fk_models_project_id` FOREIGN KEY (`project_id`) REFERENCES `projects` (`id`) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = 'AI models storage';

CREATE TABLE IF NOT EXISTS `labels`
(
    `id`         int         NOT NULL AUTO_INCREMENT COMMENT 'Label ID',
    `name`       varchar(64) NOT NULL COMMENT 'Label name',
    `category`   varchar(32) NOT NULL COMMENT 'Category (task/library/other)',
    `scope`      varchar(16) NOT NULL COMMENT 'Scope (model/dataset)',
    `created_at` timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uniq_name_category_scope` (`name`, `category`, `scope`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = 'Model labels';

CREATE TABLE IF NOT EXISTS `models_labels`
(
    `model_id`   bigint    NOT NULL COMMENT 'Reference to models.id',
    `label_id`   int       NOT NULL COMMENT 'Reference to labels.id',
    `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`model_id`, `label_id`),
    CONSTRAINT `fk_model_labels_model_id` FOREIGN KEY (`model_id`) REFERENCES `models` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_model_labels_label_id` FOREIGN KEY (`label_id`) REFERENCES `labels` (`id`) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = 'Model to labels mapping';

CREATE TABLE IF NOT EXISTS `datasets`
(
    `id`             bigint       NOT NULL AUTO_INCREMENT COMMENT 'Dataset ID',
    `name`           varchar(255) NOT NULL COMMENT 'Dataset name',
    `project_id`     int          NOT NULL COMMENT 'Reference to projects.id',
    `default_branch` varchar(64)  NOT NULL DEFAULT 'main' COMMENT 'Default branch name',
    `is_popular`     tinyint(1)   NOT NULL DEFAULT 0 COMMENT 'popular dataset flag',
    `num_rows`       varchar(64)  NOT NULL DEFAULT '' COMMENT 'Number of rows in dataset',
    `size`           bigint       NOT NULL DEFAULT 0 COMMENT 'Dataset size in Bytes',
    `readme_content` longtext     NOT NULL COMMENT 'Dataset README content',
    `created_at`     timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`     timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uniq_project_name` (`project_id`, `name`),
    KEY `idx_updated_at` (`updated_at`),
    CONSTRAINT `fk_datasets_project_id` FOREIGN KEY (`project_id`) REFERENCES `projects` (`id`) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = 'Datasets storage';

CREATE TABLE IF NOT EXISTS `datasets_labels`
(
    `dataset_id` bigint    NOT NULL COMMENT 'Reference to datasets.id',
    `label_id`   int       NOT NULL COMMENT 'Reference to labels.id',
    `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`dataset_id`, `label_id`),
    CONSTRAINT `fk_dataset_labels_dataset_id` FOREIGN KEY (`dataset_id`) REFERENCES `datasets` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_dataset_labels_label_id` FOREIGN KEY (`label_id`) REFERENCES `labels` (`id`) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = 'Dataset to labels mapping';

CREATE TABLE `access_tokens` (
     `id` int NOT NULL AUTO_INCREMENT,
     `name` varchar(64) NOT NULL,
     `user_id` bigint NOT NULL,
     `token_hash` varchar(128) NOT NULL DEFAULT '',
     `enabled` tinyint(1) NOT NULL DEFAULT '0',
     `expire_at` timestamp NULL DEFAULT NULL,
     `created_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
     `updated_at` timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
     PRIMARY KEY (`id`),
     UNIQUE KEY `uniq_token_hash` (`token_hash`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `sync_policies`
(
    `id`                  int       NOT NULL AUTO_INCREMENT,
    `name`                varchar(255) NOT NULL,
    `description`         text,
    `policy_type`         int       NOT NULL DEFAULT 1, -- 1: pull, 2: push
    `trigger_type`        int       NOT NULL DEFAULT 1, -- 1: manual, 2: scheduled
    `source_registry_id`  int,
    `resource_name`       varchar(255),
    `resource_types`      varchar(255),                 -- comma separated: model,dataset
    `target_resource_name` varchar(255),
    `target_project_name`  varchar(255),
    `bandwidth`           varchar(64),
    `is_overwrite`        tinyint(1) NOT NULL DEFAULT 0,
    `is_disabled`         tinyint(1) NOT NULL DEFAULT 0,
    `created_at`          timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`          timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_name` (`name`),
    KEY `idx_source_registry_id` (`source_registry_id`),
    CONSTRAINT `fk_sync_policies_registry_id`
        FOREIGN KEY (`source_registry_id`) REFERENCES `registries` (`id`) ON DELETE SET NULL
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE IF NOT EXISTS `sync_tasks`
(
    `id`                  int       NOT NULL AUTO_INCREMENT,
    `sync_policy_id`      int       NOT NULL,
    `trigger_type`        int       NOT NULL DEFAULT 1, -- 1: manual, 2: scheduled
    `status`              int       NOT NULL DEFAULT 1, -- 1: running, 2: succeeded, 3: failed, 4: stopped
    `started_timestamp`   bigint    DEFAULT 0,
    `completed_timestamp` bigint    DEFAULT 0,
    `total_items`         int       DEFAULT 0,
    `successful_items`    int       DEFAULT 0,
    `complete_percents`   int       DEFAULT 0,
    `created_at`          timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`          timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `idx_sync_policy_id` (`sync_policy_id`),
    KEY `idx_status` (`status`),
    CONSTRAINT `fk_sync_tasks_policy_id`
        FOREIGN KEY (`sync_policy_id`) REFERENCES `sync_policies` (`id`) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE IF NOT EXISTS `sync_jobs`
(
    `id`                   int          NOT NULL AUTO_INCREMENT,
    `remote_registry_id`   int          NOT NULL,
    `remote_project_name`  varchar(255) NOT NULL,
    `remote_resource_name` varchar(255) NOT NULL,
    `project_name`         varchar(255),
    `resource_name`        varchar(255) NOT NULL,
    `resource_type`        varchar(64)  NOT NULL,
    `sync_type`            varchar(64)  NOT NULL,
    `sync_task_id`  int,
    `complete_percents`    int,
    `created_at`           timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`           timestamp    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

INSERT INTO `roles` (`name`, `permissions`, `scope`)
VALUES ('platform_admin', '*', 'platform'),
       ('project_admin', '*', 'project'),
       ('project_editor', 'read,write', 'project'),
       ('project_viewer', 'read', 'project');

CREATE TABLE IF NOT EXISTS sessions (
    token CHAR(43) COLLATE utf8mb4_bin PRIMARY KEY,
    data BLOB NOT NULL,
    expiry TIMESTAMP(6) NOT NULL,
    INDEX sessions_expiry_idx (expiry)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
