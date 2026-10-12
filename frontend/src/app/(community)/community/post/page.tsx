import { PostComposer } from '@/features/post';

type CommunityPostPageProps = {
  searchParams: Promise<{ spot?: string | string[] }>;
};

export default async function CommunityPostPage({ searchParams }: CommunityPostPageProps) {
  const { spot } = await searchParams;
  return <PostComposer initialSpotId={typeof spot === 'string' ? spot : undefined} />;
}
