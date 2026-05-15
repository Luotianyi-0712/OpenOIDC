export interface ApiResponse<T = unknown> {
  success: boolean
  data?: T
  error?: { code: string; message: string }
  meta?: { total: number; offset: number; limit: number }
}

class ApiClient {
  private base = '/api/v1'

  async request<T>(method: string, path: string, body?: unknown, extraHeaders?: Record<string, string>): Promise<ApiResponse<T>> {
    const opts: RequestInit = {
      method,
      headers: { 'Content-Type': 'application/json', ...extraHeaders },
      credentials: 'same-origin',
    }
    if (body) opts.body = JSON.stringify(body)
    const res = await fetch(this.base + path, opts)
    const data: ApiResponse<T> = await res.json()
    if (!res.ok || !data.success) {
      throw new Error(data.error?.message || `Request failed: ${res.status}`)
    }
    return data
  }

  get<T>(path: string) { return this.request<T>('GET', path) }
  post<T>(path: string, body?: unknown, headers?: Record<string, string>) { return this.request<T>('POST', path, body, headers) }
  put<T>(path: string, body?: unknown) { return this.request<T>('PUT', path, body) }
  del<T>(path: string) { return this.request<T>('DELETE', path) }
}

export const api = new ApiClient()
