CREATE TABLE IF NOT EXISTS `users`
(
    `id`           CHAR(36)    NOT NULL,
    `username`     varchar(64) NOT NULL,
    `created_at`   timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`   timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `username` (`username`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;


CREATE TABLE `roles`
(
    `name`        varchar(64) NOT NULL,
    `description` varchar(255)         DEFAULT NULL,
    `type`        varchar(64) NOT NULL DEFAULT '0',
    `gproduct`    varchar(64)          DEFAULT NULL,
    `permissions` text        NOT NULL,
    `auth_scope`  varchar(64) NOT NULL,
    `created_at`  timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`  timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`name`) USING BTREE
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE IF NOT EXISTS `projects`
(
    `id`           int         NOT NULL AUTO_INCREMENT,
    `name`         varchar(64) DEFAULT "",
    `created_at`   timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`   timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `name` (`name`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE IF NOT EXISTS `members_roles_projects` (
  `id`            int NOT NULL AUTO_INCREMENT,
  `member_name`   varchar(64) NOT NULL,
  `member_type`   varchar(64) NOT NULL,
  `role_name`     varchar(64) NOT NULL,
  `project_id`     int NOT NULL,
  `created_at`    timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  `updated_at`    timestamp NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`),
  INDEX `project_id_index` (`project_id`),
  UNIQUE KEY `composite_index` (`member_name`,`member_type`,`role_name`, `project_id`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE IF NOT EXISTS `models`
(
    `id`           int         NOT NULL AUTO_INCREMENT,
    `name`         varchar(64) NOT NULL,
    `project_id`     int NOT NULL,
    `created_at`   timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`   timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `name` (`name`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE IF NOT EXISTS `datasets`
(
    `id`           int         NOT NULL AUTO_INCREMENT,
    `name`         varchar(64) NOT NULL,
    `project_id`     int NOT NULL,
    `created_at`   timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`   timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `name` (`name`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;

CREATE TABLE IF NOT EXISTS `ssh_keys`
(
    `id`           int         NOT NULL AUTO_INCREMENT,
    `name`         varchar(64) NOT NULL,
    `public_key`   text,
    `user_id`      CHAR(36)     NOT NULL,
    `created_at`   timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `updated_at`   timestamp   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    KEY `name` (`name`)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4;