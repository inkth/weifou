-- 一次性清理已退役的打赏、付费真人咨询和音视频通话数据库结构。
-- 执行前请先备份生产数据库；GORM AutoMigrate 不会主动删除旧表或旧列。

BEGIN;

DROP TABLE IF EXISTS profit_shares;
DROP TABLE IF EXISTS consult_sessions;
DROP TABLE IF EXISTS consult_slots;
DROP TABLE IF EXISTS consult_settings;
DROP TABLE IF EXISTS refunds;

ALTER TABLE IF EXISTS orders
  DROP COLUMN IF EXISTS profile_id,
  DROP COLUMN IF EXISTS payee_user_id,
  DROP COLUMN IF EXISTS duration_min,
  DROP COLUMN IF EXISTS slot_id,
  DROP COLUMN IF EXISTS scheduled_at,
  DROP COLUMN IF EXISTS message,
  DROP COLUMN IF EXISTS source;

COMMIT;
