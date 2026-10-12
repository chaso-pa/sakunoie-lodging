# 佐久の家 サービスページ — バックエンド

Go（Gin + Huma v2）の API サーバー。DB は MySQL、マイグレーションは Prisma、Go からの DB アクセスは GORM。
セッションは `alexedwards/scs` のサーバーサイドセッション（MySQL の `sessions` テーブル）。

## 必要なもの

- Go 1.26 以上
- Node.js 24 / npm（Prisma CLI 用。yarn は不要）
- Docker（ローカルの MySQL と MinIO）

## セットアップ

```bash
# リポジトリのルートで MySQL と MinIO を起動
docker compose up -d

cd backend
cp .env.example .env        # 値はローカル用。本番では INVITE_CODE などを必ず変更する
npm install
npx prisma migrate deploy   # マイグレーション適用（スキーマ変更時は migrate dev --name <name>）
go run ./cmd/seed           # サンプルデータ投入（何度実行してもよい）
go run ./cmd/server         # http://localhost:8080
```

`npm run setup` はマイグレーション適用とサンプルデータ投入をまとめて行う。

### 招待コードとレート制限

`INVITE_CODE` は長くランダムな値にする（release モードでは、例の値のまま、または 12 文字未満だと起動時に
警告を出す。起動は止めない）。

招待コード検証（5 回/分）と引き継ぎ（3 回/分、5 回連続失敗で 15 分ロック）のクライアント単位の制限は、
**クライアントのアドレスを転送するリバースプロキシの背後でのみ効く**（フロントエンドで
`TRUST_FORWARDED_HEADERS=true`）。既定の構成では Next.js のプロキシがクライアントのアドレスを転送しないため、
全員が 127.0.0.1 として届く。その場合はクライアント単位の制限をかけると 1 人が全員を締め出せてしまうので、
全員共通の上限（どちらも 30 回/分）に切り替え、連続失敗のロックは適用しない。引き継ぎコードは約 62 ビットの
乱数なのでこの頻度では総当たりできず、招待コードの総当たりは 1 日 43,200 回までに抑えられる。

### 足あとの公開範囲

足あと（`POST /api/gps-visits`）は、`is_public` も `exact_public` も省略時は false。会員が明示的にオンに
しない限り、足あとは他の会員に見えず、公開しても正確な座標は返さない。自分の足あとには常に座標が含まれる。

### 画像ストレージ（MinIO / S3）

バケットの手動作成は不要。サーバーは起動時に `S3_BUCKET` が無ければ作成し、バケットにポリシーが
1 つも無い場合に限り「誰でもオブジェクトを読める（一覧は不可）」ポリシーを付ける。すでにポリシーがある
バケットには触れない。起動時にストレージへ届かなくてもサーバーは起動し、最初のアップロード時に同じ確認を
やり直す。`S3_ENDPOINT` などが空なら画像アップロードだけが 503 になり、他の機能は動く。

アップロードされた画像は Go 側で先頭バイトから種類を判定し（JPEG / PNG / WebP のみ）、デコードして
JPEG に再エンコードしてから保存する。EXIF（GPS を含む）は残らない。投稿写真は幅 1400px、アバターは
512px 四方に収まるよう縮小する。投稿を削除すると、その写真のオブジェクトも削除する（削除に失敗した場合は
ログに残し、オブジェクトだけがバケットに残る）。

### 外部サービス

| 機能 | 環境変数 | 未設定時 |
|------|----------|----------|
| AI 旅行アシスタント | `OPENROUTER_API_KEY`, `OPENROUTER_MODEL` | `POST /api/ai/plan` が 503（外部呼び出しなし） |
| 翻訳 | `TRANSLATION_PROVIDER`（`mymemory` または `openrouter`） | 空・未知の値、または `openrouter` でキーが空なら 503。翻訳結果は `post_translations` にキャッシュする |
| 天気 | なし（open-meteo、キー不要） | — |

タイムアウトは AI 30 秒、翻訳 10 秒、天気 5 秒。外部サービスの失敗は 502 で、応答にもログにも
外部 URL・API キー・外部の応答本文は出さない。

### ログ

`slog` の JSON を標準出力に出す。リクエストごとに 1 行（`request_id`、メソッド、ルートのテンプレート、
ステータス、所要時間、`member_id`）。`X-Request-Id` レスポンスヘッダで同じ ID を返す。URL のクエリ文字列、
ヘッダ、Cookie、リクエスト本文は記録しない（検索語・座標・セッショントークンを含むため）。

## テスト

```bash
go test ./...
```

名前が `Integration` で終わるテストは MySQL を使う（S3 のテストは MinIO も使う。既定は
`http://127.0.0.1:9000`、`TEST_S3_ENDPOINT` などで変更可。テストごとに専用バケットを作って消す）。既定の接続先は docker compose の MySQL
（`root:root@tcp(127.0.0.1:3306)`）で、`TEST_MYSQL_DSN` で変更できる。テストは
`sakunoie_test_<パッケージ名>` という専用データベースを作り直し、`prisma/migrations` の SQL を適用して使う
（開発用の `sakunoie` データベースには触れない）。

MySQL に接続できない場合、これらのテストは失敗ではなくスキップになる。スキップを失敗として扱いたいとき
（CI など）は `REQUIRE_TEST_DB=1 go test ./...` とする（MinIO に届かない場合も失敗になる）。
AI・翻訳・天気のテストはスタブの HTTP サーバーを使い、実サービスには接続しない。

## 構成

```
cmd/server/        サーバー起動（設定読み込み、DB・セッション初期化）
cmd/seed/          サンプルデータ投入
internal/server/   Gin エンジンの組み立て（ミドルウェア、2 つの Huma API、ルート登録）
internal/routes/   ルート登録（Setup<Resource>Routes(publicAPI, authAPI, deps)）
internal/handlers/ Huma ハンドラ
internal/services/ ビジネスロジック
internal/models/   GORM 構造体と Huma の入出力構造体
internal/middlewares/ 認証、レート制限、ボディサイズ上限、リクエスト ID とアクセスログ
internal/session/  scs セッションマネージャ
internal/storage/  画像アップロードのインターフェースと S3 実装
internal/logging/  slog の設定とリクエスト ID の受け渡し
prisma/            スキーマとマイグレーション
```

API は `/api` 配下。公開 API のドキュメントは `/api/docs`、認証必須 API のドキュメントは
`/api/docs/auth`（ログインが必要）。規約は `CLAUDE.md` を参照。
