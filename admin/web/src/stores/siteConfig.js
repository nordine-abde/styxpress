import { defineStore } from 'pinia'
import { ref } from 'vue'
import { apiRequest } from '../api/client'
import { useUiStore } from './ui'

export const defaultSiteConfig = {
    title: 'Styxpress',
    description: 'Latest posts',
    theme: {
        palette: 'ink',
        font: 'system',
        layout: 'classic',
        radius: 'soft'
    },
    header: {
        variant: 'nav',
        title: '',
        tagline: '',
        links: [
            { label: 'Home', href: '/' },
            { label: 'RSS', href: '/feed.xml' }
        ]
    },
    footer: {
        variant: 'simple',
        text: 'Published with Styxpress',
        links: [
            { label: 'RSS', href: '/feed.xml' }
        ]
    }
}

export const useSiteConfigStore = defineStore('siteConfig', () => {
    const config = ref(cloneDefault())
    const loading = ref(false)
    const saving = ref(false)
    const error = ref('')

    async function loadSiteConfig() {
        const uiStore = useUiStore()
        loading.value = true
        error.value = ''
        try {
            config.value = mergeConfig(await apiRequest('/api/site-config'))
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
            uiStore.setNotice('Site configuration saved.')
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            saving.value = false
        }
    }

    return {
        config,
        loading,
        saving,
        error,
        loadSiteConfig,
        saveSiteConfig
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
        theme: {
            ...defaults.theme,
            ...(value.theme || {})
        },
        header: {
            ...defaults.header,
            ...(value.header || {}),
            links: Array.isArray(value.header?.links) ? value.header.links : defaults.header.links
        },
        footer: {
            ...defaults.footer,
            ...(value.footer || {}),
            links: Array.isArray(value.footer?.links) ? value.footer.links : defaults.footer.links
        }
    }
}
