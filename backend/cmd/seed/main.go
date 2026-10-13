// Command seed inserts the sample data ported from the prototype
// (docs/pages/sns.html). It is idempotent: rows have fixed ids and existing
// rows are left untouched, so it is safe to run repeatedly.
//
// Usage (from backend/): go run ./cmd/seed
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"log"
	"time"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/config"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/database"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/services"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// seedNamespace makes sample row ids stable across runs.
var seedNamespace = uuid.MustParse("6f1d1c5e-6c0e-4f57-9a0e-5a6b0d7c2a11")

func seedID(kind, key string) string {
	return uuid.NewSHA1(seedNamespace, []byte(kind+":"+key)).String()
}

type seedMember struct {
	nickname string
	language string
}

// The first five are the prototype's post authors, the last three its
// review authors.
var seedMembers = []seedMember{
	{"Mika", "ja"},
	{"旅するKen", "ja"},
	{"Yui", "ja"},
	{"TABINOTE編集部", "ja"},
	{"Emily · UK", "en"},
	{"Ava · UK", "en"},
	{"Kenta", "ja"},
	{"Mina · Korea", "ko"},
}

// Positions of the public landmarks, from OpenStreetMap (© OpenStreetMap
// contributors), so the place picker can draw the curated spots on its map.
// 佐久の家 has none on purpose: it is a home the host lives in, GET /spots is
// public, and whether to publish its position is the host's decision.
var seedSpots = []models.Spot{
	{Name: "白駒の池", Area: "東信", Category: "自然", Emoji: "🌲", Description: "苔の森に囲まれた、静かな湖。朝の散策がおすすめ。", Latitude: at(36.0512), Longitude: at(138.3617)},
	{Name: "軽井沢・雲場池", Area: "東信", Category: "自然", Emoji: "🍁", Description: "水面に映る木々が美しい散策スポット。", Latitude: at(36.3521), Longitude: at(138.6269)},
	{Name: "佐久の家", Area: "東信", Category: "宿泊", Emoji: "🏡", Description: "長野県佐久市にある、旅の拠点になる家主居住型の宿。"},
	{Name: "松本城", Area: "中信", Category: "歴史", Emoji: "🏯", Description: "黒い天守が印象的な国宝。城下町さんぽも楽しめます。", Latitude: at(36.2386), Longitude: at(137.9689)},
	{Name: "諏訪湖", Area: "南信", Category: "自然", Emoji: "🌅", Description: "湖畔の散歩や夕景を楽しめるスポット。", Latitude: at(36.0479), Longitude: at(138.0833)},
	{Name: "善光寺", Area: "北信", Category: "歴史", Emoji: "⛩️", Description: "門前町の食べ歩きと合わせて訪れたい名所。", Latitude: at(36.6614), Longitude: at(138.1877)},
}

func at(v float64) *float64 { return &v }

type seedPost struct {
	key      string
	user     string
	title    string
	place    string
	date     string
	weather  string
	temp     float64
	body     string
	tags     []string
	language string
	// likes is how many sample members like the post. The prototype's like
	// counts (16 to 42) exceed the number of sample members, so these keep
	// only its ordering.
	likes int
}

var seedPosts = []seedPost{
	{"1", "Mika", "朝の苔の森を歩く", "白駒の池", "2026-10-08", "晴れ", 13, "静かな森の空気が気持ちよかった。足元の苔まできれいで、ゆっくり歩きたくなる場所。", []string{"自然", "朝さんぽ"}, "ja", 5},
	{"2", "旅するKen", "軽井沢で深呼吸", "軽井沢・雲場池", "2026-10-05", "くもり", 16, "水面に映る秋の色。駅周辺でカフェに寄るのもおすすめ。", []string{"軽井沢", "秋旅"}, "ja", 4},
	{"3", "Yui", "城下町とコーヒー", "松本城", "2026-09-29", "晴れ", 21, "お城を見てから、近くの路地を散策。歩いているだけで楽しい一日でした。", []string{"歴史", "カフェ"}, "ja", 6},
	{"4", "TABINOTE編集部", "湖畔で夕焼けを待つ", "諏訪湖", "2026-09-22", "晴れ", 22, "空の色がゆっくり変わる時間。湖のそばでのんびり過ごしました。", []string{"夕景", "湖"}, "ja", 7},
	{"5", "Emily · UK", "A peaceful morning by the pond", "軽井沢・雲場池", "2026-10-10", "晴れ", 14, "Such a peaceful place in the morning! The reflections on the water were beautiful, and the little cafés nearby were perfect for a warm coffee.", []string{"Karuizawa", "cafe", "autumn"}, "en", 3},
}

// The prototype shows these on the 佐久の家 profile.
var seedReviews = []struct {
	user  string
	stars int
	text  string
}{
	{"Ava · UK", 5, "The room was clean and cozy. The local recommendations were really helpful!"},
	{"Kenta", 5, "家のようにくつろげて、観光の拠点にぴったり。"},
	{"Mina · Korea", 5, "佐久のカフェ巡りが楽しかったです。次は紅葉の季節に来たい！"},
}

const reviewSpot = "佐久の家"

type seedAiSpot struct {
	name     string
	purposes []string
	area     string
	kind     string
	drive    int
	transit  int
	stay     int
	genres   []string
	weathers []string
	desc     string
	mapQuery string
	url      string
}

var seedAiSpots = []seedAiSpot{
	{"龍岡城五稜郭", []string{"japan", "scenic", "family"}, "佐久市田口", "歴史・穴場", 25, 55, 45, []string{"穴場スポット", "歴史・文化"}, []string{"any", "sunny", "rainy"}, "函館だけじゃない、佐久にある星形の城郭。静かな町並みを歩きながら、珍しい五角形の歴史に触れられます。", "龍岡城五稜郭 佐久市", "https://www.google.com/search?q=龍岡城五稜郭+佐久市+公式"},
	{"新海三社神社", []string{"japan", "scenic"}, "佐久市田口", "森・歴史", 30, 65, 35, []string{"穴場スポット", "自然・絶景", "歴史・文化"}, []string{"any", "sunny", "rainy"}, "木々に包まれた境内と歴史ある社殿が魅力。観光地のにぎわいから少し離れて、森の静けさを味わいたい人に。", "新海三社神社 佐久市", "https://www.google.com/search?q=新海三社神社+公式"},
	{"五稜郭展望台（田口城跡）", []string{"hiking", "scenic"}, "佐久市田口", "絶景・穴場", 27, 60, 50, []string{"穴場スポット", "自然・絶景", "歴史・文化"}, []string{"any", "sunny"}, "高台から田園風景を見渡すスポット。遊歩道を歩くため、歩きやすい靴で。晴れの日の寄り道にぴったり。", "五稜郭展望台 田口城跡 佐久市", "https://www.google.com/search?q=五稜郭展望台+田口城跡"},
	{"YUSHI CAFE", []string{"karuizawa", "rainyday"}, "佐久市協和", "古民家カフェ", 25, 60, 60, []string{"穴場スポット", "カフェ・グルメ"}, []string{"any", "sunny", "rainy"}, "落ち着いた雰囲気でひと休みできるカフェ。自然や町歩きの合間に、地元の時間をゆっくり楽しみたい人に。", "YUSHI CAFE 佐久市", "https://www.google.com/search?q=YUSHI+CAFE+佐久市"},
	{"稲荷山公園", []string{"hiking", "scenic", "family"}, "佐久市臼田", "景色・散策", 25, 50, 40, []string{"穴場スポット", "自然・絶景"}, []string{"any", "sunny"}, "高台の景色と散策を楽しめる公園。天気のいい日に、地元の空気を感じながらゆっくり歩きたいときに。", "稲荷山公園 佐久市", "https://www.google.com/search?q=稲荷山公園+佐久市"},
	{"Uchide CAFE", []string{"rainyday", "family"}, "佐久市野沢", "ローカルカフェ", 15, 30, 45, []string{"穴場スポット", "カフェ・グルメ"}, []string{"any", "sunny", "rainy"}, "野沢エリアでカフェタイムを楽しむなら候補に。近くのお店や町並みと合わせて、気軽な寄り道に。", "Uchide CAFE 佐久市", "https://www.google.com/search?q=Uchide+CAFE+佐久市"},
	{"旧軽井沢銀座通り", []string{"karuizawa", "japan", "family"}, "軽井沢町", "街歩き・ショッピング", 40, 65, 120, []string{"カフェ・グルメ", "歴史・文化", "穴場スポット"}, []string{"any", "sunny", "rainy"}, "お店やカフェを巡りながら、軽井沢らしい街歩きを楽しめる定番エリア。混雑しやすい時期は時間に余裕を持って。", "旧軽井沢銀座通り 軽井沢", "https://www.google.com/search?q=旧軽井沢銀座通り+公式"},
	{"星野温泉 トンボの湯", []string{"onsen", "karuizawa"}, "軽井沢町・星野", "日帰り温泉", 40, 70, 100, []string{"自然・絶景", "穴場スポット"}, []string{"any", "sunny", "rainy"}, "森に囲まれた温泉でゆっくり過ごしたい日に。営業時間・料金・休館日は訪問前に公式情報を確認してね。", "星野温泉 トンボの湯 軽井沢", "https://www.google.com/search?q=星野温泉+トンボの湯+公式"},
	{"布引観音 釈尊寺", []string{"hiking", "japan", "scenic"}, "小諸市", "歴史・ハイキング", 25, 55, 75, []string{"自然・絶景", "歴史・文化", "穴場スポット"}, []string{"sunny"}, "山あいの景色と歴史を楽しめるスポット。坂道や歩きにくい場所があるため、歩きやすい靴で、天候を確認して訪れてね。", "布引観音 釈尊寺 小諸市", "https://www.google.com/search?q=布引観音+釈尊寺+公式"},
	{"佐久市立近代美術館", []string{"rainyday", "japan", "family"}, "佐久市猿久保", "美術館・屋内", 12, 30, 60, []string{"歴史・文化", "カフェ・グルメ"}, []string{"any", "rainy"}, "雨の日にも立ち寄りやすい屋内スポット。展示内容や開館日は事前に確認してから出かけよう。", "佐久市立近代美術館", "https://www.google.com/search?q=佐久市立近代美術館+公式"},
	{"白駒の池", []string{"hiking", "scenic"}, "佐久穂町・北八ヶ岳", "森・湖・ハイキング", 65, 120, 120, []string{"自然・絶景", "穴場スポット"}, []string{"sunny"}, "苔の森と湖の景色が魅力の自然スポット。標高が高く天候が変わりやすいので、季節・道路状況・装備を確認してね。", "白駒の池 長野", "https://www.google.com/search?q=白駒の池+公式"},
}

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config: ", err)
	}
	db, err := database.Open(cfg.GormDatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect database: ", err)
	}
	if err := db.Transaction(seed); err != nil {
		log.Fatal("Seed failed: ", err)
	}

	for _, table := range []string{"members", "spots", "posts", "post_tags", "likes", "reviews", "ai_spots"} {
		var n int64
		if err := db.Table(table).Count(&n).Error; err != nil {
			log.Fatal("Count failed: ", err)
		}
		log.Printf("%-10s %d rows", table, n)
	}
	log.Println("Seed completed")
}

func seed(tx *gorm.DB) error {
	// Existing rows win, which keeps re-runs from touching anything.
	// Session() makes the clause reusable across the Create calls below.
	tx = tx.Clauses(clause.OnConflict{DoNothing: true}).Session(&gorm.Session{})

	for _, m := range seedMembers {
		hash, err := unusableTransferCodeHash()
		if err != nil {
			return err
		}
		member := models.Member{
			ID:                seedID("member", m.nickname),
			Nickname:          m.nickname,
			PreferredLanguage: m.language,
			TransferCodeHash:  hash,
			SessionVersion:    1,
		}
		if err := tx.Create(&member).Error; err != nil {
			return err
		}
	}

	for _, s := range seedSpots {
		s.ID = seedID("spot", s.Name)
		s.IsSample = true
		s.Source = models.SpotSourceSeed
		if err := tx.Create(&s).Error; err != nil {
			return err
		}
		// Rows seeded before spots had positions get theirs here. A position
		// already on the row is left alone.
		if s.Latitude != nil && s.Longitude != nil {
			if err := tx.Model(&models.Spot{}).
				Where("id = ? AND source = ? AND latitude IS NULL AND longitude IS NULL", s.ID, models.SpotSourceSeed).
				Updates(map[string]any{"latitude": *s.Latitude, "longitude": *s.Longitude}).Error; err != nil {
				return err
			}
		}
	}

	for _, p := range seedPosts {
		date, err := models.ParseCalendarDate(p.date)
		if err != nil {
			return err
		}
		// Posted at noon, Japan time, on the day of the visit.
		postedAt, err := time.ParseInLocation(models.DateLayout, p.date, time.FixedZone("Asia/Tokyo", 9*60*60))
		if err != nil {
			return err
		}
		postID := seedID("post", p.key)
		spotID := seedID("spot", p.place)
		weather, temp := p.weather, p.temp
		var area *string
		for _, s := range seedSpots {
			if s.Name == p.place {
				area = &s.Area
			}
		}
		// The schema has no title column, so the title leads the caption.
		post := models.Post{
			ID:          postID,
			MemberID:    seedID("member", p.user),
			SpotID:      &spotID,
			Caption:     p.title + "\n" + p.body,
			Area:        area,
			Language:    p.language,
			Weather:     &weather,
			Temperature: &temp,
			VisitedAt:   &date,
			IsSample:    true,
			CreatedAt:   postedAt.Add(12 * time.Hour),
		}
		if err := tx.Create(&post).Error; err != nil {
			return err
		}
		for _, tag := range p.tags {
			row := models.PostTag{ID: seedID("post_tag", p.key+":"+tag), PostID: postID, Tag: tag}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
		for _, m := range seedMembers[:p.likes] {
			row := models.Like{
				ID:       seedID("like", p.key+":"+m.nickname),
				MemberID: seedID("member", m.nickname),
				PostID:   postID,
			}
			if err := tx.Create(&row).Error; err != nil {
				return err
			}
		}
	}

	for _, r := range seedReviews {
		review := models.Review{
			ID:       seedID("review", r.user),
			MemberID: seedID("member", r.user),
			SpotID:   seedID("spot", reviewSpot),
			Stars:    r.stars,
			Text:     r.text,
			IsSample: true,
		}
		if err := tx.Create(&review).Error; err != nil {
			return err
		}
	}

	for _, s := range seedAiSpots {
		spot := models.AiSpot{
			ID:          seedID("ai_spot", s.name),
			Name:        s.name,
			Purposes:    jsonArray(s.purposes),
			Area:        s.area,
			Kind:        s.kind,
			DriveMin:    s.drive,
			TransitMin:  s.transit,
			StayMin:     s.stay,
			Genres:      jsonArray(s.genres),
			Weathers:    jsonArray(s.weathers),
			Description: s.desc,
			MapQuery:    s.mapQuery,
			SearchURL:   s.url,
		}
		if err := tx.Create(&spot).Error; err != nil {
			return err
		}
	}
	return nil
}

// unusableTransferCodeHash returns the hash of 32 random bytes, which no
// 12-character transfer code can produce: sample members cannot be restored.
func unusableTransferCodeHash() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return services.HashTransferCode(hex.EncodeToString(b)), nil
}

func jsonArray(values []string) string {
	b, err := json.Marshal(values)
	if err != nil {
		panic(err)
	}
	return string(b)
}
