import classes from '@/features/lodging/components/LodgingContent.module.scss';

// Ported from the footer shared by docs/pages/{guide,pricing,terms}.html.
export function LodgingFooter() {
  return (
    <footer className={classes.footer} lang='ja'>
      <p>
        佐久の家（家主居住型 簡易宿所）
        <br />
        所在地：長野県佐久市安原1412-1
        <br />
        旅館業法に基づく許可：長野県佐久保健所指令07佐保第11-68号
      </p>
    </footer>
  );
}
