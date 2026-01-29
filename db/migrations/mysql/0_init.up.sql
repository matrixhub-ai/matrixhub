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