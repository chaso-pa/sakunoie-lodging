import type { Metadata } from 'next';
import { GuideContent, LodgingPage } from '@/features/lodging';

// Title and description as published in docs/pages/guide.html.
export const metadata: Metadata = {
  title: 'ウェルカムガイド｜佐久の家',
  description: '長野県佐久市の家主居住型民泊「佐久の家」の宿泊者向けウェルカムガイドです。',
  robots: { index: false }
};

export default function GuidePage() {
  return (
    <LodgingPage title='ウェルカムガイド'>
      <GuideContent />
    </LodgingPage>
  );
}
