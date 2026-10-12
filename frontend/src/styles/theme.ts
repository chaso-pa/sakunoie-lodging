import { createTheme, type MantineColorsTuple } from '@mantine/core';

// Scale built around the prototype's --green (#1e6a58, index 7) and --mint (#e9f4ef, index 0).
const sakuGreen: MantineColorsTuple = [
  '#e9f4ef',
  '#d6ebe2',
  '#b3d8c8',
  '#8cc4ad',
  '#68b094',
  '#489b7d',
  '#2f836a',
  '#1e6a58',
  '#175647',
  '#0f4034'
];

export const theme = createTheme({
  colors: { sakuGreen },
  primaryColor: 'sakuGreen',
  primaryShade: 7,
  black: '#20312c',
  fontFamily: 'system-ui, -apple-system, "Noto Sans JP", sans-serif',
  headings: {
    fontFamily: 'system-ui, -apple-system, "Noto Sans JP", sans-serif',
    fontWeight: '800'
  },
  // The prototype uses 11–12px on controls, 18px on cards and 24px on the auth card.
  radius: { xs: '8px', sm: '11px', md: '12px', lg: '18px', xl: '24px' },
  defaultRadius: 'md',
  other: {
    mint: '#e9f4ef',
    ink: '#20312c',
    muted: '#718079',
    line: '#e3eae6',
    paper: '#fbfcfa',
    orange: '#f0a15b'
  }
});
