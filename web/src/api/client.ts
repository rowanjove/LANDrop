export interface ApiError {
  status: number
  code?: string
  message: string
}

export class ApiException extends Error {
  status: number
  code?: string

  constructor(error: ApiError) {
    super(error.message)
    this.name = 'ApiException'
    this.status = error.status
    this.code = error.code
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const url = path.startsWith('/') ? path : `/${path}`

  const headers = new Headers(options.headers || {})
  if (!headers.has('Accept')) {
    headers.set('Accept', 'application/json')
  }

  const response = await fetch(url, {
    ...options,
    headers,
  })

  if (!response.ok) {
    let errMessage = `HTTP ${response.status}: ${response.statusText}`
    let errCode: string | undefined

    try {
      const data = await response.json()
      if (data && typeof data === 'object') {
        if (data.error) errMessage = String(data.error)
        if (data.message) errMessage = String(data.message)
        if (data.code) errCode = String(data.code)
      }
    } catch {
      // Body not JSON, fallback to status text
    }

    throw new ApiException({
      status: response.status,
      code: errCode,
      message: errMessage,
    })
  }

  const contentType = response.headers.get('content-type')
  if (contentType && contentType.includes('application/json')) {
    return (await response.json()) as T
  }

  return (await response.text()) as unknown as T
}

export function apiGet<T>(path: string, query?: Record<string, string | number | boolean | undefined>): Promise<T> {
  let url = path
  if (query) {
    const params = new URLSearchParams()
    Object.entries(query).forEach(([k, v]) => {
      if (v !== undefined && v !== '') {
        params.append(k, String(v))
      }
    })
    const qs = params.toString()
    if (qs) {
      url += (url.includes('?') ? '&' : '?') + qs
    }
  }
  return request<T>(url, { method: 'GET' })
}

export function apiPost<T>(path: string, body?: unknown): Promise<T> {
  const isFormData = typeof FormData !== 'undefined' && body instanceof FormData
  return request<T>(path, {
    method: 'POST',
    headers: isFormData ? undefined : { 'Content-Type': 'application/json' },
    body: isFormData ? (body as FormData) : body ? JSON.stringify(body) : undefined,
  })
}

export function apiPut<T>(path: string, body?: unknown): Promise<T> {
  return request<T>(path, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: body ? JSON.stringify(body) : undefined,
  })
}

export function apiDelete<T>(path: string): Promise<T> {
  return request<T>(path, { method: 'DELETE' })
}
