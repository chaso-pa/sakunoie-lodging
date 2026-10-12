// Illustration from the welcome view of docs/pages/sns.html.
export function WelcomeArt({ label }: { label: string }) {
  return (
    <svg viewBox='0 0 420 320' role='img' aria-label={label} preserveAspectRatio='xMidYMid slice'>
      <defs>
        <linearGradient id='welcome-sky' x1='0' y1='0' x2='0' y2='1'>
          <stop offset='0' stopColor='#c6e1d1' />
          <stop offset='1' stopColor='#f5edce' />
        </linearGradient>
        <linearGradient id='welcome-lake' x1='0' y1='0' x2='0' y2='1'>
          <stop offset='0' stopColor='#91c4c1' />
          <stop offset='1' stopColor='#d4e7d8' />
        </linearGradient>
      </defs>
      <rect width='420' height='320' fill='url(#welcome-sky)' />
      <circle cx='318' cy='65' r='29' fill='#f4c979' opacity='.92' />
      <path d='M0 190 L80 85 L145 167 L222 55 L330 183 L376 120 L420 174 L420 320 L0 320Z' fill='#8cae96' />
      <path d='M0 220 L90 140 L152 204 L242 116 L337 224 L386 174 L420 207 L420 320 L0 320Z' fill='#527e68' />
      <path d='M0 230 Q100 208 210 230 T420 222 L420 320 L0 320Z' fill='url(#welcome-lake)' />
      <path d='M0 256 Q90 240 175 257 T420 250' stroke='#f5f3dc' strokeWidth='3' opacity='.65' fill='none' />
      <path d='M0 278 Q90 266 170 280 T420 272' stroke='#f5f3dc' strokeWidth='2' opacity='.5' fill='none' />
      <g transform='translate(52 170)'>
        <path d='M0 48 L38 5 L76 48Z' fill='#f6f2df' />
        <path d='M12 34 L38 5 L64 34Z' fill='#fffdf4' />
        <rect x='9' y='48' width='58' height='39' rx='2' fill='#f4e4c6' />
        <path d='M2 49 L38 27 L75 49Z' fill='#a45f43' />
        <rect x='31' y='62' width='13' height='25' fill='#9b7554' />
        <rect x='16' y='58' width='10' height='10' fill='#91b8ae' />
        <rect x='50' y='58' width='10' height='10' fill='#91b8ae' />
      </g>
      <g fill='#365d48'>
        <path d='M350 224 L372 151 L394 224Z' />
        <path d='M368 248 L390 174 L412 248Z' />
        <path d='M325 248 L346 178 L367 248Z' />
      </g>
      <g transform='translate(238 202) rotate(-8)'>
        <rect x='0' y='0' width='58' height='78' rx='5' fill='#fffdf5' stroke='#e0d8c4' strokeWidth='2' />
        <rect x='7' y='7' width='44' height='35' rx='2' fill='#a8cbb8' />
        <path d='M7 35 L20 20 L29 30 L38 15 L51 35Z' fill='#4d7d63' />
        <path d='M9 51 H48 M9 59 H41 M9 67 H34' stroke='#b7c5ba' strokeWidth='2' strokeLinecap='round' />
      </g>
      <g transform='translate(310 234) rotate(9)'>
        <rect x='0' y='0' width='55' height='73' rx='5' fill='#fffdf5' stroke='#e0d8c4' strokeWidth='2' />
        <rect x='6' y='6' width='43' height='36' rx='2' fill='#e8c6a4' />
        <circle cx='31' cy='19' r='8' fill='#f7e5b2' />
        <path d='M6 38 L18 25 L28 33 L38 22 L49 38Z' fill='#648b6b' />
        <path d='M8 51 H46 M8 59 H38' stroke='#b7c5ba' strokeWidth='2' strokeLinecap='round' />
      </g>
    </svg>
  );
}
