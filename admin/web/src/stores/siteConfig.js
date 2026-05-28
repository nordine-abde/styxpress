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
        radius: 'soft',
        customCss: ''
    },
    savedThemes: [],
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
    const previewing = ref(false)
    const previewHtml = ref('')
    const previewUrl = ref('')
    const previewError = ref('')
    const styleGuiding = ref(false)
    const styleGuide = ref(defaultStyleGuide())
    const styleGuideError = ref('')
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

    async function previewSiteConfig(nextConfig) {
        const uiStore = useUiStore()
        previewing.value = true
        previewError.value = ''
        try {
            const payload = await apiRequest('/api/site-config/preview', {
                method: 'POST',
                body: mergeConfig(nextConfig)
            })
            previewHtml.value = payload?.html || ''
            setPreviewUrl(previewHtml.value)
            return payload
        } catch (err) {
            previewError.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            previewing.value = false
        }
    }

    async function loadStyleGuide(nextConfig) {
        const uiStore = useUiStore()
        styleGuiding.value = true
        styleGuideError.value = ''
        try {
            const payload = await apiRequest('/api/site-config/style-guide', {
                method: 'POST',
                body: mergeConfig(nextConfig)
            })
            styleGuide.value = mergeStyleGuide(payload)
            return styleGuide.value
        } catch (err) {
            styleGuideError.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            styleGuiding.value = false
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

    return {
        config,
        loading,
        saving,
        previewing,
        previewHtml,
        previewUrl,
        previewError,
        styleGuiding,
        styleGuide,
        styleGuideError,
        error,
        loadSiteConfig,
        saveSiteConfig,
        previewSiteConfig,
        loadStyleGuide
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
            ...(value.theme || {}),
            customCss: value.theme?.customCss || defaults.theme.customCss
        },
        savedThemes: Array.isArray(value.savedThemes) ? value.savedThemes.map(mergeTheme) : defaults.savedThemes,
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

function mergeTheme(theme = {}) {
    return {
        id: theme.id || '',
        name: theme.name || '',
        palette: theme.palette || defaultSiteConfig.theme.palette,
        font: theme.font || defaultSiteConfig.theme.font,
        layout: theme.layout || defaultSiteConfig.theme.layout,
        radius: theme.radius || defaultSiteConfig.theme.radius,
        customCss: theme.customCss || ''
    }
}

function defaultStyleGuide() {
    return {
        starterCss: '',
        customCssIncluded: false,
        bodyClasses: [],
        selectors: [],
        headerVariants: [],
        footerVariants: [],
        notes: []
    }
}

function mergeStyleGuide(value = {}) {
    return {
        ...defaultStyleGuide(),
        ...value,
        starterCss: value.starterCss || '',
        customCssIncluded: Boolean(value.customCssIncluded),
        bodyClasses: Array.isArray(value.bodyClasses) ? value.bodyClasses : [],
        selectors: Array.isArray(value.selectors) ? value.selectors.map(mergeStyleSelector) : [],
        headerVariants: Array.isArray(value.headerVariants) ? value.headerVariants.map(mergeStyleVariant) : [],
        footerVariants: Array.isArray(value.footerVariants) ? value.footerVariants.map(mergeStyleVariant) : [],
        notes: Array.isArray(value.notes) ? value.notes : []
    }
}

function mergeStyleSelector(selector = {}) {
    return {
        selector: selector.selector || '',
        className: selector.className || '',
        kind: selector.kind || 'base',
        current: Boolean(selector.current),
        description: selector.description || ''
    }
}

function mergeStyleVariant(variant = {}) {
    return {
        variant: variant.variant || '',
        selector: variant.selector || '',
        className: variant.className || '',
        current: Boolean(variant.current),
        rendered: Boolean(variant.rendered)
    }
}
