# 佐久の家 サービスページ

長野県佐久市の家主居住型民泊「佐久の家」の宿泊者向けサービスページです。

- **宿泊情報**（誰でも閲覧可）: ウェルカムガイド、宿泊料金表、宿泊約款
- **宿泊者コミュニティ**（招待コードが必要）: フィード、スポット、投稿、旅の記録、マイページ、AI スポット

## 構成

| ディレクトリ | 内容 |
|---|---|
| `frontend/` | Next.js (App Router) + Mantine。詳細は [frontend/README.md](frontend/README.md) |
| `backend/` | Go + Gin + Huma の API。DB は MySQL（マイグレーションは Prisma、実行時アクセスは GORM）。詳細は [backend/README.md](backend/README.md) |
| `docs/pages/` | 移植元の HTML プロトタイプ |
| `tools/` | 宿泊者名簿・アンケート用のスクリプト（このサービスとは独立） |
| `docker-compose.yml` | ローカル開発用の MySQL と MinIO |

ブラウザは Next.js にだけ接続します。`/api/*` は Next.js が Go の API へ中継するので、セッション Cookie は同一オリジンで扱われます。

## ローカルでの起動

必要なもの: Docker、Go 1.26 以上、Node.js 24 以上。

```bash
# 1. MySQL と MinIO を起動
docker compose up -d

# 2. バックエンド: 設定、マイグレーション、初期データ、起動（ポート 8080）
cd backend
cp .env.example .env
npm install
npm run setup
go run ./cmd/server

# 3. フロントエンド（別のターミナルで。ポート 3000）
cd frontend
cp .env.example .env
npm install
npm run dev
```

ブラウザで `http://localhost:3000` を開きます。コミュニティに入るための招待コードは `backend/.env` の `INVITE_CODE` です。

## 本番で使う前に

- `INVITE_CODE` を 12 文字以上のランダムな値に変更する。
- AI 機能を使う場合は `OPENROUTER_API_KEY` を設定する。未設定でも他の機能は動作する。
- Next.js の前段にリバースプロキシを置き、フロントエンドで `TRUST_FORWARDED_HEADERS=true` を設定すると、利用者ごとのレート制限が有効になる。置かない場合は全利用者共通の上限で動作する（[backend/README.md](backend/README.md) の「招待コードとレート制限」を参照）。
- `docs/pages/guide.html` 由来のウェルカムガイドには未記入の箇所が残っている（プロジェクターやエアコンの手順など）。

## テスト

```bash
cd backend && REQUIRE_TEST_DB=1 go test ./...   # MySQL と MinIO が起動していること
cd frontend && npx biome check && npm run build
```
