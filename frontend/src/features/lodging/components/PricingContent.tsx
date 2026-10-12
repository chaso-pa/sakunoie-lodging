// Ported from docs/pages/pricing.html. This is the guesthouse's published text: keep the Japanese wording,
// headings and numbering identical to the source when editing.
import { Table, TableTbody, TableTd, TableTh, TableTr, Title } from '@mantine/core';
import Link from 'next/link';
import { paths } from '@/config/paths';
import classes from '@/features/lodging/components/LodgingContent.module.scss';

export function PricingContent() {
  return (
    <>
      <Title order={2} className={classes.h2}>
        基本宿泊料
      </Title>
      <div className={classes.price}>
        <div className={classes.label}>ゲストルーム（クイーンベッド×1／定員1〜2名）1室1泊あたり</div>
        <div className={classes.amount}>15,000円 〜 28,000円</div>
        <div className={classes.unit}>（消費税・宿泊税込み）</div>
      </div>
      <ul>
        <li>
          料金は<strong>1室単位</strong>です。1名様・2名様とも同一料金です。
        </li>
        <li>
          宿泊日の季節・曜日・需要状況等に応じて、上記の範囲内で変動します。ご予約時に表示される金額が適用されます。
        </li>
        <li>素泊まりの料金です（食事の提供はありません）。</li>
      </ul>

      <Title order={2} className={classes.h2}>
        料金が変動する主な要因
      </Title>
      <ul>
        <li>曜日（平日／金曜・土曜・休前日）</li>
        <li>季節（夏季・紅葉シーズン・年末年始・大型連休など）</li>
        <li>周辺イベントの開催状況、予約時期、連泊日数</li>
      </ul>

      <Title order={2} className={classes.h2}>
        料金に含まれるもの
      </Title>
      <Table className={classes.kvTable} withTableBorder withColumnBorders>
        <TableTbody>
          <TableTr>
            <TableTh>客室</TableTh>
            <TableTd>ゲストルーム（5.2畳）の専用利用、クイーンベッド、プロジェクター</TableTd>
          </TableTr>
          <TableTr>
            <TableTh>共用設備</TableTh>
            <TableTd>LDK、パントリー、ランドリールーム、浴室、脱衣所、トイレ（一部ホストと共用）</TableTd>
          </TableTr>
          <TableTr>
            <TableTh>アメニティ等</TableTh>
            <TableTd>パントリーのコーヒー・軽食、歯磨きセット、化粧水、乳液</TableTd>
          </TableTr>
          <TableTr>
            <TableTh>駐車場</TableTh>
            <TableTd>無料（1台）</TableTd>
          </TableTr>
          <TableTr>
            <TableTh>Wi-Fi</TableTh>
            <TableTd>無料でご利用いただけます</TableTd>
          </TableTr>
        </TableTbody>
      </Table>

      <Title order={2} className={classes.h2}>
        ご利用条件
      </Title>
      <Table className={classes.kvTable} withTableBorder withColumnBorders>
        <TableTbody>
          <TableTr>
            <TableTh>定員</TableTh>
            <TableTd>1〜2名（定員を超えるご宿泊はお受けできません）</TableTd>
          </TableTr>
          <TableTr>
            <TableTh>乳幼児</TableTh>
            <TableTd>
              乳幼児の安全に配慮した設備が十分ではないため、満5歳以下のお子様を伴うご宿泊はご遠慮いただくようお願いしております
            </TableTd>
          </TableTr>
          <TableTr>
            <TableTh>チェックイン</TableTh>
            <TableTd>15:00〜</TableTd>
          </TableTr>
          <TableTr>
            <TableTh>チェックアウト</TableTh>
            <TableTd>〜10:00</TableTd>
          </TableTr>
          <TableTr>
            <TableTh>喫煙</TableTh>
            <TableTd>敷地内全面禁煙</TableTd>
          </TableTr>
          <TableTr>
            <TableTh>門限・静粛時間</TableTh>
            <TableTd>門限なし／22:00〜翌5:00は静粛時間（騒音禁止）</TableTd>
          </TableTr>
        </TableTbody>
      </Table>

      <Title order={2} className={classes.h2}>
        別途かかる料金
      </Title>
      <Table className={classes.kvTable} withTableBorder withColumnBorders>
        <TableTbody>
          <TableTr>
            <TableTh>予約サイトの手数料</TableTh>
            <TableTd>
              Airbnb
              等の予約サイトを通じたご予約の場合、予約サイト所定のサービス料・清掃料金等が加算されることがあります。
            </TableTd>
          </TableTr>
          <TableTr>
            <TableTh>時間外利用</TableTh>
            <TableTd>
              アーリーチェックイン・レイトチェックアウトは、空室状況により対応可能な場合があります。料金は事前にご案内します。
            </TableTd>
          </TableTr>
        </TableTbody>
      </Table>

      <Title order={2} className={classes.h2}>
        キャンセルについて
      </Title>
      <p>キャンセルのお手続きは Airbnb を通じて行っていただき、返金は Airbnb のキャンセルポリシーに準拠します。</p>
      <Table className={classes.kvTable} withTableBorder withColumnBorders>
        <TableTbody>
          <TableTr>
            <TableTh>チェックイン日の5日前まで</TableTh>
            <TableTd>全額返金</TableTd>
          </TableTr>
          <TableTr>
            <TableTh>チェックイン日の5日前を過ぎてから</TableTh>
            <TableTd>Airbnb のキャンセルポリシーに基づき一部返金</TableTd>
          </TableTr>
        </TableTbody>
      </Table>
      <p className={classes.note}>
        詳しくは<Link href={paths.terms.getHref()}>宿泊約款</Link>別表第2をご確認ください。
      </p>

      <p className={`${classes.note} ${classes.spaced}`}>
        2026年10月1日現在の料金です。料金は予告なく変更される場合があります。
      </p>
    </>
  );
}
