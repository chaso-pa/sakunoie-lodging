/** Shows a transfer code as XXXX-XXXX-XXXX. */
export const formatTransferCode = (code: string) =>
  code
    .replace(/[\s-]/g, '')
    .toUpperCase()
    .replace(/(.{4})(?=.)/g, '$1-');
