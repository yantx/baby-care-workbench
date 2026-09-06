-- ============================================================
-- 爸妈育儿工作台 Phase 1 数据库初始化脚本（PostgreSQL 13+）
-- 说明：应用启动时默认使用 GORM AutoMigrate 自动建表，
--       生产环境请关闭 AutoMigrate，改用本脚本受控迁移：
--       psql -h 127.0.0.1 -U postgres -d baby_care -f 001_init.sql
-- ============================================================

-- 用户表
CREATE TABLE IF NOT EXISTS users (
    id                BIGSERIAL PRIMARY KEY,
    openid            VARCHAR(64)  NOT NULL,
    unionid           VARCHAR(64)  DEFAULT '',
    nickname          VARCHAR(64)  DEFAULT '',
    avatar_url        VARCHAR(512) DEFAULT '',
    active_family_id  BIGINT       DEFAULT 0,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_openid ON users (openid);
CREATE INDEX IF NOT EXISTS idx_users_active_family ON users (active_family_id);

-- 家庭表
CREATE TABLE IF NOT EXISTS families (
    id           BIGSERIAL PRIMARY KEY,
    name         VARCHAR(64) NOT NULL,
    invite_code  VARCHAR(8)  NOT NULL,
    owner_id     BIGINT      NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_families_invite_code ON families (invite_code);
CREATE INDEX IF NOT EXISTS idx_families_owner ON families (owner_id);

-- 家庭成员表
CREATE TABLE IF NOT EXISTS family_members (
    id         BIGSERIAL PRIMARY KEY,
    family_id  BIGINT      NOT NULL,
    user_id    BIGINT      NOT NULL,
    role       VARCHAR(16) NOT NULL,          -- father/mother/grandparent/other
    nickname   VARCHAR(64) DEFAULT '',
    joined_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_family_user ON family_members (family_id, user_id);

-- 宝宝档案表
CREATE TABLE IF NOT EXISTS babies (
    id         BIGSERIAL PRIMARY KEY,
    family_id  BIGINT      NOT NULL,
    name       VARCHAR(64) NOT NULL,
    gender     INT         DEFAULT 0,        -- 0未知 1男 2女
    birthday   TIMESTAMPTZ,
    avatar_url VARCHAR(512) DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_babies_family ON babies (family_id);

-- 护理记录表（核心表：喂奶/睡眠/尿布/体温/用药 统一存储）
CREATE TABLE IF NOT EXISTS care_records (
    id          BIGSERIAL PRIMARY KEY,
    family_id   BIGINT       NOT NULL,
    baby_id     BIGINT       NOT NULL,
    recorder_id BIGINT       NOT NULL,
    type        VARCHAR(16)  NOT NULL,        -- feeding/sleep/diaper/temperature/medicine
    started_at  TIMESTAMPTZ  NOT NULL,
    ended_at    TIMESTAMPTZ,                  -- 睡眠等有时长的记录
    amount_ml   NUMERIC(8,1),                 -- 奶量(ml)
    content     VARCHAR(128) DEFAULT '',      -- 类型化内容：尿布(wet/dirty/mixed)、喂奶(母乳/奶粉/左/右)、药品名
    temp_value  NUMERIC(4,1),                 -- 体温(℃)
    note        VARCHAR(512) DEFAULT '',
    details     JSONB,                        -- 扩展明细
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_family_started ON care_records (family_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_baby_type ON care_records (baby_id, type);
CREATE INDEX IF NOT EXISTS idx_recorder ON care_records (recorder_id);
