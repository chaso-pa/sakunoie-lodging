import classes from '@/app/(lodging)/layout.module.scss';
import { LodgingFooter } from '@/features/lodging';

export default function LodgingLayout({ children }: { children: React.ReactNode }) {
  return (
    <div className={classes.root}>
      <div className={classes.column}>
        {children}
        <LodgingFooter />
      </div>
    </div>
  );
}
