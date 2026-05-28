<script setup>
import { computed, onMounted, reactive, ref, watch } from 'vue'
import SiteLinkEditor from './SiteLinkEditor.vue'
import UiBadge from './ui/UiBadge.vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import UiPanel from './ui/UiPanel.vue'
import UiSelect from './ui/UiSelect.vue'
import { usePublishingStore } from '../stores/publishing'
import { cloneDefault, mergeConfig, useSiteConfigStore } from '../stores/siteConfig'

const siteConfigStore = useSiteConfigStore()
const publishingStore = usePublishingStore()
const form = reactive(cloneDefault())
const themeName = ref('')
const activeThemeId = ref('')
const activeCustomizerSection = ref('identity')
const previewExpanded = ref(false)
const previewSignature = ref('')
const styleCssCurrentSignature = ref('')
const styleCssStructureSignature = ref('')
const showCssGuide = ref(false)
let styleCssRequestId = 0

const paletteOptions = [
    { value: 'warm', label: 'Warm' },
    { value: 'ink', label: 'Ink' },
    { value: 'sage', label: 'Sage' },
    { value: 'clay', label: 'Clay' },
    { value: 'midnight', label: 'Midnight' }
]
const fontOptions = [
    { value: 'system', label: 'System' },
    { value: 'serif', label: 'Serif' },
    { value: 'mono', label: 'Mono' }
]
const layoutOptions = [
    { value: 'classic', label: 'Classic' },
    { value: 'wide', label: 'Wide' }
]
const radiusOptions = [
    { value: 'soft', label: 'Soft' },
    { value: 'none', label: 'None' }
]
const headerOptions = [
    { value: 'nav', label: 'Navigation' },
    { value: 'centered', label: 'Centered' },
    { value: 'minimal', label: 'Minimal' },
    { value: 'hidden', label: 'Hidden' }
]
const footerOptions = [
    { value: 'simple', label: 'Simple' },
    { value: 'links', label: 'Links' },
    { value: 'hidden', label: 'Hidden' }
]
const blankCssPlaceholder = 'body.theme-warm.font-system.layout-classic.radius-soft {\n}'
const customizerSections = [
    { id: 'identity', label: 'Identity' },
    { id: 'theme', label: 'Theme' },
    { id: 'header', label: 'Header' },
    { id: 'footer', label: 'Footer' },
    { id: 'css', label: 'CSS' },
    { id: 'publish', label: 'Publish' }
]
const cssGuideSections = [
    {
        title: 'Body pattern',
        items: [
            {
                selector: 'body.theme-{palette}.font-{font}.layout-{layout}.radius-{radius}',
                description: 'Full-page target for a saved theme.'
            },
            {
                selector: '.theme-warm, .theme-ink, .theme-sage, .theme-clay, .theme-midnight',
                description: 'Palette classes, usually used to override color variables.'
            },
            {
                selector: '.font-system, .font-serif, .font-mono',
                description: 'Typography classes applied to the body.'
            },
            {
                selector: '.layout-classic, .layout-wide, .radius-soft, .radius-none',
                description: 'Layout width and corner radius switches.'
            }
        ]
    },
    {
        title: 'Theme variables',
        items: [
            {
                selector: ':root',
                description: '--site-bg, --site-surface, --site-text, --site-muted, --site-heading, --site-accent, --site-border, --site-radius, --site-width.'
            },
            {
                selector: '.theme-{palette}',
                description: 'Override palette variables for one palette without replacing all CSS.'
            }
        ]
    },
    {
        title: 'Chrome',
        items: [
            {
                selector: '.site-header, .site-header-inner, .site-branding, .site-title, .site-nav',
                description: 'Header shell, brand, and navigation.'
            },
            {
                selector: '.site-header-nav, .site-header-centered, .site-header-minimal',
                description: 'Header variant classes.'
            },
            {
                selector: '.site-footer, .site-footer-inner, .site-footer-simple, .site-footer-links',
                description: 'Footer shell and footer variants.'
            }
        ]
    },
    {
        title: 'Content',
        items: [
            {
                selector: '.site-main, .post-section, .post-list, .post-card',
                description: 'Homepage layout and post list cards.'
            },
            {
                selector: '.post-header, .post-content, .post-content pre, .post-content code',
                description: 'Post page header, rich text, and code blocks.'
            },
            {
                selector: '.skip-link, a, img',
                description: 'Base accessibility and media styles.'
            }
        ]
    }
]

const currentPreviewSignature = computed(() => {
    const config = mergeConfig(form)
    return JSON.stringify({
        title: config.title,
        description: config.description,
        theme: config.theme,
        header: config.header,
        footer: config.footer
    })
})
const currentStyleCssSignature = computed(() => JSON.stringify({
    theme: {
        palette: form.theme.palette,
        font: form.theme.font,
        layout: form.theme.layout,
        radius: form.theme.radius,
        customCss: form.theme.customCss
    },
    header: {
        variant: form.header.variant
    },
    footer: {
        variant: form.footer.variant
    }
}))
const currentStyleCssStructureSignature = computed(() => JSON.stringify({
    theme: {
        palette: form.theme.palette,
        font: form.theme.font,
        layout: form.theme.layout,
        radius: form.theme.radius
    },
    header: {
        variant: form.header.variant
    },
    footer: {
        variant: form.footer.variant
    }
}))
const customCssEmpty = computed(() => form.theme.customCss.trim() === '')
const hasStyleCss = computed(() => {
    const css = siteConfigStore.styleCss
    return Boolean(
        css.currentCss ||
        css.themeCss ||
        css.blankThemeCss ||
        css.bodyClasses.length
    )
})
const previewIsStale = computed(() => {
    return Boolean(siteConfigStore.previewUrl && previewSignature.value !== currentPreviewSignature.value)
})
const styleCssIsStale = computed(() => {
    return Boolean(
        hasStyleCss.value &&
        (
            styleCssCurrentSignature.value !== currentStyleCssSignature.value ||
            styleCssStructureSignature.value !== currentStyleCssStructureSignature.value
        )
    )
})
const previewStatusLabel = computed(() => {
    if (!siteConfigStore.previewUrl) {
        return 'No preview'
    }
    return previewIsStale.value ? 'Preview out of date' : 'Preview current'
})
const previewStatusTone = computed(() => {
    if (!siteConfigStore.previewUrl) {
        return 'neutral'
    }
    return previewIsStale.value ? 'warning' : 'success'
})
const styleCssStatusLabel = computed(() => {
    if (!hasStyleCss.value) {
        return 'CSS not loaded'
    }
    return styleCssIsStale.value ? 'CSS out of date' : 'CSS current'
})
const styleCssStatusTone = computed(() => {
    if (!hasStyleCss.value) {
        return 'neutral'
    }
    return styleCssIsStale.value ? 'warning' : 'success'
})
const currentCssStateLabel = computed(() => {
    return siteConfigStore.styleCss.customCssIncluded ? 'Includes custom CSS' : 'Theme/base only'
})
const bodyClassText = computed(() => {
    return siteConfigStore.styleCss.bodyClasses.map((className) => `.${className}`).join(' ')
})
const activeSavedTheme = computed(() => {
    return form.savedThemes.find((theme) => theme.id === activeThemeId.value) || null
})
const themeContextLabel = computed(() => {
    if (activeSavedTheme.value) {
        return `Editing custom theme "${activeSavedTheme.value.name}"`
    }
    return 'Editing a predefined theme'
})
const currentCssEditorPlaceholder = computed(() => {
    return siteConfigStore.styleCss.currentCss || blankCssPlaceholder
})
const currentCssReady = computed(() => {
    return Boolean(
        siteConfigStore.styleCss.currentCss &&
        styleCssCurrentSignature.value === currentStyleCssSignature.value
    )
})
const cssEditorHelp = computed(() => {
    if (activeSavedTheme.value) {
        return 'Save changes to update this custom theme, or save as a new theme to branch it.'
    }
    return 'Predefined themes are not overwritten. Save as a new theme when you want to keep this CSS.'
})
const activeCustomizerTitle = computed(() => {
    return customizerSections.find((section) => section.id === activeCustomizerSection.value)?.label || 'Identity'
})

watch(
    () => siteConfigStore.config,
    (value) => Object.assign(form, mergeConfig(value)),
    { deep: true, immediate: true }
)

watch(
    () => siteConfigStore.loading,
    (loading) => {
        if (!loading) {
            refreshStyleCss().catch(() => {})
        }
    }
)

onMounted(() => {
    if (!siteConfigStore.loading) {
        refreshStyleCss().catch(() => {})
    }
})

async function save() {
    await siteConfigStore.saveSiteConfig(form)
    await refreshStyleCss()
}

async function preview() {
    await siteConfigStore.previewSiteConfig(form)
    previewSignature.value = currentPreviewSignature.value
}

async function refreshStyleCss() {
    const requestId = styleCssRequestId + 1
    const currentSignature = currentStyleCssSignature.value
    const structureSignature = currentStyleCssStructureSignature.value
    styleCssRequestId = requestId
    await siteConfigStore.loadStyleCss(form)
    if (requestId === styleCssRequestId) {
        styleCssCurrentSignature.value = currentSignature
        styleCssStructureSignature.value = structureSignature
    }
}

async function renderSite() {
    await siteConfigStore.saveSiteConfig(form)
    await publishingStore.renderSite()
}

async function publishSite() {
    await siteConfigStore.saveSiteConfig(form)
    await publishingStore.publishSite(publishingStore.sshPassphrase)
}

function saveThemeAsNew() {
    const name = themeName.value.trim()
    if (!name) {
        return
    }

    const existing = form.savedThemes.find((theme) => {
        return theme.id === slugify(name) || theme.name.toLowerCase() === name.toLowerCase()
    })
    const id = existing?.id || uniqueThemeId(name)
    const nextTheme = {
        id,
        name,
        palette: form.theme.palette,
        font: form.theme.font,
        layout: form.theme.layout,
        radius: form.theme.radius,
        customCss: form.theme.customCss
    }

    form.savedThemes = existing
        ? form.savedThemes.map((theme) => theme.id === existing.id ? nextTheme : theme)
        : [...form.savedThemes, nextTheme]
    activeThemeId.value = id
    themeName.value = ''
}

function saveActiveTheme() {
    if (!activeSavedTheme.value) {
        return
    }
    const nextTheme = {
        ...activeSavedTheme.value,
        palette: form.theme.palette,
        font: form.theme.font,
        layout: form.theme.layout,
        radius: form.theme.radius,
        customCss: form.theme.customCss
    }
    form.savedThemes = form.savedThemes.map((theme) => theme.id === nextTheme.id ? nextTheme : theme)
}

function applyTheme(theme) {
    activeThemeId.value = theme.id
    form.theme.palette = theme.palette
    form.theme.font = theme.font
    form.theme.layout = theme.layout
    form.theme.radius = theme.radius
    form.theme.customCss = theme.customCss || ''
}

function deleteTheme(id) {
    form.savedThemes = form.savedThemes.filter((theme) => theme.id !== id)
    if (activeThemeId.value === id) {
        activeThemeId.value = ''
    }
}

function useCurrentCss() {
    applyCustomCss(siteConfigStore.styleCss.currentCss)
}

function applyCustomCss(css) {
    if (!css) {
        return
    }
    form.theme.customCss = css
}

function uniqueThemeId(name) {
    const base = slugify(name)
    const usedIds = new Set(form.savedThemes.map((theme) => theme.id))
    if (!usedIds.has(base)) {
        return base
    }

    let index = 2
    while (usedIds.has(`${base}-${index}`)) {
        index += 1
    }
    return `${base}-${index}`
}

function slugify(value) {
    const slug = value
        .toLowerCase()
        .trim()
        .replace(/[^a-z0-9]+/g, '-')
        .replace(/^-+|-+$/g, '')

    return slug || `theme-${Date.now()}`
}
</script>

<template>
    <form class="site-config" @submit.prevent="save">
        <div class="customizer-shell" :class="{ 'preview-expanded': previewExpanded }">
            <aside class="customizer-sidebar">
                <nav class="customizer-nav" aria-label="Style sections">
                    <button
                        v-for="section in customizerSections"
                        :key="section.id"
                        type="button"
                        :class="{ active: activeCustomizerSection === section.id }"
                        @click="activeCustomizerSection = section.id"
                    >
                        {{ section.label }}
                    </button>
                </nav>

                <div class="customizer-controls">
                    <UiPanel
                        v-if="activeCustomizerSection === 'identity'"
                        title="Site identity"
                        subtitle="These values render into the public blog, feed, and generated pages."
                    >
                        <div class="field-grid">
                            <UiField v-model="form.title" label="Site title" />
                            <UiField v-model="form.description" label="Site description" />
                        </div>
                    </UiPanel>

                    <UiPanel
                        v-else-if="activeCustomizerSection === 'theme'"
                        title="Theme"
                        subtitle="Preset controls and saved custom themes."
                    >
                        <div class="appearance-controls">
                            <div class="two-column">
                                <UiSelect v-model="form.theme.palette" label="Palette" :options="paletteOptions" />
                                <UiSelect v-model="form.theme.font" label="Font" :options="fontOptions" />
                                <UiSelect v-model="form.theme.layout" label="Layout" :options="layoutOptions" />
                                <UiSelect v-model="form.theme.radius" label="Corners" :options="radiusOptions" />
                            </div>

                            <div class="theme-preview" :class="`palette-${form.theme.palette}`" aria-hidden="true">
                                <span></span>
                                <span></span>
                                <span></span>
                            </div>

                            <div class="theme-tools">
                                <p class="theme-context">{{ themeContextLabel }}</p>
                                <div class="theme-save">
                                    <UiField v-model="themeName" label="New theme name" placeholder="Editorial green" />
                                    <UiButton :disabled="!themeName.trim()" @click="saveThemeAsNew">
                                        Save as new theme
                                    </UiButton>
                                </div>
                                <UiButton v-if="activeSavedTheme" tone="primary" @click="saveActiveTheme">
                                    Save changes to theme
                                </UiButton>

                                <div v-if="form.savedThemes.length" class="saved-themes">
                                    <article v-for="theme in form.savedThemes" :key="theme.id" class="saved-theme">
                                        <div>
                                            <strong>{{ theme.name }}</strong>
                                            <p>{{ theme.palette }} / {{ theme.font }} / {{ theme.layout }} / {{ theme.radius }}</p>
                                        </div>
                                        <div class="button-row">
                                            <UiButton tone="ghost" @click="applyTheme(theme)">
                                                Apply
                                            </UiButton>
                                            <UiButton tone="danger" @click="deleteTheme(theme.id)">
                                                Delete
                                            </UiButton>
                                        </div>
                                    </article>
                                </div>
                                <p v-else class="muted compact-text">
                                    No saved themes yet.
                                </p>
                            </div>
                        </div>
                    </UiPanel>

                    <UiPanel v-else-if="activeCustomizerSection === 'header'" title="Header">
                        <div class="field-grid">
                            <UiSelect v-model="form.header.variant" label="Header style" :options="headerOptions" />
                            <div class="two-column">
                                <UiField v-model="form.header.title" label="Header title" />
                                <UiField v-model="form.header.tagline" label="Header tagline" />
                            </div>
                            <SiteLinkEditor v-model="form.header.links" title="Header links" />
                        </div>
                    </UiPanel>

                    <UiPanel v-else-if="activeCustomizerSection === 'footer'" title="Footer">
                        <div class="field-grid">
                            <UiSelect v-model="form.footer.variant" label="Footer style" :options="footerOptions" />
                            <UiField v-model="form.footer.text" label="Footer text" />
                            <SiteLinkEditor v-model="form.footer.links" title="Footer links" />
                        </div>
                    </UiPanel>

                    <UiPanel
                        v-else-if="activeCustomizerSection === 'css'"
                        title="CSS"
                        subtitle="One editor for the CSS applied to this draft."
                    >
                        <section class="custom-css-workspace">
                            <header class="section-header">
                                <div>
                                    <h4>{{ customCssEmpty ? 'No custom CSS saved' : 'Custom CSS draft' }}</h4>
                                    <p>{{ cssEditorHelp }}</p>
                                </div>
                                <div class="button-row">
                                    <UiButton tone="ghost" :busy="siteConfigStore.styleCssLoading" @click="refreshStyleCss">
                                        Refresh CSS
                                    </UiButton>
                                    <UiButton tone="ghost" @click="showCssGuide = true">
                                        Open guide
                                    </UiButton>
                                </div>
                            </header>

                            <div class="status-row">
                                <UiBadge :tone="styleCssStatusTone">
                                    {{ styleCssStatusLabel }}
                                </UiBadge>
                                <UiBadge v-if="hasStyleCss">
                                    {{ currentCssStateLabel }}
                                </UiBadge>
                            </div>
                            <code v-if="bodyClassText" class="body-class-line">{{ bodyClassText }}</code>

                            <p v-if="siteConfigStore.styleCssError" class="error-text compact-text">
                                {{ siteConfigStore.styleCssError }}
                            </p>

                            <UiButton tone="ghost" :disabled="!currentCssReady" @click="useCurrentCss">
                                Load current CSS into editor
                            </UiButton>

                            <UiField
                                v-model="form.theme.customCss"
                                label="Theme CSS"
                                :help="themeContextLabel"
                                multiline
                                :rows="22"
                                :placeholder="currentCssEditorPlaceholder"
                            />

                            <div class="css-theme-actions">
                                <p class="theme-context">{{ themeContextLabel }}</p>
                                <div class="theme-save">
                                    <UiField v-model="themeName" label="New theme name" placeholder="Editorial green" />
                                    <UiButton :disabled="!themeName.trim()" @click="saveThemeAsNew">
                                        Save as new theme
                                    </UiButton>
                                </div>
                                <UiButton v-if="activeSavedTheme" tone="primary" @click="saveActiveTheme">
                                    Save changes to theme
                                </UiButton>
                            </div>
                        </section>
                    </UiPanel>

                    <UiPanel
                        v-else
                        title="Publish site"
                        subtitle="Save this form, then render or publish every public page."
                    >
                        <div class="publish-grid">
                            <div class="publish-connection">
                                <UiBadge :tone="publishingStore.sshStatusTone">
                                    {{ publishingStore.sshStatusLabel }}
                                </UiBadge>
                                <p>SSH passphrase and connection test live in Configuration.</p>
                            </div>

                            <div class="button-row publish-actions">
                                <UiButton tone="primary" type="submit" :busy="siteConfigStore.saving">
                                    Save site
                                </UiButton>
                                <UiButton tone="ghost" :busy="siteConfigStore.saving || publishingStore.rendering" @click="renderSite">
                                    Save and render site
                                </UiButton>
                                <UiButton tone="primary" :busy="siteConfigStore.saving || publishingStore.publishing" @click="publishSite">
                                    Save and publish site
                                </UiButton>
                                <UiButton tone="ghost" :busy="siteConfigStore.loading" @click="siteConfigStore.loadSiteConfig">
                                    Reload
                                </UiButton>
                            </div>
                        </div>

                        <div v-if="publishingStore.lastResult" class="result">
                            <strong>Last result</strong>
                            <p v-if="publishingStore.lastResult.posts">
                                {{ publishingStore.lastResult.posts.length }} posts rendered
                            </p>
                            <p v-if="publishingStore.lastResult.site">
                                {{ publishingStore.lastResult.site.indexPath }}
                            </p>
                            <p v-if="publishingStore.lastResult.publish">
                                {{ publishingStore.lastResult.publish.uploadedPaths.length }} uploaded files
                            </p>
                        </div>
                        <p v-if="publishingStore.error" class="error-text compact-text">
                            {{ publishingStore.error }}
                        </p>
                    </UiPanel>
                </div>
            </aside>

            <section class="customizer-preview-column" :aria-label="`${activeCustomizerTitle} preview`">
                <UiPanel title="Draft preview" subtitle="Renders the current draft without saving.">
                    <div class="preview-toolbar">
                        <div class="status-row">
                            <UiBadge :tone="previewStatusTone">
                                {{ previewStatusLabel }}
                            </UiBadge>
                        </div>
                        <div class="button-row">
                            <UiButton tone="primary" :busy="siteConfigStore.previewing" @click="preview">
                                Refresh preview
                            </UiButton>
                            <UiButton tone="ghost" @click="previewExpanded = !previewExpanded">
                                {{ previewExpanded ? 'Close large preview' : 'Expand preview' }}
                            </UiButton>
                        </div>
                    </div>

                    <div class="iframe-wrap">
                        <iframe
                            v-if="siteConfigStore.previewUrl"
                            :src="siteConfigStore.previewUrl"
                            title="Site draft preview"
                        ></iframe>
                        <div v-else class="preview-empty">
                            <p>Preview not rendered yet.</p>
                        </div>
                    </div>
                    <p v-if="siteConfigStore.previewError" class="error-text compact-text">
                        {{ siteConfigStore.previewError }}
                    </p>
                </UiPanel>
            </section>
        </div>

        <div
            v-if="showCssGuide"
            class="guide-overlay"
            role="dialog"
            aria-modal="true"
            aria-labelledby="css-guide-title"
            @click.self="showCssGuide = false"
        >
            <section class="css-guide-dialog">
                <header class="guide-dialog-header">
                    <div>
                        <h3 id="css-guide-title">CSS guide</h3>
                        <p>Static reference for selectors that the public renderer emits.</p>
                    </div>
                    <UiButton tone="ghost" @click="showCssGuide = false">
                        Close
                    </UiButton>
                </header>

                <div class="guide-dialog-grid">
                    <article
                        v-for="section in cssGuideSections"
                        :key="section.title"
                        class="guide-dialog-section"
                    >
                        <h4>{{ section.title }}</h4>
                        <ul>
                            <li v-for="item in section.items" :key="item.selector">
                                <code>{{ item.selector }}</code>
                                <span>{{ item.description }}</span>
                            </li>
                        </ul>
                    </article>
                </div>
            </section>
        </div>
    </form>
</template>

<style scoped>
.site-config {
    display: grid;
    gap: 1rem;
}

.customizer-shell,
.customizer-sidebar,
.customizer-controls,
.customizer-preview-column,
.appearance-controls,
.custom-css-workspace,
.css-theme-actions,
.theme-tools,
.saved-themes,
.guide-dialog-grid,
.guide-dialog-section,
.guide-dialog-section ul {
    display: grid;
    gap: 1rem;
}

.customizer-sidebar,
.customizer-controls,
.customizer-preview-column {
    min-width: 0;
}

.customizer-preview-column {
    order: 2;
}

.customizer-sidebar {
    order: 1;
}

.appearance-controls,
.custom-css-workspace,
.css-theme-actions,
.theme-tools,
.saved-themes {
    gap: 0.85rem;
}

.customizer-nav {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0.4rem;
}

.customizer-nav button {
    min-height: 2.4rem;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 0 0.75rem;
    background: color-mix(in srgb, var(--color-surface) 78%, white);
    color: var(--color-text);
    font-weight: 800;
    text-align: left;
}

.customizer-nav button:hover,
.customizer-nav button.active {
    border-color: color-mix(in srgb, var(--color-accent) 48%, var(--color-border));
    background: color-mix(in srgb, var(--color-accent) 22%, var(--color-surface));
    color: var(--color-heading);
}

.section-header {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    align-items: start;
    justify-content: space-between;
}

.section-header h4 {
    margin: 0;
    color: var(--color-heading);
    font-size: 0.9rem;
}

.section-header p {
    margin: 0.2rem 0 0;
    color: var(--color-muted);
    font-size: 0.86rem;
}

.theme-preview {
    display: flex;
    min-height: 3rem;
    overflow: hidden;
    border: 1px solid var(--color-border);
    border-radius: 8px;
}

.theme-preview span {
    flex: 1;
}

.palette-warm span:nth-child(1) {
    background: #fff3dd;
}

.palette-warm span:nth-child(2) {
    background: #4a220d;
}

.palette-warm span:nth-child(3) {
    background: #f1a5ac;
}

.palette-ink span:nth-child(1) {
    background: #f7f5ef;
}

.palette-ink span:nth-child(2) {
    background: #252a2e;
}

.palette-ink span:nth-child(3) {
    background: #2a6f73;
}

.palette-sage span:nth-child(1) {
    background: #f3f6f1;
}

.palette-sage span:nth-child(2) {
    background: #26342d;
}

.palette-sage span:nth-child(3) {
    background: #2f7158;
}

.palette-clay span:nth-child(1) {
    background: #f7f2ed;
}

.palette-clay span:nth-child(2) {
    background: #342b27;
}

.palette-clay span:nth-child(3) {
    background: #9a4f3d;
}

.palette-midnight span:nth-child(1) {
    background: #101416;
}

.palette-midnight span:nth-child(2) {
    background: #dce3df;
}

.palette-midnight span:nth-child(3) {
    background: #7fc7b1;
}

code {
    font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', monospace;
    font-size: 0.78rem;
    overflow-wrap: anywhere;
}

.guide-dialog-section h4,
.guide-dialog-header h3 {
    margin: 0;
    color: var(--color-heading);
}

.guide-dialog-section h4 {
    font-size: 0.9rem;
}

.guide-dialog-header p,
.guide-dialog-section span {
    margin: 0;
    color: var(--color-muted);
    font-size: 0.84rem;
}

.body-class-line {
    display: block;
    color: var(--color-muted);
}

.guide-overlay {
    position: fixed;
    inset: 0;
    z-index: 30;
    display: grid;
    align-items: start;
    justify-items: center;
    overflow: auto;
    padding: 1rem;
    background: rgb(18 22 25 / 58%);
}

.css-guide-dialog {
    display: grid;
    width: min(48rem, 100%);
    max-height: calc(100vh - 2rem);
    gap: 1rem;
    overflow: auto;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 1rem;
    background: var(--color-surface);
    box-shadow: var(--shadow-soft);
}

.guide-dialog-header {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    align-items: start;
    justify-content: space-between;
}

.guide-dialog-section {
    gap: 0.55rem;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 0.75rem;
    background: var(--color-surface-muted);
}

.guide-dialog-section ul {
    margin: 0;
    padding: 0;
    list-style: none;
    gap: 0.65rem;
}

.guide-dialog-section li {
    display: grid;
    gap: 0.25rem;
}

.guide-dialog-section code {
    color: var(--color-heading);
}

.guide-dialog-section span {
    color: var(--color-muted);
}

.theme-save {
    display: grid;
    gap: 0.6rem;
    align-items: end;
}

.theme-context {
    margin: 0;
    color: var(--color-muted);
    font-size: 0.86rem;
}

.saved-theme,
.result {
    display: grid;
    gap: 0.65rem;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 0.75rem;
    background: var(--color-surface-muted);
}

.saved-theme strong,
.result strong {
    color: var(--color-heading);
}

.saved-theme p,
.result p,
.preview-empty p,
.compact-text {
    margin: 0;
    overflow-wrap: anywhere;
}

.saved-theme p,
.result p,
.preview-empty p {
    color: var(--color-muted);
}

.preview-toolbar {
    display: grid;
    gap: 0.75rem;
}

.iframe-wrap {
    min-height: 28rem;
    overflow: hidden;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    background: white;
}

iframe {
    display: block;
    width: 100%;
    height: 28rem;
    border: 0;
}

.preview-empty {
    display: grid;
    min-height: 28rem;
    place-items: center;
    padding: 1rem;
    text-align: center;
}

.publish-grid {
    display: grid;
    gap: 0.85rem;
    align-items: end;
}

.publish-connection {
    display: grid;
    gap: 0.45rem;
}

.publish-connection p {
    margin: 0;
    color: var(--color-muted);
    font-size: 0.86rem;
}

.publish-actions {
    align-items: end;
}

@media (min-width: 700px) {
    .guide-dialog-grid {
        grid-template-columns: repeat(2, minmax(0, 1fr));
    }
}

@media (min-width: 860px) {
    .theme-save,
    .publish-grid {
        grid-template-columns: minmax(14rem, 0.55fr) minmax(0, 1fr);
    }
}

@media (min-width: 1080px) {
    .customizer-shell {
        grid-template-columns: minmax(22rem, 28rem) minmax(0, 1fr);
        align-items: start;
    }

    .customizer-sidebar {
        position: sticky;
        top: 1rem;
        order: 1;
    }

    .customizer-preview-column {
        order: 2;
    }

    .iframe-wrap,
    .preview-empty {
        min-height: calc(100vh - 13rem);
    }

    iframe {
        height: calc(100vh - 13rem);
    }
}

.preview-expanded .customizer-preview-column {
    position: fixed;
    inset: 1rem;
    z-index: 35;
    display: grid;
    overflow: auto;
}

.preview-expanded .customizer-sidebar {
    visibility: hidden;
}

.preview-expanded .iframe-wrap,
.preview-expanded .preview-empty {
    min-height: calc(100vh - 11rem);
}

.preview-expanded iframe {
    height: calc(100vh - 11rem);
}

@media (min-width: 1280px) {
    .iframe-wrap,
    .preview-empty {
        min-height: calc(100vh - 11rem);
    }

    iframe {
        height: calc(100vh - 11rem);
    }
}
</style>
