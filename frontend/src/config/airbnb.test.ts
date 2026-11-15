import { describe, expect, it } from 'vitest';
import { airbnbEmbedOrigin, airbnbListingId, buildAirbnbEmbedSrc, isAirbnbEmbedOrigin } from '@/config/airbnb';
import { defaultFacility } from '@/config/site';

describe('Airbnb embed', () => {
  it('reads the listing id from the facility Airbnb URL', () => {
    expect(airbnbListingId(defaultFacility.airbnbUrl)).toBe('1668838036170160414');
    expect(airbnbListingId('https://www.airbnb.com/rooms/42')).toBe('42');
    expect(airbnbListingId('https://example.com/')).toBeUndefined();
    expect(airbnbListingId('not a url')).toBeUndefined();
  });

  it('builds the iframe URL the SDK would build (sorted params, frame id as JSON in the fragment)', () => {
    const src = buildAirbnbEmbedSrc({
      listingId: '1668838036170160414',
      pageUrl: 'https://sakunoie.example/en',
      locale: 'en'
    });
    expect(src).toBe(
      'https://www.airbnb.com/embeddable/home?externalPageUrl=https%3A%2F%2Fsakunoie.example%2Fen&hide-price=true&id=1668838036170160414#%7B%22frameId%22%3A1%7D'
    );
  });

  it('uses airbnb.jp for Japanese pages and airbnb.com otherwise; accepts messages from both only', () => {
    expect(airbnbEmbedOrigin('ja')).toBe('https://www.airbnb.jp');
    for (const l of ['en', 'zh-CN', 'ko']) expect(airbnbEmbedOrigin(l)).toBe('https://www.airbnb.com');
    expect(
      buildAirbnbEmbedSrc({ listingId: '1', pageUrl: 'https://sakunoie.example/ja', locale: 'ja' }).startsWith(
        'https://www.airbnb.jp/embeddable/home?'
      )
    ).toBe(true);
    expect(isAirbnbEmbedOrigin('https://www.airbnb.jp')).toBe(true);
    expect(isAirbnbEmbedOrigin('https://www.airbnb.com')).toBe(true);
    expect(isAirbnbEmbedOrigin('https://evil.example')).toBe(false);
    expect(isAirbnbEmbedOrigin('https://www.airbnb.jp.evil.example')).toBe(false);
  });
});
