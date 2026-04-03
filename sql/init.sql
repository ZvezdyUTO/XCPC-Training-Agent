CREATE TABLE users
(
    id        VARCHAR(32)  NOT NULL,
    name      VARCHAR(255) NOT NULL,
    password  VARCHAR(255) NOT NULL,
    status    TINYINT      NOT NULL,
    is_system TINYINT      NOT NULL,

    cf_handle VARCHAR(64) NULL,
    ac_handle VARCHAR(64) NULL,

    create_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    update_at TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    delete_at TIMESTAMP NULL DEFAULT NULL,

    PRIMARY KEY (id),
    UNIQUE KEY uk_cf_handle (cf_handle),
    UNIQUE KEY uk_ac_handle (ac_handle)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE daily_training_stats
(
    student_id       VARCHAR(32) NOT NULL,
    stat_date        DATE        NOT NULL,

    -- Codeforces 当日新增
    cf_new_total     INT         NOT NULL DEFAULT 0,
    cf_new_undefined INT         NOT NULL DEFAULT 0,
    cf_new_800       INT         NOT NULL DEFAULT 0,
    cf_new_900       INT         NOT NULL DEFAULT 0,
    cf_new_1000      INT         NOT NULL DEFAULT 0,
    cf_new_1100      INT         NOT NULL DEFAULT 0,
    cf_new_1200      INT         NOT NULL DEFAULT 0,
    cf_new_1300      INT         NOT NULL DEFAULT 0,
    cf_new_1400      INT         NOT NULL DEFAULT 0,
    cf_new_1500      INT         NOT NULL DEFAULT 0,
    cf_new_1600      INT         NOT NULL DEFAULT 0,
    cf_new_1700      INT         NOT NULL DEFAULT 0,
    cf_new_1800      INT         NOT NULL DEFAULT 0,
    cf_new_1900      INT         NOT NULL DEFAULT 0,
    cf_new_2000      INT         NOT NULL DEFAULT 0,
    cf_new_2100      INT         NOT NULL DEFAULT 0,
    cf_new_2200      INT         NOT NULL DEFAULT 0,
    cf_new_2300      INT         NOT NULL DEFAULT 0,
    cf_new_2400      INT         NOT NULL DEFAULT 0,
    cf_new_2500      INT         NOT NULL DEFAULT 0,
    cf_new_2600      INT         NOT NULL DEFAULT 0,
    cf_new_2700      INT         NOT NULL DEFAULT 0,
    cf_new_2800_plus INT         NOT NULL DEFAULT 0,

    -- AtCoder 当日新增
    ac_new_total     INT         NOT NULL DEFAULT 0,
    ac_new_undefined INT         NOT NULL DEFAULT 0,
    ac_new_0_399     INT         NOT NULL DEFAULT 0,
    ac_new_400_799   INT         NOT NULL DEFAULT 0,
    ac_new_800_1199  INT         NOT NULL DEFAULT 0,
    ac_new_1200_1599 INT         NOT NULL DEFAULT 0,
    ac_new_1600_1999 INT         NOT NULL DEFAULT 0,
    ac_new_2000_2399 INT         NOT NULL DEFAULT 0,
    ac_new_2400_2799 INT         NOT NULL DEFAULT 0,
    ac_new_2800_plus INT         NOT NULL DEFAULT 0,

    created_at       TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at       TIMESTAMP NULL DEFAULT NULL,

    PRIMARY KEY (student_id, stat_date),

    CONSTRAINT fk_daily_user
        FOREIGN KEY (student_id)
            REFERENCES users (id)
            ON DELETE CASCADE,

    INDEX            idx_stat_date (stat_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE contest_records
(
    student_id    VARCHAR(32)  NOT NULL,
    platform      ENUM('CF','AC') NOT NULL,

    contest_id    VARCHAR(64)  NOT NULL,
    contest_name  VARCHAR(255) NOT NULL,
    contest_date  DATETIME     NOT NULL,

    contest_rank  INT          NOT NULL,
    old_rating    INT          NOT NULL,
    new_rating    INT          NOT NULL,
    rating_change INT          NOT NULL,

    performance   INT NULL,

    created_at    TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at    TIMESTAMP NULL DEFAULT NULL,

    PRIMARY KEY (student_id, platform, contest_id),

    CONSTRAINT fk_contest_user
        FOREIGN KEY (student_id)
            REFERENCES users (id)
            ON DELETE CASCADE,

    INDEX         idx_contest_date (contest_date),
    INDEX         idx_platform (platform)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE student_sync_state
(
    student_id              VARCHAR(32) NOT NULL,
    is_fully_initialized    TINYINT     NOT NULL DEFAULT 0,
    latest_successful_date  DATE        NULL,

    created_at              TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at              TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (student_id),

    CONSTRAINT fk_sync_state_user
        FOREIGN KEY (student_id)
            REFERENCES users (id)
            ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE training_alerts
(
    id            BIGINT       NOT NULL AUTO_INCREMENT,
    student_id    VARCHAR(32)  NOT NULL,
    alert_date    DATE         NOT NULL,
    alert_type    VARCHAR(32)  NOT NULL,
    severity      ENUM('low', 'medium', 'high') NOT NULL,
    status        ENUM('new', 'ack', 'resolved') NOT NULL DEFAULT 'new',
    title         VARCHAR(255) NOT NULL,
    evidence_json JSON         NOT NULL,
    actions_json  JSON         NOT NULL,
    created_at    TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at    TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (id),

    CONSTRAINT fk_alert_user
        FOREIGN KEY (student_id)
            REFERENCES users (id)
            ON DELETE CASCADE,

    UNIQUE KEY uk_alert_unique (student_id, alert_date, alert_type),
    INDEX idx_alert_date (alert_date),
    INDEX idx_alert_status (status),
    INDEX idx_alert_severity (severity),
    INDEX idx_alert_student_date (student_id, alert_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE anomaly_rule_config
(
    id          TINYINT      NOT NULL,
    config_json JSON         NOT NULL,
    created_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  TIMESTAMP    NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 初始化异常规则配置（id 固定为 1）。
-- 关键约束：difficulty_drop_current_window_days <=
--          difficulty_drop_medium_days_threshold <=
--          difficulty_drop_high_days_threshold。
INSERT INTO anomaly_rule_config (id, config_json)
VALUES (
    1,
    JSON_OBJECT(
        'current_window_days', 7,
        'baseline_window_days', 30,
        'baseline_min_daily', 1.0,
        'current_min_daily_for_alert', 2.0,
        'volume_recovery_ratio_1d', 0.80,
        'drop_low_threshold', 0.35,
        'drop_medium_threshold', 0.50,
        'drop_high_threshold', 0.70,
        'inactive_days_threshold', 3,
        'inactive_days_medium_threshold', 5,
        'inactive_days_high_threshold', 7,
        'inactive_baseline_min_daily', 1.0,
        'difficulty_drop_current_window_days', 3,
        'difficulty_drop_medium_days_threshold', 5,
        'difficulty_drop_high_days_threshold', 7,
        'difficulty_drop_baseline_window_days', 30,
        'difficulty_drop_min_current_total', 1,
        'difficulty_drop_min_baseline_high_ratio', 0.15,
        'difficulty_level_round_base', 100,
        'difficulty_relative_high_delta', 200,
        'difficulty_relative_easy_delta', 200,
        'difficulty_drop_low_threshold', 0.35,
        'difficulty_drop_medium_threshold', 0.50,
        'difficulty_drop_high_threshold', 0.70
    )
)
ON DUPLICATE KEY UPDATE
    config_json = VALUES(config_json),
    updated_at = CURRENT_TIMESTAMP;
