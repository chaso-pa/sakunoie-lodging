import { LogoutButton } from '@/features/auth';
import { ProfilePage } from '@/features/profile';

export default function CommunityProfilePage() {
  // The profile feature may not import the auth feature, so the sign-out control is composed here.
  return <ProfilePage logoutButton={<LogoutButton fullWidth size='md' variant='default' c='#8a4a4a' />} />;
}
