import { fetchData } from '../../../core/data-fetcher';
import { FetchParams, DataFetcher as DFetcher } from '../../../core/types';
import { getApiClient } from '../../../core/api-client';
import { KanbanSubFetchConfig } from './kanban-types';

export interface KanbanFetchOptions extends KanbanSubFetchConfig {
  element?: HTMLElement;
  params?: FetchParams;
}

export async function kanbanFetch(opts: KanbanFetchOptions) {
  const filterParams: FetchParams = {
    ...opts.params,
    filters: {
      ...(opts.params?.filters || {}),
    },
  };

  if (opts.filterBy && opts.filterValue) {
    filterParams.filters![opts.filterBy] = opts.filterValue;
  }

  let fetcher: DFetcher | undefined;
  if (opts.dataFetcher) {
    fetcher = opts.dataFetcher;
  }

  return fetchData({
    localData: opts.localData,
    fetcher,
    element: opts.element,
    dataSource: opts.dataSource,
    model: opts.model,
    fetchHeaders: opts.fetchHeaders,
    fetchOptions: opts.fetchOptions ? JSON.parse(opts.fetchOptions) : undefined,
    params: filterParams,
  });
}

export async function kanbanCreate(
  model: string | undefined,
  dataSource: string | undefined,
  data: Record<string, unknown>,
): Promise<Record<string, unknown> | null> {
  if (dataSource) {
    const { BcSetup } = await import('../../../core/bc-setup');
    const baseUrl = BcSetup.getBaseUrl();
    let url = dataSource;
    if (url && !url.startsWith('http') && baseUrl) url = baseUrl + url;
    const headers = { 'Content-Type': 'application/json', ...BcSetup.getHeaders() };
    const res = await fetch(url, { method: 'POST', headers, body: JSON.stringify(data) });
    if (!res.ok) throw new Error(`Create failed: ${res.status}`);
    return res.json();
  }
  if (model) {
    const api = getApiClient();
    return api.create(model, data);
  }
  return null;
}

export async function kanbanUpdate(
  model: string | undefined,
  dataSource: string | undefined,
  id: string,
  data: Record<string, unknown>,
): Promise<Record<string, unknown> | null> {
  if (dataSource) {
    const { BcSetup } = await import('../../../core/bc-setup');
    const baseUrl = BcSetup.getBaseUrl();
    let url = dataSource;
    if (url && !url.startsWith('http') && baseUrl) url = baseUrl + url;
    url = `${url.replace(/\/$/, '')}/${id}`;
    const headers = { 'Content-Type': 'application/json', ...BcSetup.getHeaders() };
    const res = await fetch(url, { method: 'PUT', headers, body: JSON.stringify(data) });
    if (!res.ok) throw new Error(`Update failed: ${res.status}`);
    return res.json();
  }
  if (model) {
    const api = getApiClient();
    return api.update(model, id, data);
  }
  return null;
}

export async function kanbanRemove(
  model: string | undefined,
  dataSource: string | undefined,
  id: string,
): Promise<void> {
  if (dataSource) {
    const { BcSetup } = await import('../../../core/bc-setup');
    const baseUrl = BcSetup.getBaseUrl();
    let url = dataSource;
    if (url && !url.startsWith('http') && baseUrl) url = baseUrl + url;
    url = `${url.replace(/\/$/, '')}/${id}`;
    const headers = BcSetup.getHeaders();
    const res = await fetch(url, { method: 'DELETE', headers });
    if (!res.ok) throw new Error(`Delete failed: ${res.status}`);
    return;
  }
  if (model) {
    const api = getApiClient();
    await api.remove(model, id);
  }
}

export async function kanbanUpload(
  dataSource: string | undefined,
  files: File[],
): Promise<unknown[]> {
  if (dataSource) {
    const { BcSetup } = await import('../../../core/bc-setup');
    const baseUrl = BcSetup.getBaseUrl();
    let url = dataSource;
    if (url && !url.startsWith('http') && baseUrl) url = baseUrl + url;
    const form = new FormData();
    for (const f of files) form.append('files', f);
    const headers = BcSetup.getHeaders();
    const res = await fetch(url, { method: 'POST', headers, body: form });
    if (!res.ok) throw new Error(`Upload failed: ${res.status}`);
    const json = await res.json();
    return json.files || json.data || json;
  }
  const api = getApiClient();
  const results: unknown[] = [];
  for (const f of files) {
    const result = await api.upload(f);
    results.push(result);
  }
  return results;
}
