export interface FormResult<T = unknown> {
  ok: boolean
  data?: T
  networkError?: boolean
}

export async function submitFormData<T = unknown>(url: string, options?: RequestInit): Promise<FormResult<T>> {
  let response: Response
  try {
    response = await fetch(new Request(url, { method: "POST" }), options)
  } catch {
    return { ok: false, networkError: true }
  }

  try {
    const data: T = await response.json()
    return { ok: response.ok, data }
  } catch {
    return { ok: response.ok }
  }
}
