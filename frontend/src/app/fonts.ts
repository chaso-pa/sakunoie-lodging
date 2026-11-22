import { Inter, Noto_Sans_JP } from 'next/font/google';

// Self-hosted by next/font at build time, so the CSP keeps `font-src 'self'`.
export const inter = Inter({ subsets: ['latin'], variable: '--font-inter', display: 'swap' });
export const notoSansJp = Noto_Sans_JP({
  subsets: ['latin'],
  weight: ['400', '500', '700'],
  variable: '--font-noto-sans-jp',
  display: 'swap',
  preload: false
});
