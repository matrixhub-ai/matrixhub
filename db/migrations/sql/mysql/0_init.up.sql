CREATE TABLE IF NOT EXISTS `projects`
(
    `id`          int       NOT NULL AUTO_INCREMENT,
    `name`        varchar(64)        DEFAULT "",
    `type`        tinyint   NOT NULL DEFAULT 0,
    `registry_id` int                DEFAULT NULL,
    `created_at`  timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`  timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `name` (`name`)
) ENGINE = InnoDb DEFAULT CHARSET = utf8mb4;

CREATE TABLE IF NOT EXISTS `users`
(
    `id`         varchar(36)  NOT NULL,
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
    `id`           int         NOT NULL AUTO_INCREMENT,
    `name`         varchar(64) NOT NULL,
    `display_name` varchar(64) NOT NULL,
    `permissions`  text        NOT NULL,
    `scope`        varchar(64) NOT NULL,
    `created_at`   timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`   timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `name` (`name`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE IF NOT EXISTS `members_roles_projects`
(
    `id`          int         NOT NULL AUTO_INCREMENT,
    `member_id`   varchar(64) NOT NULL,
    `member_type` varchar(64) NOT NULL,
    `role_id`     varchar(64) NOT NULL,
    `project_id`  int         NOT NULL,
    `created_at`  timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`  timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    INDEX `project_id_index` (`project_id`),
    UNIQUE KEY `composite_index` (`member_id`, `member_type`, `role_id`, `project_id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

CREATE TABLE IF NOT EXISTS `models`
(
    `id`              bigint       NOT NULL AUTO_INCREMENT COMMENT 'Model ID',
    `name`            varchar(255) NOT NULL COMMENT 'Model name (e.g., Llama-2-7b-hf)',
    `project_id`      int          NOT NULL COMMENT 'Reference to projects.id',
    `size`            bigint       NOT NULL COMMENT 'Model size in Bytes',
    `parameter_count` bigint       NOT NULL COMMENT 'Number of model parameters',
    `readme_content`  longtext     NOT NULL COMMENT 'Model README content',
    `is_featured`     tinyint(1)   NOT NULL DEFAULT 0 COMMENT 'Featured model flag',
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
    `category`   varchar(32) NOT NULL COMMENT 'Category (task/framework)',
    `scope`      varchar(16) NOT NULL COMMENT 'Scope (model/dataset)',
    `created_at` timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uniq_name_category_scope` (`name`, `category`, `scope`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = 'Model labels';

CREATE TABLE IF NOT EXISTS `model_labels`
(
    `model_id`   bigint      NOT NULL COMMENT 'Reference to models.id',
    `label_id`   int         NOT NULL COMMENT 'Reference to labels.id',
    `created_at` timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`model_id`, `label_id`),
    CONSTRAINT `fk_model_labels_model_id` FOREIGN KEY (`model_id`) REFERENCES `models` (`id`) ON DELETE CASCADE,
    CONSTRAINT `fk_model_labels_label_id` FOREIGN KEY (`label_id`) REFERENCES `labels` (`id`) ON DELETE CASCADE
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4 COMMENT = 'Model to labels mapping';

CREATE TABLE IF NOT EXISTS `datasets`
(
    `id`         int         NOT NULL AUTO_INCREMENT,
    `name`       varchar(64) NOT NULL,
    `project_id` int         NOT NULL,
    `created_at` timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `name` (`name`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;

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

CREATE TABLE IF NOT EXISTS `access_tokens`
(
    `id`         int         NOT NULL AUTO_INCREMENT,
    `name`       varchar(64) NOT NULL,
    `secret`     VARCHAR(128),
    `user_id`    CHAR(36)    NOT NULL,
    `created_at` timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at` timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`)
) ENGINE = InnoDB DEFAULT CHARSET = utf8mb4;
