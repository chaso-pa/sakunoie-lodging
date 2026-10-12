/**
 * The API stores areas and weather as the Japanese words the prototype used. These tables map each stored
 * value to a message key under `community.areas` / `community.weather`, so the UI can show it in the
 * viewer's language. A value missing from a table is shown as stored.
 */
export const areaKeys = {
  東信: 'toshin',
  北信: 'hokushin',
  中信: 'chushin',
  南信: 'nanshin',
  未設定: 'unset'
} as const;

/** The four regions a post or a filter can use, in the prototype's order. */
export const areaValues = ['東信', '北信', '中信', '南信'] as const;

export const weatherKeys = {
  晴れ: 'sunny',
  晴れ時々くもり: 'partlyCloudy',
  くもり: 'cloudy',
  霧: 'fog',
  小雨: 'lightRain',
  雨: 'rain',
  雪: 'snow',
  雷雨: 'thunder',
  未設定: 'unset'
} as const;

/** Weather values offered in filters and forms (the prototype's list). */
export const weatherValues = ['晴れ', 'くもり', '小雨', '雨', '雪'] as const;

/** Every weather value the weather endpoint can return, for the composer's select. */
export const allWeatherValues = ['晴れ', '晴れ時々くもり', 'くもり', '霧', '小雨', '雨', '雪', '雷雨'] as const;

const weatherIcons: Record<string, string> = {
  晴れ: '☀️',
  晴れ時々くもり: '🌤️',
  くもり: '☁️',
  霧: '🌫️',
  小雨: '🌦️',
  雨: '🌧️',
  雪: '❄️',
  雷雨: '⛈️'
};

export const getWeatherIcon = (weather: string) => weatherIcons[weather] ?? '🌤️';

export const isKnownArea = (value: string): value is keyof typeof areaKeys => value in areaKeys;
export const isKnownWeather = (value: string): value is keyof typeof weatherKeys => value in weatherKeys;

/** Today's date as YYYY-MM-DD in the device's time zone. */
export const todayIsoDate = () => {
  const now = new Date();
  const pad = (value: number) => String(value).padStart(2, '0');
  return `${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`;
};
