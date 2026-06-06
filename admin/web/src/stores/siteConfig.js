import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiRequest } from '../api/client'
import { useUiStore } from './ui'

export const defaultSiteConfig = {
    title: 'Styxpress',
    description: 'Latest posts',
    header: {
        links: [
            { label: 'Home', href: '/' },
            { label: 'RSS', href: '/feed.xml' }
        ]
    },
    footer: {
        text: '',
        showWatermark: true,
        links: [
            { label: 'RSS', href: '/feed.xml' }
        ]
    }
}

export const useSiteConfigStore = defineStore('siteConfig', () => {
    const config = ref(cloneDefault())
    const loading = ref(false)
    const saving = ref(false)
    const previewing = ref(false)
    const previewHtml = ref('')
    const previewUrl = ref('')
    const previewError = ref('')
    const error = ref('')
    const isDirty = ref(false)
    let previewRequestId = 0

    async function loadSiteConfig() {
        const uiStore = useUiStore()
        loading.value = true
        error.value = ''
        try {
            config.value = mergeConfig(await apiRequest('/api/site-config'))
            isDirty.value = false
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
        } finally {
            loading.value = false
        }
    }

    async function saveSiteConfig(nextConfig) {
        const uiStore = useUiStore()
        saving.value = true
        error.value = ''
        try {
            config.value = mergeConfig(await apiRequest('/api/site-config', {
                method: 'POST',
                body: mergeConfig(nextConfig)
            }))
            isDirty.value = false
            uiStore.setNotice('Site configuration saved.')
            return config.value
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            saving.value = false
        }
    }

    async function previewSiteConfig(nextConfig) {
        const uiStore = useUiStore()
        const requestId = previewRequestId + 1
        previewRequestId = requestId
        previewing.value = true
        previewError.value = ''
        try {
            const payload = await apiRequest('/api/site-config/preview', {
                method: 'POST',
                body: mergeConfig(nextConfig)
            })
            if (requestId !== previewRequestId) {
                return payload
            }
            previewHtml.value = payload?.html || ''
            setPreviewUrl(previewHtml.value)
            return payload
        } catch (err) {
            if (requestId === previewRequestId) {
                previewError.value = err.message
                uiStore.captureError(err)
            }
            throw err
        } finally {
            if (requestId === previewRequestId) {
                previewing.value = false
            }
        }
    }

    function setPreviewUrl(html) {
        if (previewUrl.value) {
            URL.revokeObjectURL(previewUrl.value)
        }
        previewUrl.value = html
            ? URL.createObjectURL(new Blob([html], { type: 'text/html' }))
            : ''
    }

    function reset() {
        config.value = cloneDefault()
        isDirty.value = false
        previewHtml.value = ''
        setPreviewUrl('')
        previewError.value = ''
        error.value = ''
    }

    function setDirty(value) {
        isDirty.value = Boolean(value)
    }

    return {
        config,
        loading,
        saving,
        previewing,
        previewHtml,
        previewUrl,
        previewError,
        error,
        isDirty,
        loadSiteConfig,
        saveSiteConfig,
        previewSiteConfig,
        setDirty,
        reset
    }
})

export function cloneDefault() {
    return JSON.parse(JSON.stringify(defaultSiteConfig))
}

export function mergeConfig(value = {}) {
    const defaults = cloneDefault()
    return {
        ...defaults,
        ...value,
        header: {
            ...defaults.header,
            ...(value.header || {}),
            links: Array.isArray(value.header?.links) ? value.header.links : defaults.header.links
        },
        footer: {
            ...defaults.footer,
            ...(value.footer || {}),
            showWatermark: value.footer?.showWatermark === undefined
                ? defaults.footer.showWatermark
                : Boolean(value.footer.showWatermark),
            links: Array.isArray(value.footer?.links) ? value.footer.links : defaults.footer.links
        }
    }
}
