import { SpotDetailPage } from '@/features/map';

export default async function CommunitySpotPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  return <SpotDetailPage spotId={id} />;
}
