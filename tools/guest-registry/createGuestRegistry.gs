/**
 * 佐久の家 宿泊者名簿フォーム 自動作成スクリプト
 * script.google.com で新規プロジェクトに貼り付け、createGuestRegistry を実行してください。
 */

// 宿泊約款・ハウスルールのURL
const TERMS_URL = 'https://app.chaso-pa.com/sakunoie-terms';

function createGuestRegistry() {
  const form = FormApp.create('佐久の家 宿泊者名簿 / Guest Registration');
  form
    .setDescription([
      '旅館業法に基づき、宿泊者全員の記入をお願いしています。',
      'Under the Japanese Hotel Business Act, all guests are required to register.',
      '',
      '※お一人につき1回ずつ送信してください。送信後「別の回答を送信」から続けて入力できます。',
      '  Please submit one form per guest.',
      '※同行者の方は「宿泊代表者と同じ」にチェックすると、住所・電話番号を省略できます。',
      '  Companions may check "Same as the representative" to skip address and phone.',
      '※記入いただいた情報は宿泊者名簿としてのみ使用し、法令に基づき3年間保存します。',
      '  Your information is used only for the guest registry and kept for 3 years as required by law.',
    ].join('\n'))
    .setCollectEmail(false)
    .setAllowResponseEdits(false)
    .setShowLinkToRespondAgain(true)
    .setConfirmationMessage(
      'ご記入ありがとうございました。同行者の方は「別の回答を送信」から続けてご記入ください。\n' +
      'Thank you. Companions, please continue via "Submit another response".'
    );

  // ■ 宿泊日
  form.addSectionHeaderItem().setTitle('宿泊日 / Stay dates');
  form.addDateItem().setTitle('チェックイン日 / Check-in date').setIncludesYear(true).setRequired(true);
  form.addDateItem().setTitle('チェックアウト日 / Check-out date').setIncludesYear(true).setRequired(true);
  form.addTimeItem()
    .setTitle('チェックイン予定時刻 / Estimated arrival time')
    .setHelpText('チェックインは15:00からです。代表者の方はご記入ください。\nCheck-in starts at 3:00 PM. Representatives, please fill in.');

  // ■ あなたについて
  form.addSectionHeaderItem().setTitle('あなたについて / About you');
  form.addCheckboxItem()
    .setTitle('宿泊代表者 / Representative')
    .setChoiceValues(['私が宿泊代表者です / I am the representative of this booking']);
  form.addTextItem()
    .setTitle('宿泊代表者の氏名 / Representative\'s full name')
    .setHelpText('代表者の方はご自身の氏名を記入してください。\nIf you are the representative, enter your own name.')
    .setRequired(true);
  form.addTextItem().setTitle('氏名 / Full name').setRequired(true);
  form.addTextItem()
    .setTitle('フリガナ')
    .setHelpText('日本にお住まいの方のみ、カタカナで記入してください。\nGuests living outside Japan may leave this blank.')
    .setValidation(FormApp.createTextValidation()
      .requireTextMatchesPattern('^[ァ-ヴー・\\s　]+$')
      .setHelpText('カタカナで入力してください')
      .build());

  // ■ 住所・連絡先
  form.addSectionHeaderItem().setTitle('住所・連絡先 / Address & contact');
  form.addCheckboxItem()
    .setTitle('住所・電話番号 / Address & phone')
    .setHelpText('チェックした場合、住所・電話番号は空欄で構いません。\nIf checked, you may leave address and phone blank.')
    .setChoiceValues(['宿泊代表者と同じ / Same as the representative']);
  form.addParagraphTextItem()
    .setTitle('住所 / Address')
    .setHelpText('代表者・「宿泊代表者と同じ」にチェックしていない方は必須です。\nRequired unless "Same as the representative" is checked.');
  form.addTextItem()
    .setTitle('電話番号 / Phone number')
    .setHelpText('国際番号の場合は + から記入してください。\nInclude country code with + if outside Japan.')
    .setValidation(FormApp.createTextValidation()
      .requireTextMatchesPattern('^\\+?[0-9\\-() ]{6,20}$')
      .setHelpText('電話番号の形式で入力してください / Please enter a valid phone number')
      .build());
  form.addTextItem()
    .setTitle('メールアドレス / Email')
    .setValidation(FormApp.createTextValidation()
      .requireTextIsEmail()
      .setHelpText('メールアドレスの形式で入力してください / Please enter a valid email')
      .build());

  // ■ 国籍・旅券
  form.addSectionHeaderItem().setTitle('国籍・旅券 / Nationality & passport');
  form.addMultipleChoiceItem()
    .setTitle('日本国内に住所がありますか？ / Do you have an address in Japan?')
    .setHelpText('「いいえ」の方は国籍・旅券番号を記入し、チェックイン時にパスポートをご提示ください。\nIf "No", fill in nationality and passport number, and present your passport at check-in.')
    .setChoiceValues(['はい / Yes', 'いいえ / No'])
    .setRequired(true);
  form.addTextItem().setTitle('国籍 / Nationality');
  form.addTextItem().setTitle('旅券番号 / Passport number');

  // ■ 同意
  form.addSectionHeaderItem().setTitle('同意 / Consent');
  form.addCheckboxItem()
    .setTitle('宿泊約款・ハウスルール / Terms & house rules')
    .setHelpText(TERMS_URL)
    .setChoiceValues(['同意します / I agree'])
    .setRequired(true);
  form.addCheckboxItem()
    .setTitle('個人情報の取り扱い / Privacy')
    .setHelpText('宿泊者名簿の作成・保存（3年間）および緊急時の連絡にのみ使用します。\nUsed only for the guest registry (kept 3 years) and emergency contact.')
    .setChoiceValues(['同意します / I agree'])
    .setRequired(true);

  // 回答先スプレッドシート
  const ss = SpreadsheetApp.create('佐久の家 宿泊者名簿（回答）');
  form.setDestination(FormApp.DestinationType.SPREADSHEET, ss.getId());
  SpreadsheetApp.flush();

  buildRegistrySheet_(SpreadsheetApp.openById(ss.getId()));

  // 削除処理などから参照するIDを保存
  PropertiesService.getScriptProperties().setProperties({
    FORM_ID: form.getId(),
    SPREADSHEET_ID: ss.getId(),
  });

  Logger.log('回答用URL: ' + form.getPublishedUrl());
  Logger.log('編集用URL: ' + form.getEditUrl());
  Logger.log('スプレッドシート: ' + ss.getUrl());
}

/**
 * 回答シートを参照し、「宿泊代表者と同じ」を代表者の住所・電話に展開した名簿シートを作る。
 * 回答シートの列: A タイムスタンプ / B チェックイン / C チェックアウト / D チェックイン予定時刻 /
 *   E 代表者 / F 代表者氏名 / G 氏名 / H フリガナ / I 代表者と同じ / J 住所 / K 電話 / L メール /
 *   M 国内住所 / N 国籍 / O 旅券番号
 */
function buildRegistrySheet_(ss) {
  const resp = ss.getSheets().find(s => s.getFormUrl());
  const R = `'${resp.getName()}'!`;
  const col = c => `${R}${c}2:${c}`;
  // 全角・半角スペースを除去して氏名を比較
  const norm = x => `REGEXREPLACE(${x}&"","[\\s　]","")`;

  // 同じチェックイン日で、代表者にチェックした人のうち氏名が一致する行から値を引く
  const expand = target => `=MAP(${col('B')},${col('F')},${col('I')},${col(target)},` +
    `LAMBDA(ci,rep,same,v,IF(ci="","",IF(same="",v,IFERROR(INDEX(FILTER(${col(target)},` +
    `${col('B')}=ci,ARRAYFORMULA(${norm(col('G'))})=${norm('rep')},${col('E')}<>""),1),"要確認")))))`;
  const pass = c => `=ARRAYFORMULA(IF(${col('B')}="","",${col(c)}))`;

  // 電話番号・旅券番号の先頭0や + が数値・数式として解釈されないようにする
  resp.getRange('K2:K').setNumberFormat('@');
  resp.getRange('O2:O').setNumberFormat('@');

  const columns = [
    ['タイムスタンプ', pass('A')],
    ['チェックイン', pass('B')],
    ['チェックアウト', pass('C')],
    ['チェックイン予定時刻', pass('D')],
    ['代表者', `=ARRAYFORMULA(IF(${col('B')}="","",IF(${col('E')}<>"","○","")))`],
    ['宿泊代表者氏名', pass('F')],
    ['氏名', pass('G')],
    ['フリガナ', pass('H')],
    ['住所', expand('J')],
    ['電話番号', expand('K')],
    ['メール', pass('L')],
    ['国内住所', pass('M')],
    ['国籍', pass('N')],
    ['旅券番号', pass('O')],
  ];

  const sheet = ss.insertSheet('宿泊者名簿', 0);
  sheet.getRange(1, 1, 1, columns.length).setValues([columns.map(c => c[0])]).setFontWeight('bold');
  sheet.getRange(2, 1, 1, columns.length).setFormulas([columns.map(c => c[1])]);
  // n8n で解析しやすい形式に固定
  sheet.getRange('A2:A').setNumberFormat('yyyy-mm-dd hh:mm:ss');
  sheet.getRange('B2:C').setNumberFormat('yyyy-mm-dd');
  sheet.getRange('D2:D').setNumberFormat('hh:mm');
  sheet.setFrozenRows(1);

  // 既定の空シートを削除
  const blank = ss.getSheetByName('シート1') || ss.getSheetByName('Sheet1');
  if (blank) ss.deleteSheet(blank);
}

/**
 * 既に createGuestRegistry を実行済みの場合に、名簿シートを作り直す。
 * スクリプトプロパティ SPREADSHEET_ID（と FORM_ID）を手動で設定してから実行する。
 */
function rebuildRegistrySheet() {
  const ss = SpreadsheetApp.openById(PropertiesService.getScriptProperties().getProperty('SPREADSHEET_ID'));
  const old = ss.getSheetByName('宿泊者名簿');
  if (old) ss.deleteSheet(old);
  buildRegistrySheet_(ss);
}

// 保存期間（チェックアウト日からの年数）
const RETENTION_YEARS = 3;

/**
 * チェックアウトから3年を過ぎた回答を、フォーム本体と回答シートの両方から削除する。
 */
function purgeExpired() {
  const props = PropertiesService.getScriptProperties();
  const cutoff = new Date();
  cutoff.setHours(0, 0, 0, 0);
  cutoff.setFullYear(cutoff.getFullYear() - RETENTION_YEARS);
  const cutoffYmd = Utilities.formatDate(cutoff, Session.getScriptTimeZone(), 'yyyy-MM-dd');

  // フォーム本体の回答
  const form = FormApp.openById(props.getProperty('FORM_ID'));
  const checkOutItem = form.getItems().find(i => i.getTitle().startsWith('チェックアウト日'));
  let formDeleted = 0;
  form.getResponses().forEach(r => {
    const res = r.getResponseForItem(checkOutItem);
    if (res && res.getResponse() < cutoffYmd) {
      form.deleteResponse(r.getId());
      formDeleted++;
    }
  });

  // 回答シート（C列 = チェックアウト日）。行番号がずれないよう下から削除
  const ss = SpreadsheetApp.openById(props.getProperty('SPREADSHEET_ID'));
  const resp = ss.getSheets().find(s => s.getFormUrl());
  const checkOuts = resp.getRange(2, 3, Math.max(resp.getLastRow() - 1, 1), 1).getValues();
  let rowsDeleted = 0;
  for (let i = checkOuts.length - 1; i >= 0; i--) {
    const v = checkOuts[i][0];
    if (v instanceof Date && v < cutoff) {
      resp.deleteRow(i + 2);
      rowsDeleted++;
    }
  }

  Logger.log(`削除: フォーム ${formDeleted} 件 / シート ${rowsDeleted} 行（基準日 ${cutoffYmd} より前のチェックアウト）`);
}

/** purgeExpired を毎日 4 時台に実行するトリガーを登録する（1回だけ実行） */
function installPurgeTrigger() {
  ScriptApp.getProjectTriggers()
    .filter(t => t.getHandlerFunction() === 'purgeExpired')
    .forEach(t => ScriptApp.deleteTrigger(t));
  ScriptApp.newTrigger('purgeExpired').timeBased().everyDays(1).atHour(4).create();
}
