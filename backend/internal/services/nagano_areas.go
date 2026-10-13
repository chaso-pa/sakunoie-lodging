package services

// naganoAreas files each of Nagano Prefecture's 77 municipalities under one
// of the four regions the feed is filtered by. The regions follow the
// prefecture's ten regional development bureaus: 佐久 and 上田 are 東信;
// 長野 and 北信 are 北信; 松本, 木曽 and 北アルプス are 中信; 諏訪, 上伊那 and
// 南信州 are 南信.
var naganoAreas = func() map[string]string {
	regions := map[string][]string{
		"東信": {
			"小諸市", "佐久市", "小海町", "川上村", "南牧村", "南相木村", "北相木村", "佐久穂町", "軽井沢町", "御代田町", "立科町",
			"上田市", "東御市", "青木村", "長和町",
		},
		"北信": {
			"長野市", "須坂市", "千曲市", "坂城町", "小布施町", "高山村", "信濃町", "小川村", "飯綱町",
			"中野市", "飯山市", "山ノ内町", "木島平村", "野沢温泉村", "栄村",
		},
		"中信": {
			"松本市", "塩尻市", "安曇野市", "麻績村", "生坂村", "山形村", "朝日村", "筑北村",
			"上松町", "南木曽町", "木祖村", "王滝村", "大桑村", "木曽町",
			"大町市", "池田町", "松川村", "白馬村", "小谷村",
		},
		"南信": {
			"岡谷市", "諏訪市", "茅野市", "下諏訪町", "富士見町", "原村",
			"伊那市", "駒ヶ根市", "辰野町", "箕輪町", "飯島町", "南箕輪村", "中川村", "宮田村",
			"飯田市", "松川町", "高森町", "阿南町", "阿智村", "平谷村", "根羽村", "下條村", "売木村", "天龍村", "泰阜村", "喬木村", "豊丘村", "大鹿村",
		},
	}
	byName := map[string]string{}
	for area, names := range regions {
		for _, name := range names {
			byName[name] = area
		}
	}
	return byName
}()

// NaganoArea returns the region of a Nagano municipality by its Japanese
// name ("佐久市" → "東信"), or "" for anything else. Several of these names
// exist in other prefectures too, so the caller must already know the
// place is in Nagano.
func NaganoArea(municipality string) string {
	return naganoAreas[municipality]
}
