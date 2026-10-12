import type { Metadata } from 'next';
import { LodgingPage, TermsContent } from '@/features/lodging';

// Title and description as published in docs/pages/terms.html.
export const metadata: Metadata = {
  title: '宿泊約款｜佐久の家',
  description: '長野県佐久市の家主居住型民泊「佐久の家」の宿泊約款です。'
};

export default function TermsPage() {
  return (
    <LodgingPage title='宿泊約款'>
      <TermsContent />
    </LodgingPage>
  );
}
