import type { Metadata } from 'next';
import { LodgingPage, PricingContent } from '@/features/lodging';

// Title and description as published in docs/pages/pricing.html.
export const metadata: Metadata = {
  title: '宿泊料金表｜佐久の家',
  description: '長野県佐久市の家主居住型民泊「佐久の家」の宿泊料金表です。'
};

export default function PricingPage() {
  return (
    <LodgingPage title='宿泊料金表'>
      <PricingContent />
    </LodgingPage>
  );
}
