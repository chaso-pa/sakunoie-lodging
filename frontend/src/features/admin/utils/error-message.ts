import { ApiError } from '@/lib/api-client';

/** Japanese copy for admin API failures, keyed by the stable envelope `code`. */
const messages: Record<string, string> = {
  network_error: '通信できませんでした。接続を確認して、もう一度お試しください。',
  unauthenticated: 'ログインの有効期限が切れました。もう一度ログインしてください。',
  not_authorized: 'この管理画面を利用する権限がありません。',
  forbidden: 'この施設のデータを扱う権限がありません。',
  not_found: '登録が見つかりません。削除されたか、権限のない施設の登録です。',
  precondition_failed: 'ほかの操作で内容が更新されました。最新の内容を読み込み直してから、もう一度お試しください。',
  capacity_exceeded: '施設の定員を超えています。超える場合は「定員超過を許可」を選んでください。',
  validation_error: '入力内容を確認してください。',
  rate_limited: '操作が多すぎます。しばらく待ってからお試しください。'
};

export const adminErrorMessage = (error: unknown) => {
  if (error instanceof ApiError) {
    if (error.status >= 500) return 'サーバーでエラーが発生しました。時間をおいてもう一度お試しください。';
    return messages[error.code] ?? '処理できませんでした。入力内容を確認してください。';
  }
  return '予期しないエラーが発生しました。';
};

/** Firebase sign-in errors (`auth/...`) → Japanese. Never reveals whether an account exists. */
export const signInErrorMessage = (error: unknown) => {
  const code = typeof error === 'object' && error && 'code' in error ? String(error.code) : '';
  switch (code) {
    case 'auth/too-many-requests':
      return 'ログインの試行が多すぎます。しばらく待ってからお試しください。';
    case 'auth/network-request-failed':
      return '認証サーバーに接続できませんでした。接続を確認してください。';
    case 'auth/user-disabled':
      return 'このアカウントは無効になっています。';
    default:
      return 'メールアドレスまたはパスワードが正しくありません。';
  }
};
