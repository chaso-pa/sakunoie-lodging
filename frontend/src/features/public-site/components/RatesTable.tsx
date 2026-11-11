// Server component: use the flat Mantine exports (`TableTr`, not `Table.Tr`); compound members of a client reference are undefined in RSC.
import {
  Table,
  TableCaption,
  TableScrollContainer,
  TableTbody,
  TableTd,
  TableTh,
  TableThead,
  TableTr,
  Text
} from '@mantine/core';
import { useTranslations } from 'next-intl';
import { InfoBanner } from '@/components/InfoBanner';
import type { PageFrontMatter } from '@/features/public-site/utils/load-content';

type Props = Pick<PageFrontMatter<'rates'>, 'rates' | 'disclaimer'>;

/** Indicative rates with the "Airbnb determines the actual price" disclaimer (DESIGN §5.5). */
export function RatesTable({ rates, disclaimer }: Props) {
  const t = useTranslations('publicSite');
  return (
    <>
      <InfoBanner tone='info'>
        <Text>{disclaimer}</Text>
      </InfoBanner>
      <TableScrollContainer minWidth={300}>
        <Table withTableBorder verticalSpacing='sm' bg='var(--sakunoie-surface)'>
          <TableCaption>{t('ratesCaption')}</TableCaption>
          <TableThead>
            <TableTr>
              <TableTh scope='col'>{t('ratesItem')}</TableTh>
              <TableTh scope='col'>{t('ratesPrice')}</TableTh>
              <TableTh scope='col'>{t('ratesNote')}</TableTh>
            </TableTr>
          </TableThead>
          <TableTbody>
            {rates.map((r) => (
              <TableTr key={r.label}>
                <TableTh scope='row' fw={500}>
                  {r.label}
                </TableTh>
                <TableTd>{r.price}</TableTd>
                <TableTd>{r.note ?? ''}</TableTd>
              </TableTr>
            ))}
          </TableTbody>
        </Table>
      </TableScrollContainer>
    </>
  );
}
