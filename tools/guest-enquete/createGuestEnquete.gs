/**
 * 民泊の宿泊後アンケート（日本語・英語併記）を作成するGAS
 *
 * 使い方：
 * 1. https://script.google.com/ を開き、「新しいプロジェクト」を作成
 * 2. このコードを貼り付けて保存
 * 3. createGuestFeedbackForm を実行し、必要な権限を許可
 * 4. 実行ログに表示されるフォームURL・編集URL・回答集計シートURLを確認
 *
 * 注意：
 * - 実行するたびに新しいフォームとスプレッドシートが作成されます。
 * - このスクリプトはGoogleアカウント内にファイルを作成します。
 */

function createGuestFeedbackForm() {
  const form = FormApp.create('ご宿泊後のアンケート | Guest Feedback Form');

  form.setDescription(
    'この度はご宿泊いただき、誠にありがとうございました。\n' +
    '今後、より快適にお過ごしいただける宿づくりのため、ぜひ率直なご感想をお聞かせください。回答は1〜2分ほどで完了します。\n\n' +
    'Thank you for staying with us! We would love to hear about your experience. ' +
    'Your feedback helps us improve our accommodation. This short survey takes about 1–2 minutes.\n\n' +
    '回答は任意です。ご協力ありがとうございます。\n' +
    'All questions are optional. Thank you for your time!'
  );

  form.setConfirmationMessage(
    'アンケートにご協力いただき、ありがとうございました！\n' +
    '皆さまからのお声を、より心地よい宿づくりに活かしてまいります。\n\n' +
    'Thank you for your valuable feedback. We hope to welcome you again!'
  );
  form.setProgressBar(true);
  form.setShuffleQuestions(false);
  form.setAcceptingResponses(true);

  // Q1：旅の目的（複数選択）
  form.addCheckboxItem()
    .setTitle('Q1. 今回の旅の目的を教えてください。（複数選択可）\nWhat was the purpose of your trip? (Select all that apply)')
    .setChoiceValues([
      '観光 / Sightseeing',
      '軽井沢・小諸・上田方面へのお出かけ / Visiting Karuizawa, Komoro or Ueda',
      'アウトドア・スキー / Outdoor activities & skiing',
      '温泉 / Hot springs',
      '帰省・友人や知人の訪問 / Visiting family or friends',
      'イベント・冠婚葬祭 / Event or ceremony',
      '仕事・出張 / Business',
      'ワーケーション / Workation',
      '移住・二拠点生活の下見 / Considering relocation',
      '旅の途中の中継地点 / Stopover'
    ])
    .showOtherOption(true)
    .setRequired(false);

  // Q2：総合満足度
  form.addScaleItem()
    .setTitle('Q2. 総合的な満足度を教えてください。\nHow would you rate your overall stay?')
    .setBounds(1, 5)
    .setLabels('1：不満 / Poor', '5：とても満足 / Excellent')
    .setRequired(false);

  // Q3：清潔さ
  form.addScaleItem()
    .setTitle('Q3. お部屋や共用スペースの清潔さはいかがでしたか？\nHow would you rate the cleanliness of the room and shared spaces?')
    .setBounds(1, 5)
    .setLabels('1：不満 / Poor', '5：とても満足 / Excellent')
    .setRequired(false);

  // Q4：チェックイン案内の分かりやすさ
  form.addScaleItem()
    .setTitle('Q4. 事前のご案内や到着時のチェックインの説明は分かりやすかったですか？\nWere the pre-arrival information and check-in instructions easy to understand?')
    .setBounds(1, 5)
    .setLabels('1：分かりにくい / Unclear', '5：とても分かりやすい / Very clear')
    .setRequired(false);

  // Q5：気に入った点（複数選択）
  form.addCheckboxItem()
    .setTitle('Q5. 特に気に入ったところは何ですか？（複数選択可）\nWhat did you like most? (Select all that apply)')
    .setChoiceValues([
      'お部屋・インテリア / Room & interior',
      '清潔さ / Cleanliness',
      '設備・アメニティ / Facilities & amenities',
      'お菓子・コーヒーなど / Snacks & coffee',
      'ホストの対応 / Host hospitality',
      '佐久市の環境・立地 / Location & surroundings',
      'その他 / Other'
    ])
    .setRequired(false);

  // Q6：改善点の有無
  form.addMultipleChoiceItem()
    .setTitle('Q6. お部屋や設備で、気になった点はありましたか？\nWas there anything about the room or facilities that could be improved?')
    .setChoiceValues([
      '特になし / Nothing in particular',
      '少しあった / A few things',
      '改善してほしい点がある / Yes, some improvements needed'
    ])
    .setRequired(false);

  // Q7：改善してほしい点（任意）
  form.addParagraphTextItem()
    .setTitle('Q7. お部屋・設備・サービスについて、改善してほしい点やご要望があれば具体的に教えてください。（任意）\nWhat could we improve about the room, facilities or service? Please be specific. (Optional)')
    .setRequired(false);

  // Q8：再宿泊意向
  form.addMultipleChoiceItem()
    .setTitle('Q8. またこちらに宿泊したいと思いますか？\nWould you stay with us again?')
    .setChoiceValues([
      'ぜひ泊まりたい / Definitely',
      '機会があれば / Maybe',
      'わからない / Not sure'
    ])
    .setRequired(false);

  // Q9：滞在の感想・メッセージ（任意）
  form.addParagraphTextItem()
    .setTitle('Q9. 滞在の感想や思い出、ホストへのメッセージなど、自由にお聞かせください。（任意）\nPlease share any thoughts about your stay, favorite moments, or a message to your hosts. (Optional)')
    .setRequired(false);

  // 回答先のスプレッドシートを作成して連携
  const responseSheet = SpreadsheetApp.create('民泊アンケート回答 | Guest Feedback Responses');
  form.setDestination(FormApp.DestinationType.SPREADSHEET, responseSheet.getId());

  Logger.log('===== 作成完了 =====');
  Logger.log('回答用フォームURL: ' + form.getPublishedUrl());
  Logger.log('編集用フォームURL: ' + form.getEditUrl());
  Logger.log('回答集計スプレッドシートURL: ' + responseSheet.getUrl());
}

