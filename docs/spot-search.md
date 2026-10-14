# 場所検索とスポット登録

投稿フォームで、店・観光地・住所を検索して選ぶ、現在地の近くから選ぶ、地図で指した位置から選ぶ、
見つからなければ手入力で登録する、という 4 つの方法で場所を付けられるようにする機能の説明。
場所を付けるのは任意で、付けない投稿はこれまでどおり動く。

最終確認日: 2026-10-14（各サービスの規約・料金はこの日に読んだ内容。変わりうるので本番投入前に読み直すこと）

## 1. まず決めること（運営者の判断が必要）

1. **公開の Nominatim を使うことを、規約を読んだうえで運営者自身が決める。**
   Nominatim の利用規約には「公開 API の利用は、アプリの開発者が熟慮のうえ意図的に選び、規約の順守に直接
   責任を負う場合に限る」「LLM が生成したコードは規約のすべてを守ること」という条項がある。このバックエンドの
   既定値は公開の Nominatim なので、[規約](https://operations.osmfoundation.org/policies/nominatim/) を読み、
   2 章の表の条件を受け入れるか、`NOMINATIM_BASE_URL` を自前または有償のインスタンスに向けるかを決めること。
   `PLACE_USER_AGENT`（サイト名と連絡先）を設定するまでは本番で使わない。
2. **Google に切り替えるなら、フィードを読むたびに課金対象の呼び出しが発生する**（5 章）。
   Google のスポットは名前を保存できないため、フィード 1 ページに Google のスポットが 20 個あれば
   Place Details Pro が 20 回呼ばれる。無料枠（月 5,000 回）は 250 ページ表示で尽きる計算になる。
3. **Google の場所を Google 以外の地図に表示してはいけない**（6 章）。`PLACE_PROVIDER=google` にするなら、
   フロントエンドにブラウザ用のキー（`NEXT_PUBLIC_GOOGLE_MAPS_API_KEY`）と Map ID も設定して、場所選びの地図を
   Google の地図にする（9 章）。設定しなければ地図は OpenStreetMap のタイルのままで、Google の結果は一覧にだけ出し、
   地図で指した位置の名前は調べない（手入力での登録だけになる）。

## 2. 使っているサービスと選んだ理由

| 用途 | 既定（`PLACE_PROVIDER=osm`） | `PLACE_PROVIDER=google` |
|------|------------------------------|--------------------------|
| キーワード検索（店名・観光地・住所・郵便番号） | Nominatim `/search` | Places API (New) Text Search |
| 周辺検索 | Photon `/reverse`（半径とタグで絞り込み） | Places API (New) Nearby Search |
| 地図で選んだ位置の逆引き | Nominatim `/reverse` | Geocoding API v4（逆ジオコーディング）。Google の地図で選んだときだけ（4 章） |
| 登録時・表示時の 1 件の取得 | Nominatim `/lookup` | Places API (New) Place Details |
| キー | 不要 | サーバー側のキーが必要 |
| 結果の保存 | 名前・住所・座標を保存する（帰属表示が条件） | place ID だけ保存する |

既定を OpenStreetMap 系にした理由は、キーも請求先も要らず、データを帰属表示つきで保存できるため。
Google は、サーバー側のキー（`GOOGLE_MAPS_API_KEY`）を設定したときだけ有効になり、キーがなければ
1 回も呼び出さない。

### 各サービスの規約と、このコードの対応

**Nominatim**（[規約](https://operations.osmfoundation.org/policies/nominatim/)）

| 規約 | 対応 |
|------|------|
| 絶対に 1 秒 1 リクエストまで（利用者ごとではなくアプリ全体の合計） | サーバー内の 1 本の待ち行列で、呼び出しの開始を 1.1 秒以上あける（`PLACE_UPSTREAM_MIN_INTERVAL_MS`）。3 秒以上待つことになる呼び出しは送らずに 503 を返す |
| アプリを特定できる User-Agent または Referer（HTTP ライブラリ既定のものは不可） | すべての呼び出しに `PLACE_USER_AGENT` を付ける。未設定なら `sakunoie-lodging-backend/1.0 (TABINOTE place search)`。本番ではサイト名と連絡先を入れた値にする（release モードで未設定なら起動時に警告） |
| オートコンプリートは禁止 | サーバーは 1 回の確定操作につき 1 回だけ検索する。**クライアントは入力のたびに検索してはいけない**（ボタンまたは Enter で検索）。会員ごとに 20 回/分の制限もある |
| 結果はキャッシュすること。同じ問い合わせの繰り返しは遮断されうる | 結果をメモリに 15 分、最大 500 件キャッシュする。同じ検索・同じ位置（丸めた後）・同じ ID は再送しない |
| グリッド状の逆引き、郵便番号や町の全件取得などの体系的な問い合わせは禁止 | 会員の操作 1 回につき 1 回だけ。定期実行や一括処理はない |
| サービスの切り替えを、ソフトウェア更新なしでできること | `NOMINATIM_BASE_URL` を変えて再起動するだけで切り替わる |
| 帰属表示 | 応答の `attribution` をクライアントが結果のそばに表示する（6 章） |
| 個人情報を送らないこと | 検索語と、会員が選んだ位置だけを送る。会員 ID やセッションは送らない。リダイレクトには従わない |

**Photon**（komoot の公開インスタンス。[利用条件](https://photon.komoot.io/)）

「自分のプロジェクトに使ってよいが、公平に。過度の利用は制限される。可用性は保証しない」という条件のみ。
Nominatim と同じ間隔（1.1 秒）・同じキャッシュ・同じ User-Agent を適用している。周辺検索は会員ごとに
10 回/分。公開インスタンスが対応する言語は default / en / de / fr だけなので、日本語と中国語の会員には
現地語の名前、英語と韓国語の会員には英語の名前を返す。

**Overpass は使わない。** 公開インスタンスの説明に、問題のある使い方の例として「OSM マッパー以外に向けた
アプリを作り、公開インスタンスをバックエンドとして頼ること」が挙がっており、この用途はそれに当たるため。

**利用者が増えたら**: 公開サービスはどれも小規模な利用が前提。検索が 1 秒 1 回を超えて混み合う（503 の
「busy」が出る）ようになったら、Nominatim と Photon を自前で立てるか有償のホスティングに切り替え、
`NOMINATIM_BASE_URL` / `PHOTON_BASE_URL` を向け、`PLACE_UPSTREAM_MIN_INTERVAL_MS` を下げる。

### OpenStreetMap 系の検索の限界

- Nominatim は住所・名称の検索で、「佐久 カフェ」のようなカテゴリ検索ではない。カテゴリで探すのは
  周辺検索（`category`）の役割。
- 検索範囲は日本国内（`countrycodes=jp`）。位置を渡さなければ佐久市の周辺を優先する。
- 英語名のタグがない場所は現地語の名前で返る。

## 3. プロバイダーの切り替えと環境変数

`backend/.env`（または実行環境の環境変数）で設定し、サーバーを再起動する。

| 環境変数 | 既定 | 説明 |
|----------|------|------|
| `PLACE_PROVIDER` | `osm`（空でも同じ） | `osm` または `google`。それ以外の値、または `google` でキーが空なら場所検索は無効（503 `place search is not configured`） |
| `PLACE_USER_AGENT` | `sakunoie-lodging-backend/1.0 (TABINOTE place search)` | Nominatim / Photon に送る User-Agent。本番ではサイト名と連絡先を入れる。例: `tabinote-sakunoie/1.0 (https://example.jp; ops@example.jp)` |
| `PLACE_UPSTREAM_MIN_INTERVAL_MS` | `1100` | Nominatim / Photon それぞれへの呼び出し間隔（ミリ秒、0〜60000）。公開の Nominatim を使う間は 1000 未満にしない（release モードで警告）。自前のインスタンスなら 0 でよい |
| `NOMINATIM_BASE_URL` | `https://nominatim.openstreetmap.org` | 自前・有償の Nominatim を使うときに指定 |
| `PHOTON_BASE_URL` | `https://photon.komoot.io` | 自前の Photon を使うときに指定 |
| `GOOGLE_MAPS_API_KEY` | なし | **サーバー側専用**のキー。ブラウザに渡さない。設定すると、`PLACE_PROVIDER=osm` に戻した後も登録済みの Google スポットを表示できる |
| `GOOGLE_PLACES_BASE_URL` | `https://places.googleapis.com` | 通常は変更しない（テスト用） |
| `GOOGLE_GEOCODING_BASE_URL` | `https://geocode.googleapis.com` | 通常は変更しない（テスト用） |

ベース URL は `http(s)://ホスト[/パス]` の形だけを受け付ける（クエリ・フラグメント・認証情報つきは不可。
不正な値ならそのプロバイダーは無効）。リクエストは「ベース URL + 固定のパス + エンコードしたクエリ」で
組み立てるので、利用者の入力で送信先が変わることはない。

切り替え時の注意:

- `osm` → `google`: それまでに登録された OSM のスポットはそのまま使える。新しく選ぶ場所は Google の結果になる。
- `google` → `osm`: Google のスポットは place ID しか保存していないので、キーを残しておけば表示できる。
  キーも外すと、名前は空、`details_available: false` で、地図リンクだけが残る。

## 4. API

すべてログインが必要。検索語や位置が URL（プロキシのログやブラウザの履歴）に残らないよう、読み取りの
3 つも `POST` + JSON ボディにしている。

| エンドポイント | 本文 | 制限（会員ごと） |
|----------------|------|------------------|
| `POST /api/places/search` | `query`（必須、1〜100 文字）, `language`, `latitude` + `longitude`（任意、近くを優先）, `limit`（1〜10、既定 8） | 20 回/分 |
| `POST /api/places/nearby` | `latitude`, `longitude`（必須）, `radius_m`（100〜2000、既定 800）, `category`（`all` `food` `cafe` `shop` `sight` `nature` `onsen` `lodging` `transport`）, `language`, `limit`（1〜20、既定 15） | 10 回/分 |
| `POST /api/places/reverse` | `latitude`, `longitude`（必須）, `language`, `google_map`（Google の地図で選んだ位置なら true。既定 false） | 20 回/分 |
| `POST /api/spots` | 検索結果: `source`（`osm` / `google`）, `external_id`, `language`。手入力: `source: "manual"`, `name`（必須、1〜100 文字）, `address`（〜200 文字）, `latitude` + `longitude`, `category`, `area` | 40 回/日 |

- `language` を省略すると会員の表示言語を使う。
- `POST /api/places/reverse` は、プロバイダーが `google` で `google_map` が true でないとき、Google に問い合わせずに
  `result: null`, `needs_google_map: true` を返す（200）。Google の内容を Google 以外の地図の上に出さないため（6 章）。
  OSM のときは `google_map` を見ない。`needs_google_map` は常に応答に入る（OSM なら false）。
- Google の逆引きは、Geocoding の結果のうち最初の住所（`street_address` `premise` `subpremise`）を返す。なければ
  店・施設（`establishment` `point_of_interest`）と Plus Code 以外の最初の結果（道路・地域など）。Google は指した点に
  ある店を先頭に返すが、その place ID を登録すると Place Details が店の名前を返し、会員が確認した住所と違う名前の
  スポットができてしまうため（2026-10-14 に実サービスで確認）。
- 検索結果には、名前や住所が一致する登録済みスポット（運営のスポット、OSM から登録されたもの、手入力の
  もの）が先頭に入り、`spot_id` が付く。`spot_id` がある結果はそのまま `POST /api/posts` の `spot_id` に使える。
- `POST /api/spots` は同じ外部の場所がすでに登録されていればその行を返す（`created: false`）。
  `spots.external_key`（`source` と `external_id` の SHA-256）の一意制約で、同時に登録されても 1 行になる。
- 検索結果を登録するとき、クライアントが送るのは ID だけ。名前・住所・座標はサーバーがプロバイダーに
  問い合わせた結果を使うので、OSM や Google の場所に任意の名前や住所を付けることはできない。
- 手入力の場所は入力されたとおりに保存する。**座標が入力されなければ座標は NULL のまま**で、推測や既定値で
  埋めない。

エラー:

| 状態 | `detail` | 意味 |
|------|----------|------|
| 503 | `place search is not configured` | プロバイダーが未設定・不明、または `google` でキーなし。外部には一切接続しない |
| 503 + `Retry-After: 3` | `place search is busy, try again shortly` | 上流への待ち行列が 3 秒を超える。少し待って再試行。「未設定」の 503 には `Retry-After` が付かないので、ヘッダの有無で見分けられる |
| 502 | `place search temporarily unavailable` | 上流に届かない・エラー・タイムアウト（5 秒）・不正な応答。上流の URL や本文は応答にもログにも出さない |
| 404 | `place not found` | `external_id` の場所がプロバイダーにない |
| 422 | （内容による） | 入力の検証エラー |
| 404 | `spot not found` | スポットがない、または他の会員がまだ投稿に使っていない手入力のスポット |
| 429 + `Retry-After` | `rate limit exceeded` | 会員ごとの制限を超えた。`Retry-After` は制限が解けるまでの秒数（`X-RateLimit-Reset` と同じ時刻） |

会員ごとの制限は、検証エラーになったリクエストも含めて「試行」を数える（意図した動作。登録の 40 回/日も同じで、
不正な入力を繰り返すと枠を消費する）。busy の `Retry-After` は待ち行列の実際の長さではなく、固定の 3 秒。

プロバイダーが落ちていても、場所なしの投稿、登録済みスポットを付けた投稿、手入力の登録、フィードの表示は動く。

### スポットの地域と絵文字

- 絵文字とカテゴリは、プロバイダーの種別（OSM のタグ、Google の primaryType）から 10 種類のカテゴリ
  （グルメ・カフェ・ショップ・観光・歴史・自然・温泉・宿泊施設・交通・その他）に対応づけて決める。
  手入力では会員が選ぶ（省略時は「その他」📍）。
- 地域（東信・北信・中信・南信）は、OSM の場所なら住所の市町村名から決める（長野県 77 市町村の対応表。
  長野県外は空）。手入力では会員が選べる（任意）。Google の場所は空（Google の内容から作った情報も
  保存しないため）。
- スポットに地域がないとき、投稿の地域には投稿フォームで選んだ `area` を使う。スポットに地域があれば
  これまでどおりスポットの地域が優先される。

### 公開の一覧とランキング

- 公開の `GET /api/spots` は運営が用意したスポット（`source = seed`）だけを返す。会員が登録した場所は
  会員が書いた内容（手入力の名前や住所。自宅の住所かもしれない）であり、ゲスト限定の情報なので、
  ログインなしでは返さない。Google のスポットは、表示のたびに課金対象の呼び出しが要るという理由もある。
- スポットランキングは、運営のスポットと、投稿または公開の足あとが 1 件以上ある登録スポットを対象にする。
  コンポーザーで選ばれただけで投稿されなかった場所は載らない。同点なら運営のスポットが先。
- **手入力のスポットは、それを使った投稿ができるまで、登録した本人にしか存在しない。** 判定は 1 か所
  （`services.visibleSpots`）にあり、「手入力ではない、または自分が登録した、または投稿が 1 件以上ある」
  スポットだけが見える。検索結果だけでなく、ID を指定する操作すべてに同じ規則が掛かる: 他の会員が
  スポットのページを開く・口コミを書くと 404、投稿や足あとの `spot_id` に指定すると存在しない ID と同じ
  422（`spot_id does not match any spot`）になり、ランキングにも出ない。存在しない ID と応答が同じなので、
  スポットがあること自体も分からない。
- 「投稿がある」は読むたびに判定し、過去に投稿されたことは記録しない。**そのスポットを使った最後の投稿が
  削除されると、スポットは再び登録した本人だけのものに戻る。** 投稿の削除は公開の取り消しなので、場所も
  一緒に取り下げる。他の会員の投稿が 1 件でも残っていれば見えたまま。見えていた間に他の会員が書いた口コミや
  足あとの行は残るが、スポットのページごと見えなくなる（再び投稿されればまた表示される）。
- 運営のスポットのうち公共の名所 5 か所（白駒の池、雲場池、松本城、諏訪湖、善光寺）には、地図に表示できるよう
  OpenStreetMap 由来の座標を `cmd/seed` で入れている（既存の行は座標が NULL のときだけ埋める）。
  **「佐久の家」には座標を入れていない。** 家主が住んでいる家で、`GET /api/spots` は公開なので、位置を
  公開するかどうかは運営者が決めること（入れる場合は `cmd/seed/main.go` の `seedSpots` に追加する）。

## 5. Google Maps Platform の設定

> Google 向けの実装は、スタブの HTTP サーバーでの自動テストに加え、**2026-10-14 に運営者のキーで実際の Google の
> サービスに対して確認した**（10 章）。本番で有効にする前に、本番のキーとドメインで 10 章の確認をもう一度行うこと。

### 手順

1. Google Cloud でプロジェクトを作り、**請求先アカウントを関連づける**。公式の手順
   （[Places API のセットアップ](https://developers.google.com/maps/documentation/places/web-service/cloud-setup)）は
   「請求先アカウントのあるプロジェクト」を前提条件にしており、Google Maps Platform の開始ガイドにも
   「Cloud プロジェクトの設定には請求先アカウントの設定が必要」とある。無料枠の範囲なら請求先アカウントなしで
   使える、という記述は見つからなかった（Places API について明記した一文までは確認できていない）。
2. 「API とサービス」→「ライブラリ」で次を有効にする。
   - **Places API (New)** — 検索・周辺検索・1 件の取得
   - **Geocoding API** — 地図で選んだ位置の逆引き（v4 のエンドポイント `geocode.googleapis.com`）。コンソールで
     「Geocoding API」を有効にし、キーの API 制限で許可すれば v4 も使えることを 2026-10-14 に確認した
   - **Maps JavaScript API** — フロントエンドの場所選びの地図（9 章）。`PLACE_PROVIDER=google` にするなら必要
3. **サーバー用のキー**を作る（`GOOGLE_MAPS_API_KEY`）。
   - アプリケーションの制限: **IP アドレス**（API サーバーの送信元 IP だけ）
   - API の制限: **Places API (New)** と **Geocoding API** だけ
   - このキーはブラウザに渡さない。フロントエンドの環境変数（`NEXT_PUBLIC_*`）に入れない。
4. **ブラウザ用のキー**（`NEXT_PUBLIC_GOOGLE_MAPS_API_KEY`。サーバー用とは別のキー）を作る。
   - アプリケーションの制限: **ウェブサイト（HTTP リファラー）**。アプリを開くオリジンをすべて並べる。
     本番 `https://dover.chaso-pa.com/*`、開発 `http://localhost:3000/*` と `http://192.168.3.101:3000/*`
     （LAN 内の端末から開く場合）。別のポートで開発サーバーを動かすならそのポートも（例: `http://localhost:3107/*`）。
     リストにないオリジンでは地図が `RefererNotAllowedMapError` になり、フロントエンドは OpenStreetMap の地図に切り替える。
   - API の制限: **Maps JavaScript API** だけ（Places や Geocoding を許可しない）
   - このキーはビルド時にブラウザ用のコードに埋め込まれ、誰でも読める。悪用を防ぐのは上の 2 つの制限。
5. **Map ID** を作る（「Google Maps Platform」→「地図の管理」→「地図 ID を作成」、種類は JavaScript）。
   `NEXT_PUBLIC_GOOGLE_MAPS_MAP_ID` に入れる。地図のピンは Advanced Markers で、Map ID が必要（未設定のときは
   試用のための `DEMO_MAP_ID` を使う）。Map ID 自体に追加の料金はかからない（料金ページで Dynamic Maps の
   課金対象に「Map ID の有無を問わない」とある）。
6. **予算アラート**を設定する（「お支払い」→「予算とアラート」）。月額の予算と通知のしきい値
   （例: 50% / 90% / 100%）を決める。**予算は通知だけで、利用を止めない。**
   公式の記述: 「予算を設定しても、利用や支出の上限が自動的に設定されるわけではない」。
7. **API ごとの 1 日あたりの上限**を設定する（「API とサービス」→ 各 API →「割り当てとシステム上限」）。
   実際に利用を止められるのはこちら。無料枠に収めるなら、月の無料枠を 31 で割った値より小さくする。
   目安: Text Search 150 回/日、Nearby Search 150 回/日、Place Details 150 回/日、Geocoding 300 回/日、
   Dynamic Maps 300 回/日。上限に達するとその API は 429 を返し、このバックエンドは 502 を返す
   （フィードは名前なしのスポットとして表示を続ける）。Maps JavaScript API の上限超過（`OverQuotaMapError`）では
   地図の中に Google のエラー表示が出る（9 章）。

### 料金（2026-10-13 に Google の料金ページで確認した値。Dynamic Maps は 2026-10-14 に再確認）

| SKU | 月の無料枠 | 最初の有料帯（1,000 回あたり） | この機能での使われ方 |
|-----|-----------|-------------------------------|----------------------|
| Text Search Pro | 5,000 | $32 | キーワード検索 1 回につき 1 回 |
| Nearby Search Pro | 5,000 | $32 | 周辺検索 1 回につき 1 回 |
| Place Details Pro | 5,000 | $17 | **Google のスポットを表示するたびに 1 回**（フィード、スポットのページ、ランキング、登録時） |
| Geocoding | 10,000 | $5 | 地図で位置を選ぶたびに 1 回 |
| Dynamic Maps | 10,000 | $7 | **場所選びの地図を開くたびに 1 回**（ブラウザ用のキーを設定したときだけ） |
| Place Details Essentials | 10,000 | $5 | 使っていない |
| Autocomplete | 10,000 | $2.83 | 使っていない |
| Place Photos | 1,000 | $7 | 使っていない |

- Text Search Essentials（ID のみ）とオートコンプリートのセッション利用は無料枠の上限なし。どちらも使っていない。
- リクエストは、指定したフィールドのうち最も高い帯で課金される。このバックエンドが指定するのは
  `id`, `displayName`, `formattedAddress`, `location`, `primaryType`, `googleMapsUri` だけ。
  場所の名前（`displayName`）が Pro 帯なので、どの呼び出しも Pro になる。Enterprise 帯（評価・営業時間・
  電話番号など）のフィールドは指定していない。
- Dynamic Maps の課金対象は「地図の読み込みの成功（Successful map load）」。地図を開くたびに地図を 1 つ作るので、
  開くたびに 1 回。開いた後のパン・ズームは数えない（SKU の説明に明記）。投稿フォームを開いただけでは読み込まない。
  キーが拒否された読み込みは「成功」ではない。Advanced Markers に別の SKU は料金ページに見当たらない。
- 写真は取得していない（応答の `photo_url` は常に null）。表示 1 枚ごとに Place Photos が 1 回かかり、
  検索結果 8 件に写真を付けるだけで無料枠 1,000 回が 125 回の検索で尽きるため。
- **表示のたびに Place Details がかかる点に注意。** Google の規約で名前をキャッシュできないため、
  フィード 1 ページ（20 件）がすべて別々の Google スポットなら 20 回。1 回のレスポンスの中での重複は
  まとめるが、リクエストをまたいだ再利用はしない。費用が問題になるなら、既定の OSM のままにするか、
  一覧では名前を取得せずスポットのページでだけ取得する形に変える（未実装）。

## 6. 帰属表示

応答の `attribution`（`text` と `url`）を、結果の一覧のそばに必ず表示する。

**OpenStreetMap（`provider: "osm"`、スポットの `source: "osm"`）**

- 表記: `© OpenStreetMap contributors`、リンク先 `https://www.openstreetmap.org/copyright`
- 検索結果の一覧と、OSM 由来のスポットの名前・住所・位置を表示する画面（フィード、スポットのページ）に表示する。
  地図タイルを表示する場合は、タイルの提供元が求める表記も必要。
- データは ODbL。保存したデータを一覧にして再配布するような使い方をする場合は、ライセンスの
  継承（share-alike）条件を確認すること。

**Google（`provider: "google"`、スポットの `source: "google"`）**

- Google の地図なしで Places の内容を表示するときは、**Google Maps のロゴ**を表示する。
  [ポリシー](https://developers.google.com/maps/documentation/places/web-service/policies)（2026-10-08 更新、
  2026-10-14 に確認）: 「Attribution should take the form of the Google Maps logo whenever possible. In cases where
  space is limited, the text Google Maps is acceptable.」 ロゴは高さ 16〜19dp、周囲に上下左右 10dp・下 5dp の余白、
  無地の背景には縁取りなしのロゴ。文字にする場合も「Google Maps」の表記を変えない。
  - フロントエンドは Google の配布物（Google_Maps_Attribution_Assets.zip）の灰色（#5E5E5E）のロゴを変えずに
    使う（`frontend/src/components/GoogleMapsLogo.tsx`、高さ 16px）。検索・周辺検索の一覧の下、地図の下の
    パネル（逆引きの結果）、選んだ場所、投稿カード、スポットのページのすべてで文字ではなくロゴを出す。
    応答の `attribution.text`（`Powered by Google`）は Google のときは表示に使わない。
- Google の地図の上に表示する内容には追加の表示は要らない（ポリシー: 「You don't need to add extra attribution
  if the Content is shown on a Google Map where the attribution is already visible.」）。地図そのものには
  Google のロゴと地図データの表記が Maps JavaScript API によって常に出る。
- **Places の内容を Google 以外の地図に表示してはいけない。** Google Maps Platform 利用規約 3.2.3(e):
  「Customer will not (i) display or use Places content on a non-Google Map」。`PLACE_PROVIDER=google` のとき、
  検索結果や Google のスポットを OpenStreetMap のタイルの地図にピンで表示しない。地図で指した位置の逆引き
  （Geocoding）も、Google の地図の上で指したときだけ行う（4 章、9.5）。
- アプリは、Google の利用規約とプライバシーポリシーを取り込んだ、公開の利用規約とプライバシーポリシーを
  用意する必要がある（ポリシーのページの記載）。

## 7. 保存のルール

`spots` テーブルの `source` で扱いが変わる。

| `source` | 保存するもの | 表示のしかた |
|----------|--------------|--------------|
| `seed` | 運営が用意した内容すべて | 保存した内容 |
| `osm` | 名前（登録した会員の言語）、現地語の名前、住所（日本語）、緯度経度、地図 URL、カテゴリ、地域、OSM の ID、登録した会員 | 保存した内容 |
| `manual` | 会員が入力した名前・住所・緯度経度・カテゴリ・地域、登録した会員 | 保存した内容 |
| `google` | **place ID、`source`、登録した会員、登録日時だけ** | 読むたびに Place Details で取得（閲覧する会員の言語で） |

Google の行は、`name` `category` `area` `description` が空文字、`emoji` が既定の 📍、`original_name`
`address` `latitude` `longitude` `map_url` が NULL のまま。Google のポリシーが保存を認めているのは place ID
だけで（「place ID はキャッシュの制限の対象外」）、それ以外の Places の内容の事前取得・キャッシュ・保存は
認められていないため。あわせて次も守っている。

- Google の結果はメモリにもキャッシュしない（OSM の結果だけ 15 分キャッシュする）。
- Google の内容から作った情報（カテゴリ、地域）も保存しない（利用規約 3.2.3(c) が Google Maps の内容に
  基づくコンテンツの作成を禁じているため）。
- Google の内容（名前・住所・座標）も place ID も API キーもログに出さない。
- 足あと（`gps_visits.place_name`）はスポット名を既定値として写すが、Google のスポットには保存された名前が
  ないので、会員が自分で名前を入力する必要がある（Google の名前が足あとに写ることはない）。

テストでは、Google の結果を登録し、投稿・口コミ・足あと・いいね・ランキング・再登録まで行ったあとで
**データベースの全テーブルの全カラム**を調べ、Google の名前・住所・座標・地図 URL・カテゴリがどこにも
ないことを確認している（`TestGoogleSpotStoresOnlyThePlaceIDIntegration`、
`TestGooglePlaceStorageRuleIntegration`）。

Google の place ID は古くなることがある（店の閉業・統合など）。その場合 Place Details が 404 を返し、
スポットは名前なし（`details_available: false`）で表示される。定期的な更新処理は入れていない。

## 8. 位置情報とプライバシー

- 会員の現在地を受け取るのは周辺検索（`POST /api/places/nearby`）だけ。クライアントは、会員が「近くの
  場所」を求めたときにだけ位置を取得して送る。
- サーバーは位置を**保存しない**。データベースにもログにも書かない（リクエストログはルートのテンプレート・
  状態・所要時間・会員 ID だけ。エラーには上流の状態コードか原因の種類だけが入り、URL は入らない）。
- 外部に送る前に位置を丸める。

  | 用途 | 精度 | 理由 |
  |------|------|------|
  | 周辺検索（会員の現在地） | 小数 3 桁（南北 約 110m、東西 約 90m の格子。ずれは最大で約 70m） | 数百 m の範囲の一覧として意味が保てる範囲で粗くする |
  | キーワード検索の「近くを優先」 | 小数 2 桁（約 1km） | 順位の偏りにしか使わない |
  | 地図で選んだ位置の逆引き | 小数 5 桁（約 1m） | 会員の現在地ではなく、名前を知るために会員が地図で指した点。粗くすると隣の建物の名前が返る |

  応答の `distance_m` は、丸める前の位置からサーバー内で計算する（外部には出ない）。
- メモリ上のキャッシュのキーには丸めた位置が入る（最長 15 分、最大 500 件、プロセス内のみ）。
- 逆引きを現在地で自動的に呼ばないこと（現在地を約 1m の精度で外部に送ることになる）。

### 手入力の場所について

手入力の場所は、それを使った投稿を通じて他の会員に見える。個人の自宅の住所などを公開する手段にならない
よう、次の対策を入れている。

- 登録した会員を記録する（`spots.created_by_member_id`）。
- 登録は会員ごとに 40 回/日まで。
- 公開の `GET /api/spots` には出さない（ログインした会員だけが見られる）。
- 投稿に使われるまでは、登録した本人以外には存在しないものとして扱う（検索・スポットのページ・口コミ・投稿や足あとへの指定・ランキングのすべて。4 章）。最後の投稿が削除されれば元に戻る。
- 名前は 100 文字、住所は 200 文字まで。座標は範囲を検証する。
- UI で「登録した場所は他のゲストに見えます。個人の家の住所は登録しないでください」と警告する（フロントエンド）。

入れていないもの（運営者の判断で追加を検討）:

- 会員による通報と、運営者がスポットを非表示・削除する手段（現在は DB を直接操作するしかない）
- 登録者本人による、未使用の手入力スポットの削除・修正
- スポットのページでの登録者の表示
- 手入力の座標を、一覧や地図では粗く表示する設定

## 9. フロントエンド: 地図表示の設定と手動テスト結果

<!-- ここから下はフロントエンド担当が記入する。バックエンド側では編集しない。 -->

記入日: 2026-10-13（Google の地図は 2026-10-14 に追記）

### 9.1 画面の構成

投稿フォームの「📍訪れた場所を追加（任意）」が場所の入力欄（`frontend/src/features/post/components/PlacePicker.tsx`）。

| 操作 | 動き |
|------|------|
| 入力中 | 運営のスポット（`GET /api/spots`）をブラウザ内で絞り込んで表示する。**通信しない** |
| 「検索」ボタン / Enter | `POST /api/places/search` を 1 回呼ぶ。候補は名前・現地語の名前（`original_name`、あるときだけ）・住所・カテゴリ・「地図」ボタン |
| 「地図から選ぶ」 | 全画面の地図（`PlaceMapModal.tsx`。ブラウザ用の Google のキーがあれば Google の地図、なければ Leaflet の地図。9.3）。ピンをタップ、または空いた位置をタップ・長押しすると `POST /api/places/reverse` の名前と住所を表示し、「この場所にする」で確定、「キャンセル」で閉じる |
| 「現在地の近く」 | 押したときだけブラウザに位置を求め、`POST /api/places/nearby` を呼ぶ |
| 「見つからないときは手動で登録」 | 名前（必須）・住所（任意）・地図上の位置（任意）。保存前に「ほかの宿泊者にも表示される／個人の自宅などは登録しない」の注意を出す |
| 選択後 | 名前と住所を表示し、「変更」「場所を外す」ができる |

- **スポットの登録（`POST /api/spots`）は「シェアする」を押したときに行う。** 場所を選んだだけでは何も作られないので、
  選び直した場所や投稿をやめた場所が残らない。登録に成功して投稿に失敗した場合は、登録済みのスポットを保持して
  再送する（二重登録しない）。
- `spot_id` が付いた検索結果（登録済みのスポット）は登録を呼ばず、そのまま投稿に使う。
- `?spot=<id>` のリンクは `GET /api/spots/{id}` で 1 件取得して選択済みにする（公開の一覧は運営のスポットだけなので、
  会員が登録したスポットのリンクにも対応するため）。
- 名前と住所は応答のまま表示し、翻訳・置換しない（`translate="no"`）。カテゴリは `category_key` を表示言語の
  ラベルにする。
- 投稿カードとスポットのページに、スポットの名前・住所と、`source` に応じた帰属表示（osm:
  `© OpenStreetMap contributors`、google: Google Maps のロゴ）を出す（`frontend/src/components/SpotCredit.tsx`）。

### 9.2 オートコンプリートにしていないことの確認

キーワード検索は「検索」ボタンまたは Enter の確定操作 1 回につき 1 リクエスト。入力のたびには送らない
（Nominatim の規約がクライアント側のオートコンプリートを禁じているため。2 章）。2026-10-13 に実バックエンドで
確認: 2 文字を入力する間の `/api/places/search` は 0 回、Enter で 1 回、同じ語でもう一度 Enter を押しても 1 回のまま
（5 分間はブラウザのメモリ上の結果を使う）。古い検索への応答は中断（AbortSignal）し、新しい結果を上書きしない。
429 を受けたら `Retry-After`（なければ `X-RateLimit-Reset`、どちらもなければ 10 秒、上限 60 秒）の間は送らない。

> 当初の依頼は「入力に合わせて候補を出す（デバウンスつき）」だったが、上記の規約に反するため確定操作での検索に
> している。入力に合わせた候補が必要なら、キーワード検索をそれを認めているサービス（Photon、Google の
> Autocomplete など）に替えたうえで、フロントエンドを変更する。

### 9.3 地図の表示

地図は 2 種類あり、どちらも同じ props を受け取る（`frontend/src/features/post/types/map.ts` の `PlaceMapProps`）。
どちらを使うかは `useMapKind()`（`features/post/hooks/use-map-kind.ts`）が決める: ブラウザ用のキー
`NEXT_PUBLIC_GOOGLE_MAPS_API_KEY` があり、このページで Google の地図が失敗していなければ Google の地図
（`GooglePlaceMap.tsx`）、それ以外は Leaflet の地図（`PlaceMap.tsx`）。

**Google の地図**

| 項目 | 内容 |
|------|------|
| 読み込み | Google 公式の [`@googlemaps/js-api-loader`](https://github.com/googlemaps/js-api-loader) 2.1.3（Apache-2.0、Google が保守。2026-09-22 公開で 7 日の公開待ち期間を満たす）。依存は型定義 `@types/google.maps`（MIT）だけ。Google が推奨する読み込み方（`importLibrary` のブートストラップ）をそのまま包んだもので、React 用の `@vis.gl/react-google-maps` は、使う機能（地図 1 つと独自のピン）に対して大きすぎるため使っていない |
| 読み込むとき | 地図を開いたときだけ（`next/dynamic`、`ssr: false`）。ブラウザ用のキーと読み込み処理はこのチャンクにだけ入る（ビルド後に `.next` を調べて確認: キーの文字列を含むのは地図のチャンク 1 つだけ。サーバー用のキーはどこにも含まれない） |
| 設定 | `v=weekly`、`language`（表示言語。ja / en / ko / zh-CN）、`region=JP`、`mapId`（Map ID）。表示言語はページで最初に地図を開いたときのもの（Maps JavaScript API は 1 ページで 1 回しか読み込めない）。言語を切り替えた後は、次にページを読み込んだときから変わる |
| 操作 | 1 本指で地図を動かせる（`gestureHandling: 'greedy'`。モーダルの背後のページはスクロールしない）。ズームボタンは右下。店のアイコンをタップしても Google の情報ウィンドウは出さず、その位置のタップとして扱う（`clickableIcons: false`） |
| ピン | Advanced Markers（`AdvancedMarkerElement`）に Leaflet と同じ見た目の要素を入れる。タップで選択（`gmp-click`）。現在地の青い点、タップした位置のピンも同じ |
| 帰属表示 | Google のロゴと地図データの表記を Maps JavaScript API が地図の下端に出す |
| 料金 | 開くたびに Dynamic Maps 1 回（5 章） |

**Google の地図が使えないとき**

| 状況 | 検出のしかた | 動き |
|------|--------------|------|
| キーの拒否（無効なキー、リファラーが許可されていない、API が有効でない、請求先がない） | Maps JavaScript API が呼ぶ `window.gm_authFailure`（[公式に記載された検出方法](https://developers.google.com/maps/documentation/javascript/events#auth-errors)） | Leaflet の地図に切り替え、「Google の地図を読み込めなかったため、簡易な地図を表示しています」と表示する |
| スクリプトが読み込めない（オフライン、拡張機能による遮断） | 読み込みの `<script>` のエラー（ローダーの Promise が失敗） | 同上 |
| 15 秒以内に読み込めない | ローダー側のタイムアウト | 同上 |
| 利用上限の超過（`OverQuotaMapError`） | **検出していない**。`gm_authFailure` がこの場合にも呼ばれるかは公式の記載がない | 地図の中に Google のエラー表示が出る。検索・現在地の近く・手入力の登録・投稿は地図と無関係に使える |

切り替えた後はページを読み込み直すまで Leaflet の地図のまま（Maps JavaScript API は読み込み直せないため）。
Leaflet に切り替わると、Google の結果は地図から消え、9.5 の扱いになる。どの状況でも投稿は地図に依存しない。

**Leaflet の地図**

| 項目 | 内容 |
|------|------|
| ライブラリ | [Leaflet](https://leafletjs.com) 1.9.4（BSD-2-Clause）。型は `@types/leaflet`（MIT、開発時のみ）。React 用のラッパーは使っていない（`react-leaflet` は Hippocratic License のため見送り） |
| 読み込み | 地図を開いたときだけ読み込む（`next/dynamic`、`ssr: false`）。投稿フォームを開いただけではタイルも Leaflet も取得しない |
| 既定のタイル | `https://tile.openstreetmap.org/{z}/{x}/{y}.png`（openstreetmap.org の公開タイル。キーもアカウントも不要） |
| 既定の帰属表示 | 地図の右下に `© OpenStreetMap contributors`（`https://www.openstreetmap.org/copyright` へのリンク）と `Leaflet` |
| 最初の表示 | 佐久市役所付近を中心、ズーム 10（軽井沢・小諸・白駒の池が入る範囲） |

**openstreetmap.org タイルの利用規約**（[Tile Usage Policy](https://operations.osmfoundation.org/policies/tiles/)、
2026-10-13 に確認）と、このコードの対応:

| 規約 | 対応 |
|------|------|
| 地図上に帰属表示（`© OpenStreetMap contributors`）をはっきり出す | Leaflet の帰属表示コントロールに常時表示。`NEXT_PUBLIC_MAP_TILE_ATTRIBUTION` を空にしても既定値が出る |
| 正しいタイル URL（`https://tile.openstreetmap.org/{z}/{x}/{y}.png`）を使う | 既定値がこの URL |
| ブラウザは有効な Referer を送ること（Referer を消す設定にしない） | タイルの `<img>` に `referrerpolicy="strict-origin-when-cross-origin"` を付けている。サイト全体に `Referrer-Policy: no-referrer` を設定しても、タイルには送信元が付く |
| キャッシュのヘッダーに従う。`no-cache` を送らない | ブラウザの通常のキャッシュに任せている。独自のヘッダーは付けていない |
| 表示中のタイル以外を先読みしない。一括取得・オフライン保存をしない | Leaflet が表示範囲のタイルだけを取得する。先読み・保存の処理はない |
| 可用性の保証なし（SLA なし）。負荷や違反があれば予告なく遮断されうる | タイルが取れなくても、検索・周辺検索・手動登録・ピンの操作は動く（地図の背景が出ないだけ） |

公開タイルは寄付で運営されている小規模利用向けのサービス。**宿泊者数名の利用なら範囲内だが、利用が増える場合や
確実に表示したい場合は、有償のタイル提供元か自前のタイルサーバーに切り替えること**（下の環境変数を変えて
ビルドし直すだけ）。

### 9.4 フロントエンドの環境変数

すべて任意。`NEXT_PUBLIC_*` はビルド時にブラウザ用のコードへ埋め込まれるので、**秘密の値を入れない**
（`frontend/.env.example` にも同じ説明がある）。

| 変数 | 既定 | 内容 |
|------|------|------|
| `NEXT_PUBLIC_MAP_TILE_URL` | `https://tile.openstreetmap.org/{z}/{x}/{y}.png` | ラスタータイルの URL テンプレート（`{z}` `{x}` `{y}`、必要なら `{s}` `{r}`） |
| `NEXT_PUBLIC_MAP_TILE_ATTRIBUTION` | `© OpenStreetMap contributors`（リンクつき） | 地図に出す帰属表示（HTML）。タイル提供元が求める文言にする |
| `NEXT_PUBLIC_MAP_TILE_MAX_ZOOM` | `19` | タイル提供元が配信する最大ズーム |
| `NEXT_PUBLIC_GOOGLE_MAPS_API_KEY` | なし | **ブラウザ用**の Google Maps Platform のキー。設定すると場所選びの地図が Google の地図になる。サーバー用のキー（`GOOGLE_MAPS_API_KEY`）を入れてはいけない。制限は 5 章の手順 4 |
| `NEXT_PUBLIC_GOOGLE_MAPS_MAP_ID` | `DEMO_MAP_ID` | Map ID（5 章の手順 5）。Advanced Markers に必要 |

タイル URL にキーを含める提供元（`?key=...` 形式）を使う場合、そのキーはブラウザに公開される。提供元の管理画面で
HTTP リファラー（サイトのドメイン）による制限をかけること。

ブラウザ用の Google のキーはビルド時に埋め込まれ、ブラウザで誰でも読める（Google の想定どおり）。悪用を防ぐのは
HTTP リファラーと API の制限なので、必ず両方をかけること。

### 9.5 `provider` が `google` のときの扱い

6 章の制約（Places の内容を Google 以外の地図に表示しない、place ID 以外を保存しない）への対応:

地図が Google の地図かどうか（9.3）で扱いが変わる。

| | Google の地図（ブラウザ用のキーあり） | Leaflet の地図（キーなし、または Google の地図が失敗） |
|---|---|---|
| Google の検索結果・周辺検索の結果・スポットのピン | 地図に出す | **出さない**。一覧には出し、行の「地図」は応答の `map_url`（Google マップ）を新しいタブで開くリンク。地図の下に「Google の検索結果は、この地図には表示されません」 |
| 一覧の行の「地図」 | アプリ内の地図でその場所を開く | 上記の外部リンク |
| 空いた位置のタップ・長押し | `google_map: true` で逆引きし、Geocoding の住所を地図の下に出す | `google_map: false` で送る。サーバーは Google に問い合わせず `needs_google_map: true` を返し、「この地図では、位置から名前を調べられません。名前を入力して登録できます。」と「この位置で名前を入力して登録」だけを出す（Google の Geocoding の内容を OpenStreetMap の地図の横に出さない。Geocoding の料金もかからない） |

- Leaflet の地図での逆引きの扱いは、検討した 3 案のうち「Google のときは地図からの選択を手入力の登録だけにする」
  にした。逆引きだけ OpenStreetMap 系にする案は、Google を選んだ運営者に Nominatim の利用規約（1 章）を別に
  負わせ、Google のスポットと OSM のスポットが混ざるため採らなかった。判定はサーバー側にあり、`google_map` を
  送らない古いクライアントにも Google の内容は返らない。
- 一覧の下、地図の下のパネル、選んだ場所、投稿カード、スポットのページでは、Google の内容のそばに
  Google Maps のロゴを出す（6 章）。
- ブラウザには何も保存しない。`localStorage` / `sessionStorage` / IndexedDB は使っていない。検索結果は
  ページを開いている間のメモリ（TanStack Query、最長 5 分）にだけある。登録時に送るのは `source` と
  `external_id`（place ID）と `language` だけ。

### 9.6 位置情報

- 位置を求めるのは「現在地の近く」を押したときだけ（ページを開いたときには求めない）。
- 取得した位置は、その 1 回の周辺検索のリクエスト本文と、地図上の青い点の表示にだけ使う。ブラウザにも
  サーバーにも保存せず、投稿にも付けない（投稿に入るのは選んだスポットの ID だけ）。逆引きには使わない。
- 拒否・取得失敗・タイムアウト・非対応のそれぞれでメッセージを出し、キーワード検索と地図はそのまま使える。
- 手動登録で位置を選ばなかった場合、`latitude` / `longitude` は送らない（0 や既定値で埋めない）。

### 9.7 手動テストの結果

2026-10-13、ヘッドレス Chrome（DevTools Protocol で操作、390×780 のモバイル表示、位置情報の許可・拒否を
ブラウザの権限で切り替え）。**実機のスマートフォンでは未確認。**

「実」= このブランチのバックエンド（8081、公開の Nominatim / Photon に接続）に対して実行。
「実（障害）」= 同じバックエンドを別のデータベースと設定で起動して実行。「モック」= 応答を差し替えて実行。

| 確認した操作 | 結果 | 対象 |
|--------------|------|------|
| 英語で検索（`Karuizawa Station`）して候補を選ぶ | ○ | 実 |
| 日本語で検索（`佐久平駅`）して候補を選ぶ／結果なし（`ぴんころ地蔵`）の表示 | ○ | 実 |
| 入力中は通信せず、Enter・ボタンで 1 回だけ検索。Enter で投稿が送信されない | ○ | 実 |
| 候補を地図で開く、地図のピンで選ぶ（検索結果・周辺の結果・運営のスポット） | ○ | 実 |
| 地図の空いた位置をタップ／長押し（右クリック相当のイベント）→ 逆引きの名前と住所 → 確定 | ○ | 実 |
| 地図のキャンセルで元の選択が残る | ○ | 実 |
| 現在地の近く（許可）: 15 件表示、距離つき、選択 | ○ | 実 |
| 現在地の近く（拒否）: メッセージ表示、周辺検索は呼ばれない、検索と地図は使える | ○ | 実 |
| 選択の解除と変更 | ○ | 実 |
| 手動登録（位置あり）→ 投稿 → フィードに場所が出る | ○ | 実 |
| 手動登録（位置なし）→ 投稿。スポットの緯度・経度が null | ○ | 実 |
| 新しい場所（OSM）つきの投稿 → フィードに名前・住所・帰属表示・地図リンク | ○ | 実 |
| 場所なしの投稿 | ○ | 実 |
| `?spot=` のリンク（運営のスポット、会員が登録したスポット） | ○ | 実 |
| 登録済みスポットを検索で選ぶと再登録しない | ○ | 実 |
| プロバイダーに届かない（502）: 検索・周辺・逆引きのメッセージ、再試行ボタン、手動登録で投稿できる | ○ | 実（障害） |
| 未設定（503 `not configured`）: メッセージ（再試行なし）、運営のスポットは選べる | ○ | 実（障害） |
| レート制限（429）: メッセージ、待ち時間中は入力・Enter しても送信しない | ○ | 実（障害） |
| 混雑（503 + `Retry-After`）: 「つながりません」と再試行ボタン。未設定の 503 と区別される | ○ | **モックのみ** |
| 日本語・英語・韓国語・中国語: 生のキーなし、横スクロールなし、ボタンの文字切れなし | ○ | 実 |
| 画面キーボード: 入力欄が上に寄り、結果がキーボードの上に見える | ○ | 実（キーボードは表示領域を 440px に縮めて再現。フォーカスはスクリプトで設定） |
| `provider: "google"`: 帰属表示、地図にピンを出さない、登録時に ID だけ送る、ブラウザに保存しない | ○ | **モックのみ** |
| 実機（iOS Safari / Android Chrome）でのタッチ操作・長押し・キーボード | 未実行 | — |

**Google の地図（2026-10-14）。** 同じ方法（ヘッドレス Chrome、390×780、タッチを有効にしてタップは
`Input.dispatchTouchEvent`）で、このブランチのバックエンド（`PLACE_PROVIDER=google`、運営者のサーバー用キーで
実際の Google に接続）と、運営者のブラウザ用キー・Map ID でビルドしたフロントエンド（`http://localhost:3107`、
リファラーに追加済み）に対して実行。「実 Google」= 実際の Maps JavaScript API と Places / Geocoding。

| 確認した操作 | 結果 | 対象 |
|--------------|------|------|
| 地図を開くまで Google へのリクエストがない。開くと日本語の Google の地図、佐久市が中央、運営のスポットのピン | ○ | 実 Google |
| 検索（`佐久平駅`）→ 行の「地図」がアプリ内の地図になる → Google の結果のピンが選択された状態で開く → 確定 | ○ | 実 Google |
| 一覧の下に Google Maps のロゴ（文字の `Powered by Google` は出ない）。「この地図には表示されません」は出ない | ○ | 実 Google |
| ピンのタップで選択（運営のスポット、周辺検索の Google の結果） | ○ | 実 Google |
| 空いた位置のタップ → `google_map: true` で逆引き → 住所とロゴ → 確定 → シェア → フィードのカードに名前・住所・ロゴ | ○ | 実 Google |
| 現在地の近く（許可）: 地図の中のボタンで 15 件、青い点と Google の結果のピンが地図に出る | ○ | 実 Google |
| 現在地の近く（拒否）: メッセージ、周辺検索は呼ばれない、Google の地図はそのまま | ○ | 実 Google |
| 英語・中国語の表示言語で地図の文字がその言語（`language=en` / `zh-CN`、`region=JP`） | ○ | 実 Google |
| 無効なキーでビルド: `InvalidKeyMapError` → `gm_authFailure` → 約 1 秒で Leaflet の地図とメッセージ。閉じて開き直しても Google を読み込み直さない | ○ | 実 Google（無効なキー） |
| スクリプトの遮断（拡張機能相当、DevTools で `BlockedByClient`）→ すぐに Leaflet の地図とメッセージ。逆引きは `google_map: false` で「この地図では…調べられません」、手入力の登録ができる | ○ | 実（遮断はブラウザ側で再現） |
| スクリプトが応答しない → 15 秒で Leaflet の地図とメッセージ | ○ | 実（応答の停止はブラウザ側で再現） |
| ブラウザ用キーなしでビルド、`PLACE_PROVIDER=google`: Leaflet の地図、Google の結果はピンなし・外部リンク・注意書き、タップしても Google に問い合わせない、Google へのリクエスト 0 | ○ | 実 Google（サーバー側） |
| ブラウザ用キーなし、`PLACE_PROVIDER=osm`: Leaflet の地図でタップ → Nominatim の名前と住所・OSM の帰属表示 → 確定（従来どおり） | ○ | 実（公開の Nominatim） |
| 利用上限の超過（`OverQuotaMapError`） | 未実行 | 再現できないため |
| 実機でのタッチ操作・長押し（Google の地図） | 未実行 | — |

<!-- フロントエンド担当の記入欄ここまで -->

## 10. 検証の状況

| 対象 | 状況 |
|------|------|
| OSM プロバイダー（Nominatim / Photon） | スタブでの自動テストに加え、2026-10-13 に公開サービスへ実際に接続して確認（佐久平駅・軽井沢駅の周辺、日本語と英語） |
| Google プロバイダー | スタブでの自動テストに加え、2026-10-14 に運営者のサーバー用キーで実サービスに接続して確認（下記） |
| フロントエンドの Google の地図 | 2026-10-14 に運営者のブラウザ用キーと Map ID で、実際の Maps JavaScript API に対して確認（9.7） |
| エンドポイント、一意制約（同時登録）、Google の保存ルール、ログ、レート制限 | 自動テスト（`backend/internal/services/place_*_test.go`、`backend/internal/server/place_integration_test.go`） |

**2026-10-14 の実サービスでの確認結果**（このブランチのバックエンドを `PLACE_PROVIDER=google` で起動）:

| 確認 | 結果 |
|------|------|
| キーワード検索（英語 `Karuizawa Station`、日本語 `佐久平駅`） | 200。英語は英語名、日本語は日本語名 |
| 周辺検索（カテゴリなし、`food`） | 200。距離つき。`includedTypes` の値で 400 になることはなかった（`food` で確認） |
| 逆引き（Geocoding v4） | 200。「Geocoding API」を有効にしたキーで v4 が使える |
| 検索結果の登録（Place Details） | 200。`spots` の行は place ID だけ（名前・住所・座標・地図 URL・カテゴリは空または NULL） |
| フィードでの名前の解決 | 読む会員の言語で名前が出る（英語の会員には `Sakudaira Station`） |
| 逆引きの結果の登録 | **修正した**: Geocoding は指した点の店を先頭に返し、その place ID を登録すると店の名前（例: 住所を確認したのに「Sakushi Shokuin Workers Union」）になっていた。住所の結果を選ぶように変え（4 章）、修正後は確認した住所と同じ名前（`中込３０６７`）で登録されることを確認 |

まだ確認していないこと: フィード 1 ページでの Place Details の回数を Cloud Console で見ること（下の 3）、
キーを無効にしたときのフィード（下の 4）、カテゴリつき周辺検索の全カテゴリ。

本番で Google を有効にする前に、本番のキーで確認すること:

1. キーワード検索（日本語・英語）、周辺検索（カテゴリあり・なし）、逆引き、登録が 200 で返る。
   - 周辺検索のカテゴリは `includedTypes` に `restaurant` `cafe` `store` `tourist_attraction` `museum` `park`
     `national_park` `spa` `lodging` `train_station` `bus_station` を送る。Google の種別表にない値があると
     400 になり、このバックエンドは 502 を返す。
   - 逆引きは Geocoding API v4（`GET /v4/geocode/location?location.latitude=..&location.longitude=..`）。
2. 登録後の `spots` の行に place ID 以外の Google の内容がない。
3. フィードを 1 ページ表示したときの Place Details の回数を Cloud Console で確認する。
4. キーを無効にしたとき、フィードが名前なしのスポットで表示され続ける。
