// Ported from docs/pages/terms.html. This is the guesthouse's published text: keep the Japanese wording,
// headings and numbering identical to the source when editing.
import { Table, TableTbody, TableTd, TableTh, TableTr, Title } from '@mantine/core';
import Link from 'next/link';
import { paths } from '@/config/paths';
import classes from '@/features/lodging/components/LodgingContent.module.scss';

export function TermsContent() {
  return (
    <>
      <Title order={2} className={classes.h2}>
        第1条（適用範囲）
      </Title>
      <p className={classes.clause}>
        <span className={classes.num}>1</span>
        佐久の家（以下「当施設」といいます。）が宿泊客との間で締結する宿泊契約及びこれに関連する契約は、この約款の定めるところによるものとし、この約款に定めのない事項については、法令等（法令又は法令に基づくものをいいます。以下同じ。）又は一般に確立された慣習によるものとします。
      </p>
      <p className={classes.clause}>
        <span className={classes.num}>2</span>
        当施設が、法令等及び慣習に反しない範囲で特約に応じたときは、前項の規定にかかわらず、その特約が優先するものとします。
      </p>
      <p className={classes.clause}>
        <span className={classes.num}>3</span>
        当施設は、ホスト（施設運営者）が同一建物内に居住する家主居住型の簡易宿所です。ゲストルームは宿泊客専用の客室ですが、浴室・洗面所・ランドリールーム等の一部設備はホストと共用となります。
      </p>
      <p className={classes.clause}>
        <span className={classes.num}>4</span>Airbnb
        等の宿泊予約サイト（以下「予約サイト」といいます。）を通じて成立した宿泊契約について、予約サイトの規約（キャンセルポリシー、決済条件等を含みます。）とこの約款の内容が異なる場合は、予約サイトの規約が優先するものとします。
      </p>

      <Title order={2} className={classes.h2}>
        第2条（宿泊契約の申込み）
      </Title>
      <p className={classes.clause}>
        <span className={classes.num}>1</span>
        当施設に宿泊契約の申込みをしようとする者は、次の事項を当施設に申し出ていただきます。
      </p>
      <ol className={classes.paren}>
        <li>宿泊者名及び宿泊人数</li>
        <li>宿泊日及び到着予定時刻</li>
        <li>宿泊料金（原則として別表第1の基本宿泊料によります。）</li>
        <li>その他当施設が必要と認める事項</li>
      </ol>
      <p className={classes.clause}>
        <span className={classes.num}>2</span>
        宿泊客が、宿泊中に前項第2号の宿泊日を超えて宿泊の継続を申し入れた場合、当施設は、その申し出がなされた時点で新たな宿泊契約の申込みがあったものとして処理します。
      </p>

      <Title order={2} className={classes.h2}>
        第3条（宿泊契約の成立等）
      </Title>
      <p className={classes.clause}>
        <span className={classes.num}>1</span>
        宿泊契約は、当施設が前条の申込みを承諾したときに成立するものとします。ただし、当施設が承諾をしなかったことを証明したときは、この限りではありません。
      </p>
      <p className={classes.clause}>
        <span className={classes.num}>2</span>
        予約サイトを通じた申込みについては、予約サイト上で予約が確定したときに宿泊契約が成立したものとし、宿泊料金は予約サイトの定める方法により支払っていただきます。
      </p>
      <p className={classes.clause}>
        <span className={classes.num}>3</span>
        予約サイトを通じない直接の申込みについては、宿泊契約が成立したときに、宿泊期間（3日を超えるときは3日間）の基本宿泊料を限度として当施設が定める申込金を、当施設が指定する日までにお支払いいただきます。
      </p>
      <p className={classes.clause}>
        <span className={classes.num}>4</span>
        申込金は、まず宿泊客が最終的に支払うべき宿泊料金に充当し、第6条及び第18条の規定を適用する事態が生じたときは、違約金に次いで賠償金の順序で充当し、残額があれば、第12条の規定による料金の支払いの際に返還します。
      </p>
      <p className={classes.clause}>
        <span className={classes.num}>5</span>
        第3項の申込金を当施設が指定した日までにお支払いいただけない場合は、宿泊契約はその効力を失うものとします。ただし、申込金の支払期日を指定するに当たり、当施設がその旨を宿泊客に告知した場合に限ります。
      </p>

      <Title order={2} className={classes.h2}>
        第4条（申込金の支払いを要しないこととする特約）
      </Title>
      <p className={classes.clause}>
        <span className={classes.num}>1</span>
        前条第3項の規定にかかわらず、当施設は、契約の成立後同項の申込金の支払いを要しないこととする特約に応じることがあります。
      </p>
      <p className={classes.clause}>
        <span className={classes.num}>2</span>
        宿泊契約の申込みを承諾するに当たり、当施設が前条第3項の申込金の支払いを求めなかった場合及び当該申込金の支払期日を指定しなかった場合は、前項の特約に応じたものとして取り扱います。
      </p>

      <Title order={2} className={classes.h2}>
        第4条の2（施設における感染防止対策への協力の求め）
      </Title>
      <p className={classes.clause}>
        当施設は、宿泊しようとする者に対し、旅館業法（昭和23年法律第138号）第4条の2第1項の規定による協力を求めることができます。
      </p>

      <Title order={2} className={classes.h2}>
        第5条（宿泊契約締結の拒否）
      </Title>
      <p className={classes.clause}>
        <span className={classes.num}>1</span>
        当施設は、次に掲げる場合において、宿泊契約の締結に応じないことがあります。ただし、本項は、当施設が旅館業法第5条に掲げる場合以外の場合に宿泊を拒むことがあることを意味するものではありません。
      </p>
      <ol className={classes.paren}>
        <li>宿泊の申込みが、この約款によらないとき。</li>
        <li>満室により客室の余裕がないとき。</li>
        <li>
          宿泊しようとする者が、宿泊に関し、法令の規定、公の秩序若しくは善良の風俗に反する行為をするおそれがあると認められるとき。
        </li>
        <li>
          宿泊しようとする者が、次のイからハに該当すると認められるとき。
          <ol className={classes.iroha}>
            <li>
              イ　暴力団員による不当な行為の防止等に関する法律（平成3年法律第77号）第2条第2号に規定する暴力団（以下「暴力団」といいます。）、同法第2条第6号に規定する暴力団員（以下「暴力団員」といいます。）、暴力団準構成員又は暴力団関係者その他の反社会的勢力
            </li>
            <li>ロ　暴力団又は暴力団員が事業活動を支配する法人その他の団体であるとき</li>
            <li>ハ　法人でその役員のうちに暴力団員に該当する者があるもの</li>
          </ol>
        </li>
        <li>宿泊しようとする者が、他の宿泊客又は近隣住民に著しい迷惑を及ぼす言動をしたとき。</li>
        <li>
          宿泊しようとする者が、旅館業法第4条の2第1項第2号に規定する特定感染症の患者等（以下「特定感染症の患者等」といいます。）であるとき。
        </li>
        <li>
          宿泊に関し暴力的要求行為が行われ、又は合理的な範囲を超える負担を求められたとき（宿泊しようとする者が障害を理由とする差別の解消の推進に関する法律（平成25年法律第65号。以下「障害者差別解消法」といいます。）第7条第2項又は第8条第2項の規定による社会的障壁の除去を求める場合は除きます。）。
        </li>
        <li>
          宿泊しようとする者が、当施設に対し、その実施に伴う負担が過重であって他の宿泊者に対する宿泊に関するサービスの提供を著しく阻害するおそれのある要求として旅館業法施行規則第5条の6で定めるものを繰り返したとき。
        </li>
        <li>天災、施設の故障、その他やむを得ない事由により宿泊させることができないとき。</li>
        <li>長野県旅館業法施行条例第9条の規定する場合に該当するとき。</li>
      </ol>
      <p className={classes.clause}>
        <span className={classes.num}>2</span>
        客室の定員（2名）を超える人数での宿泊の申込みは、前項第2号の客室の余裕がないときに該当するものとして取り扱います。
      </p>

      <Title order={2} className={classes.h2}>
        第5条の2（宿泊契約締結の拒否の説明）
      </Title>
      <p className={classes.clause}>
        宿泊しようとする者は、当施設に対し、当施設が前条に基づいて宿泊契約の締結に応じない場合、その理由の説明を求めることができます。
      </p>

      <Title order={2} className={classes.h2}>
        第5条の3（乳幼児を伴うご宿泊についてのお願い）
      </Title>
      <p className={classes.clause}>
        当施設は一般住宅の一室をゲストルームとしており、乳幼児の安全に配慮した設備が十分ではないため、満5歳以下のお子様を伴うご宿泊はご遠慮いただくようお願いしております。なお、本条は宿泊契約の締結を拒否する事由を定めるものではありません。
      </p>

      <Title order={2} className={classes.h2}>
        第6条（宿泊客の契約解除権）
      </Title>
      <p className={classes.clause}>
        <span className={classes.num}>1</span>宿泊客は、当施設に申し出て、宿泊契約を解除することができます。
      </p>
      <p className={classes.clause}>
        <span className={classes.num}>2</span>
        当施設は、宿泊客がその責めに帰すべき事由により宿泊契約の全部又は一部を解除した場合（第3条第3項の規定により当施設が申込金の支払期日を指定してその支払いを求めた場合であって、その支払いより前に宿泊客が宿泊契約を解除したときを除きます。）は、別表第2に掲げるところにより、違約金を申し受けます。ただし、当施設が第4条第1項の特約に応じた場合にあっては、その特約に応じるに当たって、宿泊客が宿泊契約を解除したときの違約金支払義務について、当施設が宿泊客に告知したときに限ります。
      </p>
      <p className={classes.clause}>
        <span className={classes.num}>3</span>
        宿泊客が連絡をしないで宿泊日当日に到着しなかった場合の取扱い及び返金については、当該予約に適用される Airbnb
        のキャンセルポリシーに準拠するものとします。
      </p>

      <Title order={2} className={classes.h2}>
        第7条（当施設の契約解除権）
      </Title>
      <p className={classes.clause}>
        <span className={classes.num}>1</span>
        当施設は、次に掲げる場合においては、宿泊契約を解除することがあります。ただし、本項は、当施設が旅館業法第5条に掲げる場合以外の場合に宿泊を拒むことがあることを意味するものではありません。
      </p>
      <ol className={classes.paren}>
        <li>
          宿泊客が宿泊に関し、法令の規定、公の秩序若しくは善良の風俗に反する行為をするおそれがあると認められるとき、又は同行為をしたと認められるとき。
        </li>
        <li>
          宿泊客が次のイからハに該当すると認められるとき。
          <ol className={classes.iroha}>
            <li>イ　暴力団、暴力団員、暴力団準構成員又は暴力団関係者その他の反社会的勢力</li>
            <li>ロ　暴力団又は暴力団員が事業活動を支配する法人その他の団体であるとき</li>
            <li>ハ　法人でその役員のうちに暴力団員に該当する者があるもの</li>
          </ol>
        </li>
        <li>宿泊客が他の宿泊客又は近隣住民に著しい迷惑を及ぼす言動をしたとき。</li>
        <li>宿泊客が特定感染症の患者等であるとき。</li>
        <li>
          宿泊に関し暴力的要求行為が行われ、又は合理的な範囲を超える負担を求められたとき（宿泊客が障害者差別解消法第7条第2項又は第8条第2項の規定による社会的障壁の除去を求める場合は除きます。）。
        </li>
        <li>
          宿泊客が、当施設に対し、その実施に伴う負担が過重であって他の宿泊者に対する宿泊に関するサービスの提供を著しく阻害するおそれのある要求として旅館業法施行規則第5条の6で定めるものを繰り返したとき。
        </li>
        <li>天災等不可抗力に起因する事由により宿泊させることができないとき。</li>
        <li>長野県旅館業法施行条例第9条の規定する場合に該当するとき。</li>
        <li>
          敷地内での喫煙、消防用設備等に対するいたずら、その他当施設が定める利用規則の禁止事項（火災予防上必要なものに限ります。）に従わないとき。
        </li>
      </ol>
      <p className={classes.clause}>
        <span className={classes.num}>2</span>
        当施設が前項の規定に基づいて宿泊契約を解除したときは、宿泊客がいまだ提供を受けていない宿泊サービス等の料金はいただきません。
      </p>

      <Title order={2} className={classes.h2}>
        第7条の2（宿泊契約解除の説明）
      </Title>
      <p className={classes.clause}>
        宿泊客は、当施設に対し、当施設が前条に基づいて宿泊契約を解除した場合、その理由の説明を求めることができます。
      </p>

      <Title order={2} className={classes.h2}>
        第8条（宿泊の登録）
      </Title>
      <p className={classes.clause}>
        <span className={classes.num}>1</span>宿泊客は、宿泊日当日、当施設において、次の事項を登録していただきます。
      </p>
      <ol className={classes.paren}>
        <li>宿泊客の氏名、住所及び連絡先</li>
        <li>
          日本国内に住所を有しない外国人にあっては、国籍及び旅券番号（旅券の呈示及び写しの保存にご協力いただきます。）
        </li>
        <li>その他当施設が必要と認める事項</li>
      </ol>
      <p className={classes.clause}>
        <span className={classes.num}>2</span>
        宿泊客が第12条の料金の支払いを、クレジットカード等通貨に代わり得る方法により行おうとするときは、あらかじめ、前項の登録時にそれらを呈示していただきます（予約サイトで決済済みの場合を除きます。）。
      </p>

      <Title order={2} className={classes.h2}>
        第9条（客室の使用時間）
      </Title>
      <p className={classes.clause}>
        <span className={classes.num}>1</span>
        宿泊客が当施設の客室を使用できる時間は、午後3時から翌朝10時までとします。ただし、連続して宿泊する場合においては、到着日及び出発日を除き、終日使用することができます。
      </p>
      <p className={classes.clause}>
        <span className={classes.num}>2</span>
        当施設は、前項の規定にかかわらず、同項に定める時間外の客室の使用に応じることがあります。この場合には、別途ご案内する追加料金を申し受けることがあります。
      </p>

      <Title order={2} className={classes.h2}>
        第10条（利用規則の遵守）
      </Title>
      <p className={classes.clause}>
        <span className={classes.num}>1</span>
        宿泊客は、当施設内においては、当施設が定めて施設内に掲示又は事前に案内した利用規則に従っていただきます。
      </p>
      <p className={classes.clause}>
        <span className={classes.num}>2</span>前項の利用規則には、次の事項を含みます。
      </p>
      <ol className={classes.paren}>
        <li>
          宿泊客が立ち入ることのできる範囲は、ゲストルーム、LDK、パントリー、ランドリールーム、トイレ、脱衣所及び浴室とし、主寝室及び書斎には立ち入らないこと。
        </li>
        <li>敷地内は全面禁煙（電子たばこ・加熱式たばこを含みます。）とすること。</li>
        <li>
          パーティー、大音量での音楽再生その他近隣の迷惑となる行為をしないこと。特に午後10時から翌朝5時までは静粛時間とし、騒音を伴う行為を禁止します。
        </li>
        <li>宿泊者として登録した者以外の者を施設内に立ち入らせないこと。</li>
      </ol>

      <Title order={2} className={classes.h2}>
        第11条（営業時間）
      </Title>
      <p className={classes.clause}>
        <span className={classes.num}>1</span>当施設の主なサービス時間は次のとおりとします。
      </p>
      <ol className={classes.paren}>
        <li>チェックイン：午後3時から（到着予定時刻を事前にお知らせください）</li>
        <li>チェックアウト：午前10時まで</li>
        <li>門限：なし（午後10時から翌朝5時までは静粛時間とし、夜間の出入りは静かにお願いします）</li>
        <li>
          お問い合わせ：予約サイトのメッセージ機能又は当施設が案内する連絡先にて随時受け付けます（日本語・英語）。
        </li>
      </ol>
      <p className={classes.clause}>
        <span className={classes.num}>2</span>
        前項の時間は、必要やむを得ない場合には臨時に変更することがあります。その場合には、適当な方法をもってお知らせします。
      </p>

      <Title order={2} className={classes.h2}>
        第12条（料金の支払い）
      </Title>
      <p className={classes.clause}>
        <span className={classes.num}>1</span>宿泊客が支払うべき宿泊料金等の内訳は、別表第1に掲げるところによります。
      </p>
      <p className={classes.clause}>
        <span className={classes.num}>2</span>
        前項の宿泊料金等の支払いは、予約サイトを通じた予約にあっては予約サイトの定める方法により、それ以外の予約にあっては通貨又は当施設が認めたクレジットカード等これに代わり得る方法により、宿泊客の到着の際又は当施設が請求した時に行っていただきます。
      </p>
      <p className={classes.clause}>
        <span className={classes.num}>3</span>
        当施設が宿泊客に客室を提供し、使用が可能になったのち、宿泊客が任意に宿泊しなかった場合においても、宿泊料金は申し受けます。
      </p>

      <Title order={2} className={classes.h2}>
        第13条（当施設の責任）
      </Title>
      <p className={classes.clause}>
        当施設は、宿泊契約及びこれに関連する契約の履行に当たり、又はそれらの不履行により宿泊客に損害を与えたときは、その損害を賠償します。ただし、それが当施設の責めに帰すべき事由によるものでないときは、この限りではありません。
      </p>

      <Title order={2} className={classes.h2}>
        第14条（契約した客室の提供ができないときの取扱い）
      </Title>
      <p className={classes.clause}>
        <span className={classes.num}>1</span>
        当施設は、宿泊客に契約した客室を提供できないときは、宿泊客の了解を得て、できる限り同一の条件による他の宿泊施設をあっ旋するものとします。
      </p>
      <p className={classes.clause}>
        <span className={classes.num}>2</span>
        当施設は、前項の規定にかかわらず他の宿泊施設のあっ旋ができないときは、違約金相当額の補償料を宿泊客に支払い、その補償料は損害賠償額に充当します。ただし、客室が提供できないことについて、当施設の責めに帰すべき事由がないときは、補償料を支払いません。
      </p>

      <Title order={2} className={classes.h2}>
        第15条（携帯品等の取扱い）
      </Title>
      <p className={classes.clause}>
        <span className={classes.num}>1</span>
        当施設は、宿泊客の物品、現金及び貴重品のお預かりは行いません。宿泊客が当施設内にお持込みになった物品、現金及び貴重品は、宿泊客ご自身の責任において管理していただきます。
      </p>
      <p className={classes.clause}>
        <span className={classes.num}>2</span>
        前項の物品、現金及び貴重品について、当施設の故意又は過失により滅失、毀損等の損害が生じたときは、当施設は、その損害を賠償します。
      </p>

      <Title order={2} className={classes.h2}>
        第16条（宿泊客の手荷物又は携帯品の保管）
      </Title>
      <p className={classes.clause}>
        <span className={classes.num}>1</span>
        宿泊客の手荷物が、宿泊に先立って当施設に到着した場合は、その到着前に当施設が了解したときに限って責任をもって保管し、宿泊客がチェックインする際お渡しします。
      </p>
      <p className={classes.clause}>
        <span className={classes.num}>2</span>
        宿泊客がチェックアウトしたのち、宿泊客の手荷物又は携帯品が当施設に置き忘れられていた場合において、その所有者が判明したときは、当施設は、当該所有者に連絡をするとともにその指示を求めるものとします。ただし、所有者の指示がない場合又は所有者が判明しないときは、発見日を含め7日間保管し、その後最寄りの警察署に届けます。なお、飲食物等の保管に適さないものは、当施設の判断で処分させていただきます。
      </p>
      <p className={classes.clause}>
        <span className={classes.num}>3</span>
        前2項の場合における宿泊客の手荷物又は携帯品の保管についての当施設の責任は、前条第2項の規定に準じるものとします。
      </p>

      <Title order={2} className={classes.h2}>
        第17条（駐車の責任）
      </Title>
      <p className={classes.clause}>
        宿泊客が当施設の駐車場をご利用になる場合、車両のキーの寄託の如何にかかわらず、当施設は場所をお貸しするものであって、車両の管理責任まで負うものではありません。ただし、駐車場の管理に当たり、当施設の故意又は過失によって損害を与えたときは、その賠償の責めに任じます。
      </p>

      <Title order={2} className={classes.h2}>
        第18条（宿泊客の責任）
      </Title>
      <p className={classes.clause}>
        宿泊客の故意又は過失により当施設が損害を被ったときは、当該宿泊客は当施設に対し、その損害を賠償していただきます。
      </p>

      <Title order={2} className={classes.h2}>
        別表第1　宿泊料金等の内訳（第2条第1項及び第12条第1項関係）
      </Title>
      <Table className={classes.table} withTableBorder withColumnBorders>
        <TableTbody>
          <TableTr>
            <TableTh colSpan={2}>宿泊客が支払うべき総額</TableTh>
          </TableTr>
          <TableTr>
            <TableTd>宿泊料金</TableTd>
            <TableTd>
              ① 基本宿泊料（1室1泊あたりの室料。<Link href={paths.pricing.getHref()}>宿泊料金表</Link>によります。）
            </TableTd>
          </TableTr>
          <TableTr>
            <TableTd>追加料金</TableTd>
            <TableTd>② 第9条第2項に定める時間外使用料その他当施設が別途案内する料金</TableTd>
          </TableTr>
          <TableTr>
            <TableTd>税金</TableTd>
            <TableTd>
              イ 消費税（基本宿泊料に含みます）
              <br />ロ 宿泊税（基本宿泊料に含み、当施設が納付します）
            </TableTd>
          </TableTr>
        </TableTbody>
      </Table>
      <p className={classes.note}>
        備考1　基本宿泊料は1室単位の料金であり、宿泊人数（1〜2名）による変動はありません。
        <br />
        備考2　予約サイトを通じた予約の場合、予約サイトが定めるサービス料・清掃料金等が別途かかることがあります。
      </p>

      <Title order={2} className={classes.h2}>
        別表第2　違約金（第6条第2項関係）
      </Title>
      <Table className={classes.table} withTableBorder withColumnBorders>
        <TableTbody>
          <TableTr>
            <TableTh>契約解除の通知を受けた日</TableTh>
            <TableTh>違約金</TableTh>
          </TableTr>
          <TableTr>
            <TableTd>チェックイン日の5日前まで</TableTd>
            <TableTd>なし（全額返金）</TableTd>
          </TableTr>
          <TableTr>
            <TableTd>チェックイン日の5日前を過ぎてから</TableTd>
            <TableTd>Airbnb のキャンセルポリシーに定める額（一部返金）</TableTd>
          </TableTr>
          <TableTr>
            <TableTd>不泊（連絡なし）</TableTd>
            <TableTd>Airbnb のキャンセルポリシーに定める額</TableTd>
          </TableTr>
        </TableTbody>
      </Table>
      <p className={classes.note}>
        （注）1. 契約日数が短縮した場合は、その短縮日数にかかわりなく、1日分（初日）の違約金を収受します。
        <br />
        2. キャンセルのお手続きは Airbnb
        を通じて行っていただきます。返金額の詳細は、第1条第4項の規定により、当該予約に適用される Airbnb
        のキャンセルポリシーによります。
      </p>

      <p className={`${classes.note} ${classes.spaced}`}>附則　この約款は、2026年（令和8年）10月1日から施行します。</p>
    </>
  );
}
