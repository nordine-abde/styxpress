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
const passphrase = ref('')
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
const themeCssReady = computed(() => {
    return Boolean(
        siteConfigStore.styleCss.themeCss &&
        styleCssStructureSignature.value === currentStyleCssStructureSignature.value
    )
})
const blankThemeCssReady = computed(() => {
    return Boolean(
        siteConfigStore.styleCss.blankThemeCss &&
        styleCssStructureSignature.value === currentStyleCssStructureSignature.value
    )
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
    await publishingStore.publishSite(passphrase.value)
}

function saveTheme() {
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
    themeName.value = ''
}

function applyTheme(theme) {
    form.theme.palette = theme.palette
    form.theme.font = theme.font
    form.theme.layout = theme.layout
    form.theme.radius = theme.radius
    form.theme.customCss = theme.customCss || ''
}

function deleteTheme(id) {
    form.savedThemes = form.savedThemes.filter((theme) => theme.id !== id)
}

function useThemeCss() {
    applyCustomCss(siteConfigStore.styleCss.themeCss)
}

function useBlankThemeCss() {
    applyCustomCss(siteConfigStore.styleCss.blankThemeCss)
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
        <UiPanel title="Site identity" subtitle="These values render into the public blog, feed, and generated pages.">
            <div class="field-grid">
                <UiField v-model="form.title" label="Site title" />
                <UiField v-model="form.description" label="Site description" />
            </div>
        </UiPanel>

        <div class="site-workbench">
            <aside class="site-preview-column">
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
            </aside>

            <div class="site-editor-column">
                <UiPanel title="Appearance" subtitle="Theme, typography, layout, and custom CSS.">
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

                        <section class="custom-css-workspace">
                            <header class="section-header">
                                <div>
                                    <h4>CSS</h4>
                                    <p>{{ customCssEmpty ? 'No editable custom CSS yet' : 'Editable custom CSS active' }}</p>
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

                            <p v-if="siteConfigStore.styleCssError" class="error-text compact-text">
                                {{ siteConfigStore.styleCssError }}
                            </p>

                            <section class="css-current">
                                <div class="css-current-header">
                                    <div>
                                        <h5>Current site CSS</h5>
                                        <p>Read-only CSS generated for the current draft.</p>
                                    </div>
                                    <code v-if="bodyClassText" class="body-class-line">{{ bodyClassText }}</code>
                                </div>
                                <textarea
                                    class="css-readonly"
                                    :value="siteConfigStore.styleCss.currentCss"
                                    rows="10"
                                    readonly
                                    placeholder="Current CSS is loading."
                                ></textarea>
                            </section>

                            <div class="css-action-grid">
                                <UiButton tone="ghost" :disabled="!themeCssReady" @click="useThemeCss">
                                    Use theme/base CSS
                                </UiButton>
                                <UiButton tone="primary" :disabled="!blankThemeCssReady" @click="useBlankThemeCss">
                                    Create blank theme CSS
                                </UiButton>
                            </div>

                            <div v-if="customCssEmpty" class="css-empty-mode">
                                <strong>No editable custom CSS</strong>
                                <p>Use the base theme CSS or blank class blocks above to start a saved custom theme.</p>
                            </div>

                            <UiField
                                v-model="form.theme.customCss"
                                label="Editable custom CSS"
                                help="This is saved in theme.customCss and appended after the generated theme CSS."
                                multiline
                                :rows="14"
                                :placeholder="blankCssPlaceholder"
                            />
                        </section>

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

                        <div class="theme-tools">
                            <div class="theme-save">
                                <UiField v-model="themeName" label="Theme name" placeholder="Editorial green" />
                                <UiButton :disabled="!themeName.trim()" @click="saveTheme">
                                    Save theme
                                </UiButton>
                            </div>

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

                <UiPanel title="Header">
                    <div class="field-grid">
                        <UiSelect v-model="form.header.variant" label="Header style" :options="headerOptions" />
                        <div class="two-column">
                            <UiField v-model="form.header.title" label="Header title" />
                            <UiField v-model="form.header.tagline" label="Header tagline" />
                        </div>
                        <SiteLinkEditor v-model="form.header.links" title="Header links" />
                    </div>
                </UiPanel>

                <UiPanel title="Footer">
                    <div class="field-grid">
                        <UiSelect v-model="form.footer.variant" label="Footer style" :options="footerOptions" />
                        <UiField v-model="form.footer.text" label="Footer text" />
                        <SiteLinkEditor v-model="form.footer.links" title="Footer links" />
                    </div>
                </UiPanel>

                <UiPanel title="Publish site" subtitle="Save this form, then render or publish every public page.">
                    <div class="publish-grid">
                        <UiField v-model="passphrase" label="SSH passphrase" type="password" placeholder="Optional" />

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
        </div>
    </form>
</template>

<style scoped>
.site-config {
    display: grid;
    gap: 1rem;
}

.site-workbench,
.site-editor-column,
.site-preview-column,
.appearance-controls,
.custom-css-workspace,
.css-current,
.theme-tools,
.saved-themes,
.guide-dialog-grid,
.guide-dialog-section,
.guide-dialog-section ul {
    display: grid;
    gap: 1rem;
}

.site-editor-column,
.site-preview-column {
    min-width: 0;
}

.appearance-controls,
.custom-css-workspace,
.theme-tools,
.saved-themes {
    gap: 0.85rem;
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

.css-empty-mode {
    display: grid;
    gap: 0.25rem;
    border: 1px dashed var(--color-border);
    border-radius: 8px;
    padding: 0.75rem;
    background: var(--color-surface-muted);
}

.css-empty-mode strong {
    color: var(--color-heading);
}

.css-empty-mode p {
    margin: 0;
    color: var(--color-muted);
}

code {
    font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', monospace;
    font-size: 0.78rem;
    overflow-wrap: anywhere;
}

.css-current {
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 0.75rem;
    background: var(--color-surface-muted);
}

.css-current-header {
    display: grid;
    gap: 0.45rem;
}

.css-current h5,
.guide-dialog-section h4,
.guide-dialog-header h3 {
    margin: 0;
    color: var(--color-heading);
}

.css-current h5,
.guide-dialog-section h4 {
    font-size: 0.9rem;
}

.css-current p,
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

.css-readonly {
    width: 100%;
    min-height: 14rem;
    max-height: 24rem;
    overflow: auto;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 0.72rem 0.8rem;
    background: var(--color-surface);
    color: var(--color-text);
    font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', monospace;
    font-size: 0.82rem;
    line-height: 1.55;
    resize: vertical;
    white-space: pre;
}

.css-action-grid {
    display: grid;
    gap: 0.5rem;
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
    min-height: 18rem;
    overflow: hidden;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    background: white;
}

iframe {
    display: block;
    width: 100%;
    height: 18rem;
    border: 0;
}

.preview-empty {
    display: grid;
    min-height: 18rem;
    place-items: center;
    padding: 1rem;
    text-align: center;
}

.publish-grid {
    display: grid;
    gap: 0.85rem;
    align-items: end;
}

.publish-actions {
    align-items: end;
}

@media (min-width: 700px) {
    .css-action-grid {
        grid-template-columns: repeat(2, minmax(0, 1fr));
    }

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

@media (min-width: 980px) {
    .site-workbench {
        grid-template-areas: "editor preview";
        grid-template-columns: minmax(0, 1fr) minmax(20rem, 0.72fr);
        align-items: start;
    }

    .site-editor-column {
        grid-area: editor;
    }

    .site-preview-column {
        position: sticky;
        top: 1rem;
        grid-area: preview;
    }

    .iframe-wrap,
    .preview-empty {
        min-height: 22rem;
    }

    iframe {
        height: 22rem;
    }
}
</style>
