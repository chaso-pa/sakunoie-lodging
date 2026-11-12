import '@/features/admin/api/admin-auth';
import { useMutation } from '@tanstack/react-query';
import { filenameFromDisposition } from '@/features/admin/utils/format';
import { ApiError, api } from '@/lib/api-client';

export type ExportFilter = { facilityId: number; from?: string; to?: string };

/** Downloads the CSV through the authenticated API client and saves it via a temporary blob URL. */
export const exportRegistrations = async ({ facilityId, from, to }: ExportFilter) => {
  const params: Record<string, string | number> = { facility_id: facilityId };
  if (from) params.from = from;
  if (to) params.to = to;
  const res = await api.get<Blob>('/admin/guest-registrations/export', {
    params,
    responseType: 'blob',
    // Error bodies arrive as a Blob too; parse the envelope here so the UI gets the stable code.
    validateStatus: () => true
  });
  if (res.status !== 200) {
    let envelope: { code?: string; message?: string; request_id?: string } = {};
    try {
      envelope = JSON.parse(await res.data.text());
    } catch {
      // not JSON
    }
    throw new ApiError(
      res.status,
      envelope.code ?? 'unknown_error',
      envelope.message ?? '',
      [],
      envelope.request_id ?? null
    );
  }
  const filename = filenameFromDisposition(res.headers['content-disposition'], 'registrations.csv');
  const url = URL.createObjectURL(res.data);
  try {
    const a = document.createElement('a');
    a.href = url;
    a.download = filename;
    document.body.appendChild(a);
    a.click();
    a.remove();
  } finally {
    setTimeout(() => URL.revokeObjectURL(url), 1000);
  }
  return filename;
};

export const useExportRegistrations = () => useMutation({ mutationFn: exportRegistrations });
