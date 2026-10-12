-- 佐久の家 宿泊者名簿（MySQL 8.0）
CREATE TABLE IF NOT EXISTS guests (
  id                  BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  -- 「送信日時|氏名」。n8n の UPSERT キー
  row_key             VARCHAR(255) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  submitted_at        DATETIME     NOT NULL COMMENT 'フォーム送信日時（JST）',
  check_in            DATE         NOT NULL,
  check_out           DATE         NOT NULL,
  check_in_time       TIME         NULL COMMENT 'チェックイン予定時刻',
  is_representative   TINYINT(1)   NOT NULL DEFAULT 0,
  representative_name VARCHAR(100) NOT NULL,
  full_name           VARCHAR(100) NOT NULL,
  name_kana           VARCHAR(100) NULL COMMENT 'フリガナ（日本在住者のみ）',
  address             TEXT         NULL COMMENT '「宿泊代表者と同じ」は展開済み。未解決は「要確認」',
  phone               VARCHAR(30)  NULL,
  email               VARCHAR(255) NULL,
  has_jp_address      TINYINT(1)   NOT NULL,
  nationality         VARCHAR(100) NULL,
  passport_no         VARCHAR(30)  NULL,
  created_at          DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at          DATETIME     NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_guests_row_key (row_key),
  KEY idx_guests_check_out (check_out)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci;
