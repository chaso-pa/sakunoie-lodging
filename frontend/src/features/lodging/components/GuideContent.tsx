// Ported from docs/pages/guide.html. This is the guesthouse's published text: keep the Japanese wording,
// headings and numbering identical to the source when editing.
import { Table, TableTbody, TableTd, TableTh, TableTr, Title } from '@mantine/core';
import Link from 'next/link';
import { paths } from '@/config/paths';
import classes from '@/features/lodging/components/LodgingContent.module.scss';

export function GuideContent() {
  return (
    <>
      <p className={classes.lead}>
        ようこそ佐久の家へ。ご宿泊ありがとうございます。
        <br />
        滞在中に知っておいていただきたいことをこのページにまとめました。分からないことがあれば、いつでもお気軽にお声がけください。
      </p>

      <div className={classes.cards}>
        <div className={classes.card}>
          <div className={classes.k}>チェックイン</div>
          <div className={classes.v}>15:00〜</div>
        </div>
        <div className={classes.card}>
          <div className={classes.k}>チェックアウト</div>
          <div className={classes.v}>〜10:00</div>
        </div>
        <div className={classes.card}>
          <div className={classes.k}>静粛時間</div>
          <div className={classes.v}>22:00〜5:00</div>
        </div>
        <div className={classes.card}>
          <div className={classes.k}>ホストの入浴</div>
          <div className={classes.v}>20:00〜21:00</div>
        </div>
      </div>

      <nav className={classes.toc} aria-label='目次'>
        <ol>
          <li>
            <a href='#about'>家主居住型について</a>
          </li>
          <li>
            <a href='#areas'>ご利用いただける場所</a>
          </li>
          <li>
            <a href='#living'>リビング</a>
          </li>
          <li>
            <a href='#pantry'>パントリー・冷蔵庫</a>
          </li>
          <li>
            <a href='#bath'>お風呂・洗面台・トイレ</a>
          </li>
          <li>
            <a href='#laundry'>ランドリー</a>
          </li>
          <li>
            <a href='#room'>お部屋・プロジェクター</a>
          </li>
          <li>
            <a href='#wifi'>Wi-Fi</a>
          </li>
          <li>
            <a href='#aircon'>エアコン</a>
          </li>
          <li>
            <a href='#garbage'>ゴミの分別</a>
          </li>
          <li>
            <a href='#rules'>ハウスルール</a>
          </li>
          <li>
            <a href='#key'>ルームキー・チェックアウト</a>
          </li>
          <li>
            <a href='#nearby'>周辺情報</a>
          </li>
          <li>
            <a href='#contact'>お問い合わせ</a>
          </li>
        </ol>
      </nav>

      <Title order={2} id='about' className={classes.h2}>
        家主居住型の民泊について
      </Title>
      <p>
        佐久の家は、ホストが同じ建物内に暮らす家主居住型の民泊です。ゲストの皆さまには専用のお部屋をご用意していますが、リビング・お風呂・ランドリー・トイレなど一部のスペースはホストと共用になります。
      </p>
      <p>一般住宅のため、生活音が聞こえる場合があります。あらかじめご了承ください。</p>

      <Title order={2} id='areas' className={classes.h2}>
        ご利用いただける場所
      </Title>
      <Table className={classes.kvTable} withTableBorder withColumnBorders>
        <TableTbody>
          <TableTr>
            <TableTh>ご利用いただける場所</TableTh>
            <TableTd>
              ゲストルーム、リビング（LDK）、パントリー、ランドリールーム、お風呂、脱衣所・洗面台、トイレ
            </TableTd>
          </TableTr>
          <TableTr>
            <TableTh>立ち入りをご遠慮いただく場所</TableTh>
            <TableTd>主寝室、書斎</TableTd>
          </TableTr>
        </TableTbody>
      </Table>

      <Title order={2} id='living' className={classes.h2}>
        リビング
      </Title>
      <p>リビングは共用スペースです。ソファでくつろいだり、お食事をとったり、滞在中はご自由にお使いください。</p>

      <Title order={2} id='pantry' className={classes.h2}>
        パントリー・冷蔵庫
      </Title>
      <p>パントリーにあるものは、滞在中ご自由にお使いいただけます。</p>
      <ul>
        <li>ウォーターサーバー（冷水・温水）</li>
        <li>コーヒーなどの飲み物</li>
        <li>お菓子・軽食</li>
        <li>
          ボードゲーム ──
          よかったら遊んでみてください。「あと1人足りない！」というときは、タイミングが合えばホストも喜んで参加します。
        </li>
      </ul>
      <Title order={3} className={classes.h3}>
        冷蔵庫
      </Title>
      <p>
        冷蔵庫には<strong>ゲスト用のスペース</strong>があります。購入した飲み物や食べ物などはこちらに入れてください。
      </p>

      <Title order={2} id='bath' className={classes.h2}>
        お風呂・洗面台・トイレ
      </Title>
      <p>お風呂・脱衣所・洗面台・トイレは共用スペースです。</p>
      <div className={classes.callout}>
        <strong>20:00〜21:00 はホストが入浴します。</strong>お風呂はその前後の時間帯でご利用ください。
      </div>
      <ul>
        <li>タオルは脱衣所にご用意しています。</li>
        <li>ドライヤー、アメニティ（歯磨きセット・化粧水・乳液など）は洗面台にあります。ご自由にお使いください。</li>
      </ul>
      <p className={classes.note}>
        佐久市内には温泉や日帰り入浴施設もあります。お出かけの際は、こちらのタオルを持って行っていただいて構いません。
      </p>

      <Title order={2} id='laundry' className={classes.h2}>
        ランドリー
      </Title>
      <p>ランドリールームの洗濯機はご自由にお使いいただけます。</p>
      <div className={classes.callout}>
        洗濯から乾燥まで<strong>約4〜6時間</strong>かかります。お急ぎの場合はご注意ください。
      </div>
      <p className={classes.note}>
        お近くのスーパー「ナナーズ」にもコインランドリーがあります。短時間で済ませたいときに便利です。
      </p>

      <Title order={2} id='room' className={classes.h2}>
        お部屋・プロジェクター
      </Title>
      <p>ゲストルームは、滞在中の皆さま専用のスペースです。どうぞゆっくりお過ごしください。</p>
      <Title order={3} className={classes.h3}>
        プロジェクターの使い方
      </Title>
      <p>映画や動画などを自由にお楽しみください。</p>
      <ol>
        <li>
          <mark className={classes.todo}>電源の入れ方</mark>
        </li>
        <li>
          <mark className={classes.todo}>スマートフォン・PC との接続方法</mark>
        </li>
        <li>
          <mark className={classes.todo}>音量・入力切替</mark>
        </li>
      </ol>
      <p className={classes.note}>接続方法などが分からないときは、遠慮なくお声がけください。</p>

      <Title order={2} id='wifi' className={classes.h2}>
        Wi-Fi
      </Title>
      <p>Wi-Fi は無料でご利用いただけます。</p>
      <p>
        接続用の <strong>QRコードをゲストルーム内に掲示</strong>
        しています。スマートフォンのカメラで読み込むと接続できます。ネットワーク名とパスワードも同じ場所に記載しています。
      </p>
      <p className={classes.note}>接続できない場合は、いつでもお声がけください。</p>

      <Title order={2} id='aircon' className={classes.h2}>
        エアコン
      </Title>
      <p>お部屋のリモコンで冷房・暖房などを操作できます。暑いとき・寒いときは遠慮なくお使いください。</p>
      <ol>
        <li>
          <mark className={classes.todo}>運転／停止</mark>
        </li>
        <li>
          <mark className={classes.todo}>冷房・暖房の切替</mark>
        </li>
        <li>
          <mark className={classes.todo}>温度設定</mark>
        </li>
      </ol>

      <Title order={2} id='garbage' className={classes.h2}>
        ゴミの分別
      </Title>
      <p>ゴミは次の3つに分けていただけると助かります。</p>
      <Table className={classes.kvTable} withTableBorder withColumnBorders>
        <TableTbody>
          <TableTr>
            <TableTh>燃えるゴミ</TableTh>
            <TableTd>
              生ゴミ、紙くず、ティッシュ、プラスチック容器など<mark className={classes.todo}>（区分要確認）</mark>
            </TableTd>
          </TableTr>
          <TableTr>
            <TableTh>ペットボトル</TableTh>
            <TableTd>キャップとラベルを外していただけると助かります</TableTd>
          </TableTr>
          <TableTr>
            <TableTh>ビン・カン</TableTh>
            <TableTd>中を軽くすすいでください</TableTd>
          </TableTr>
        </TableTbody>
      </Table>
      <p className={classes.note}>
        ゴミ箱の場所：<mark className={classes.todo}>○○</mark>
      </p>

      <Title order={2} id='rules' className={classes.h2}>
        ハウスルール
      </Title>
      <ul>
        <li>
          <strong>禁煙</strong>：敷地内は全面禁煙です（電子たばこ・加熱式たばこを含みます）。
        </li>
        <li>
          <strong>静粛時間</strong>
          ：22:00〜翌5:00は、大きな音を立てないようお願いします。夜間の出入りも静かにお願いします。
        </li>
        <li>パーティーや大音量での音楽再生はご遠慮ください。</li>
        <li>ご予約者以外の方を建物内に招き入れることはご遠慮ください。</li>
        <li>門限はありません。</li>
      </ul>
      <p className={classes.note}>
        詳しくは<Link href={paths.terms.getHref()}>宿泊約款</Link>をご覧ください。
      </p>

      <Title order={2} id='key' className={classes.h2}>
        ルームキー・チェックアウト
      </Title>
      <p>チェックイン時にルームキーをお渡しします。外出の際は、お忘れなくお持ちください。</p>
      <Title order={3} className={classes.h3}>
        チェックアウト（10:00まで）
      </Title>
      <ul>
        <li>
          ルームキーをご返却ください（返却方法：<mark className={classes.todo}>ホストに手渡し／○○に置く</mark>）。
        </li>
        <li>
          ゴミは分別して<mark className={classes.todo}>○○</mark>にお願いします。
        </li>
        <li>お忘れ物がないかご確認ください。</li>
      </ul>

      <Title order={2} id='nearby' className={classes.h2}>
        周辺情報
      </Title>
      <ul>
        <li>
          <strong>ナナーズ</strong>（スーパー）── お買い物に。コインランドリーもあります。
        </li>
        <li>
          <strong>温泉・日帰り入浴施設</strong> ──
          佐久市内にいくつかあります。おすすめはホストにお気軽にお尋ねください。
        </li>
        <li>
          <strong>軽井沢</strong> ── 車で約20分。小諸・上田エリアへのお出かけにも便利です。
        </li>
      </ul>
      <p className={classes.note}>駐車場はゲスト用に1台分ご用意しています（無料）。</p>

      <Title order={2} id='contact' className={classes.h2}>
        お問い合わせ
      </Title>
      <p>
        ホストは同じ建物内にいますので、お困りのときはいつでもお声がけください。
        <br />
        Airbnb のメッセージ機能からもご連絡いただけます。英語でのやりとりも可能です。
      </p>
      <p>佐久での滞在を、どうぞゆっくりお楽しみください。</p>
    </>
  );
}
