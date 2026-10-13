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

### 日付とタイムゾーン

訪問日（投稿の `visited_at`、足あとの `visited_date`）は時刻もタイムゾーンも持たない暦日として扱い、文字列の
まま保存・返却する。サーバーのタイムゾーンが変わっても日付はずれない。フィードの月での絞り込みは、訪問日が
あればその暦日で、なければ投稿日時を日本時間（Asia/Tokyo）で見た月で判定する。

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
512px 四方に収まるよう縮小する。投稿を削除すると、その写真のオブジェクトをすべて削除する（削除に失敗した
場合はログに残し、オブジェクトだけがバケットに残る）。

### 投稿の写真（1 投稿につき 5 枚まで）

`POST /api/posts`（multipart/form-data）は `media` パートを写真の枚数分くり返して受け取る。0〜5 枚。
送った順番が表示順になり、`post_media.sort_order`（0 始まり）に保存する。応答の `media` 配列もその順。

| 上限 | 値 | 超えたとき |
|------|----|-----------|
| 枚数 | 5 枚 | 422、`errors[0].message` が `too_many_photos`（`location` は `body.media`） |
| 写真 1 枚 | 4MB | 422、`photo_too_large`（`location` は `body.media[i]`、i は 0 始まり） |
| リクエスト全体 | 10MB（全エンドポイント共通） | 422、`errors[0].message` に `request body too large` を含む |

画像として読めないファイルが 1 つでもあると、投稿全体を 422（`unsupported_image`、`location` は
`body.media[i]`）で断る。`Content-Type` が JPEG / PNG / WebP 以外と申告されたパートは、その前に Huma が
422（`Invalid mime type`、`location` は `media[i]`）で断る。どの場合も `detail` に人が読める文が入る。

保存は「全部か、なし」。すべての写真を検査・再エンコードし終えてから 1 枚目を書き込むので、断られた
リクエストはストレージに何も残さない。書き込みの途中でストレージが失敗した場合（503）と、書き込み後に
DB への登録が失敗した場合は、そのリクエストで書き込んだオブジェクトを削除し、投稿も `post_media` の行も
残さない。

写真は 1 枚ずつ順番に処理する（並行しない）。スマートフォンの写真は 1 枚のデコードに数十 MB を使うため、
小さなサーバーで同時に 5 枚を展開しないようにしている。書き込みまで保持するのは再エンコード後の JPEG
（1 枚あたり数百 KB）だけ。投稿のレート制限（1 日 20 回）は写真の枚数ではなくリクエスト数で数える。

フロントエンドは送信前に各写真を長辺 1600px の JPEG に縮小するので、通常は 1 枚あたり数百 KB、5 枚でも
数 MB に収まる。10MB はリクエスト全体の上限のまま変えていない（Next.js の中継バッファ 12MB もそのまま）。

### 外部サービス

| 機能 | 環境変数 | 未設定時 |
|------|----------|----------|
| AI 旅行アシスタント | `OPENROUTER_API_KEY`, `OPENROUTER_MODEL` | `POST /api/ai/plan` が 503（外部呼び出しなし） |
| 翻訳 | `TRANSLATION_PROVIDER`（`mymemory` または `openrouter`） | 空・未知の値、または `openrouter` でキーが空なら 503。翻訳結果は `post_translations` にキャッシュする |
| 天気 | なし（open-meteo、キー不要） | — |
| 場所検索 | `PLACE_PROVIDER`（`osm` または `google`。空なら `osm`）ほか | 未知の値、または `google` でキーが空なら場所検索の 4 エンドポイントが 503。手入力の場所登録と投稿は動く |

タイムアウトは AI 30 秒、翻訳 10 秒、天気 5 秒、場所検索 5 秒。外部サービスの失敗は 502 で、応答にもログにも
外部 URL・API キー・外部の応答本文は出さない。

### 場所検索とスポットの登録

投稿に付ける場所を、キーワード検索（`POST /api/places/search`）、現在地の周辺検索
（`POST /api/places/nearby`）、地図で選んだ位置の逆引き（`POST /api/places/reverse`）で探し、
選んだ結果または手入力の場所を `POST /api/spots` でスポットにする。同じ外部の場所は誰が登録しても 1 行になる。
すべてログインが必要で、会員ごとのレート制限がある（検索 20 回/分、周辺 10 回/分、逆引き 20 回/分、登録 40 回/日）。

既定の取得元は OpenStreetMap 系の公開サービス（Nominatim と Photon）で、利用規約に合わせて呼び出しを
約 1 秒に 1 回に抑え、結果を 15 分キャッシュする。**公開の Nominatim はオートコンプリートを禁止している
ので、クライアントは入力のたびではなく確定時にだけ検索する。** `PLACE_PROVIDER=google` にすると
Places API (New) を使う。Google の結果から登録したスポットは place ID だけを保存し、名前・住所・座標は
読むたびに Google から取得する（保存もキャッシュもログ出力もしない）。

公開の `GET /api/spots` が返すのは運営が用意したスポット（`source = seed`）だけで、会員が登録した場所は
含めない。サービスの選び方、環境変数、Google Cloud の設定、帰属表示、料金は
[`docs/spot-search.md`](../docs/spot-search.md) を参照。

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
`sakunoie_test_<パッケージ名>_<プロセス ID>` という専用データベースを作り、`prisma/migrations` の SQL を
適用して使う（開発用の `sakunoie` データベースには触れない）。名前にプロセス ID を含むので、スイートを同時に
複数実行しても互いのデータベースを壊さない。データベースはパッケージのテスト終了時に削除する。強制終了などで
残ったものは、MySQL が同じマシンにある場合に限り、次回の実行時にまとめて削除する。

MySQL に接続できない場合、これらのテストは失敗ではなくスキップになる。スキップを失敗として扱いたいとき
（CI など）は `REQUIRE_TEST_DB=1 go test ./...` とする（MinIO に届かない場合も失敗になる）。
AI・翻訳・天気・場所検索のテストはスタブの HTTP サーバーを使い、実サービスには接続しない
（Google の実装はスタブでのみ検証しており、実サービスでは動かしていない）。

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

API は `/api` 配下。debug モードでは公開 API のドキュメントを `/api/docs`、認証必須 API のドキュメントを
`/api/docs/auth`（ログインが必要）で見られる。**release モード（`GIN_MODE=release`）ではドキュメント UI・
OpenAPI・JSON スキーマをどちらも配信しない**（404）。その場合、応答本文の `$schema` フィールドも付かない。
規約は `CLAUDE.md` を参照。

すべての応答に `Cache-Control: no-store`、`X-Content-Type-Options: nosniff`、`X-Frame-Options: DENY` を付け、
release モードでは `Strict-Transport-Security` も付ける。

サーバーは SIGINT / SIGTERM で新規接続の受付を止め、処理中のリクエストを最大 40 秒待ってから終了する。
