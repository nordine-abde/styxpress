import { useAuthStore } from '../stores/auth'

export class ApiError extends Error {
    constructor(status, code, message) {
        super(message)
        this.name = 'ApiError'
        this.status = status
        this.code = code
    }
}

export async function apiRequest(path, options = {}) {
    const authStore = useAuthStore()
    const headers = new Headers(options.headers || {})

    if (!(options.body instanceof FormData) && options.body !== undefined) {
        headers.set('Content-Type', 'application/json')
    }
    if (authStore.token) {
        headers.set('X-Styxpress-Session', authStore.token)
    }

    const response = await fetch(path, {
        ...options,
        headers,
        body: serializeBody(options.body)
    })

    const text = await response.text()
    const payload = text ? JSON.parse(text) : null

    if (!response.ok) {
        const error = payload?.error || {}
        throw new ApiError(
            response.status,
            error.code || 'request_failed',
            error.message || 'request failed'
        )
    }

    return payload
}

function serializeBody(body) {
    if (body === undefined || body instanceof FormData) {
        return body
    }
    return JSON.stringify(body)
}
