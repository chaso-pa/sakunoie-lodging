import { Avatar } from '@mantine/core';
import classes from '@/components/Community.module.scss';

type MemberAvatarProps = {
  /** Empty or missing means the member has no picture. */
  src?: string | null;
  size?: number;
  alt?: string;
};

export function MemberAvatar({ src, size = 31, alt = '' }: MemberAvatarProps) {
  return (
    <Avatar className={classes.avatar} src={src || null} alt={alt} size={size} radius='50%'>
      <span style={{ fontSize: Math.round(size * 0.48) }} aria-hidden>
        🙂
      </span>
    </Avatar>
  );
}
