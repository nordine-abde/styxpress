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
const styleGuideSignature = ref('')

const paletteOptions = [
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
const currentStyleGuideSignature = computed(() => JSON.stringify({
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
const hasStyleGuide = computed(() => {
    const guide = siteConfigStore.styleGuide
    return Boolean(
        guide.starterCss ||
        guide.bodyClasses.length ||
        guide.selectors.length ||
        guide.headerVariants.length ||
        guide.footerVariants.length
    )
})
const hasStarterCss = computed(() => Boolean(siteConfigStore.styleGuide.starterCss))
const previewIsStale = computed(() => {
    return Boolean(siteConfigStore.previewUrl && previewSignature.value !== currentPreviewSignature.value)
})
const styleGuideIsStale = computed(() => {
    return Boolean(hasStyleGuide.value && styleGuideSignature.value !== currentStyleGuideSignature.value)
})
const starterCssReady = computed(() => hasStarterCss.value && !styleGuideIsStale.value)
const starterActionLabel = computed(() => customCssEmpty.value ? 'Use theme starter CSS' : 'Reset to theme starter')
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
const styleGuideStatusLabel = computed(() => {
    if (!hasStyleGuide.value) {
        return 'No CSS guide'
    }
    return styleGuideIsStale.value ? 'CSS guide out of date' : 'CSS guide current'
})
const styleGuideStatusTone = computed(() => {
    if (!hasStyleGuide.value) {
        return 'neutral'
    }
    return styleGuideIsStale.value ? 'warning' : 'success'
})
const selectorGroups = computed(() => {
    const groups = []
    const indexes = new Map()

    for (const selector of siteConfigStore.styleGuide.selectors) {
        const kind = selector.kind || 'base'
        if (!indexes.has(kind)) {
            indexes.set(kind, groups.length)
            groups.push({
                kind,
                label: selectorKindLabel(kind),
                selectors: []
            })
        }
        groups[indexes.get(kind)].selectors.push(selector)
    }

    return groups
})

watch(
    () => siteConfigStore.config,
    (value) => Object.assign(form, mergeConfig(value)),
    { deep: true, immediate: true }
)

onMounted(() => {
    refreshStyleGuide().catch(() => {})
})

async function save() {
    await siteConfigStore.saveSiteConfig(form)
}

async function preview() {
    await siteConfigStore.previewSiteConfig(form)
    previewSignature.value = currentPreviewSignature.value
}

async function refreshStyleGuide() {
    await siteConfigStore.loadStyleGuide(form)
    styleGuideSignature.value = currentStyleGuideSignature.value
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

function useStarterCss() {
    if (!starterCssReady.value) {
        return
    }
    form.theme.customCss = siteConfigStore.styleGuide.starterCss
}

function selectorKindLabel(kind) {
    const labels = {
        variables: 'Variables',
        body: 'Body',
        theme: 'Palette',
        font: 'Font',
        layout: 'Layout',
        radius: 'Corners',
        headerVariant: 'Header',
        footerVariant: 'Footer',
        siteChrome: 'Site chrome',
        content: 'Content',
        base: 'Base'
    }

    return labels[kind] || kind
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
                            <UiBadge :tone="styleGuideStatusTone">
                                {{ styleGuideStatusLabel }}
                            </UiBadge>
                        </div>
                        <div class="button-row">
                            <UiButton tone="primary" :busy="siteConfigStore.previewing" @click="preview">
                                Refresh preview
                            </UiButton>
                            <UiButton tone="ghost" :busy="siteConfigStore.styleGuiding" @click="refreshStyleGuide">
                                Refresh CSS guide
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
                                    <h4>Custom CSS</h4>
                                    <p>{{ customCssEmpty ? 'Empty custom CSS' : 'Custom CSS active' }}</p>
                                </div>
                                <div class="button-row">
                                    <UiButton tone="ghost" :busy="siteConfigStore.styleGuiding" @click="refreshStyleGuide">
                                        Refresh guide
                                    </UiButton>
                                    <UiButton tone="ghost" :disabled="!starterCssReady" @click="useStarterCss">
                                        {{ starterActionLabel }}
                                    </UiButton>
                                </div>
                            </header>

                            <div v-if="customCssEmpty" class="css-empty-mode">
                                <strong>Empty custom CSS</strong>
                                <p>The generated site uses only the selected theme stylesheet.</p>
                            </div>

                            <UiField
                                v-model="form.theme.customCss"
                                label="CSS rules"
                                multiline
                                :rows="12"
                                placeholder=":root { --color-accent: #256f63; }"
                            />

                            <section class="style-guide">
                                <header class="section-header">
                                    <div>
                                        <h4>CSS guide</h4>
                                        <p>Renderer selectors for this draft.</p>
                                    </div>
                                    <UiBadge :tone="styleGuideStatusTone">
                                        {{ styleGuideStatusLabel }}
                                    </UiBadge>
                                </header>

                                <p v-if="siteConfigStore.styleGuideError" class="error-text compact-text">
                                    {{ siteConfigStore.styleGuideError }}
                                </p>

                                <p v-if="!hasStyleGuide && !siteConfigStore.styleGuiding" class="muted compact-text">
                                    CSS guide not loaded.
                                </p>

                                <div v-if="siteConfigStore.styleGuide.bodyClasses.length" class="guide-block">
                                    <h5>Body classes</h5>
                                    <div class="chip-list">
                                        <code
                                            v-for="className in siteConfigStore.styleGuide.bodyClasses"
                                            :key="className"
                                            class="code-chip"
                                        >.{{ className }}</code>
                                    </div>
                                </div>

                                <div
                                    v-if="siteConfigStore.styleGuide.headerVariants.length || siteConfigStore.styleGuide.footerVariants.length"
                                    class="variant-grid"
                                >
                                    <section v-if="siteConfigStore.styleGuide.headerVariants.length" class="guide-block">
                                        <h5>Header variants</h5>
                                        <div class="variant-list">
                                            <article
                                                v-for="variant in siteConfigStore.styleGuide.headerVariants"
                                                :key="variant.variant"
                                                class="variant-row"
                                                :class="{ current: variant.current }"
                                            >
                                                <code>{{ variant.selector }}</code>
                                                <div class="variant-meta">
                                                    <span class="state-chip" :class="{ success: variant.current }">
                                                        {{ variant.current ? 'current' : 'available' }}
                                                    </span>
                                                    <span v-if="!variant.rendered" class="state-chip warning">
                                                        hidden
                                                    </span>
                                                </div>
                                            </article>
                                        </div>
                                    </section>

                                    <section v-if="siteConfigStore.styleGuide.footerVariants.length" class="guide-block">
                                        <h5>Footer variants</h5>
                                        <div class="variant-list">
                                            <article
                                                v-for="variant in siteConfigStore.styleGuide.footerVariants"
                                                :key="variant.variant"
                                                class="variant-row"
                                                :class="{ current: variant.current }"
                                            >
                                                <code>{{ variant.selector }}</code>
                                                <div class="variant-meta">
                                                    <span class="state-chip" :class="{ success: variant.current }">
                                                        {{ variant.current ? 'current' : 'available' }}
                                                    </span>
                                                    <span v-if="!variant.rendered" class="state-chip warning">
                                                        hidden
                                                    </span>
                                                </div>
                                            </article>
                                        </div>
                                    </section>
                                </div>

                                <div v-if="selectorGroups.length" class="selector-scroll">
                                    <section v-for="group in selectorGroups" :key="group.kind" class="selector-group">
                                        <header class="selector-group-header">
                                            <h5>{{ group.label }}</h5>
                                            <span>{{ group.selectors.length }}</span>
                                        </header>
                                        <ul class="selector-list">
                                            <li
                                                v-for="selector in group.selectors"
                                                :key="selector.selector"
                                                class="selector-item"
                                                :class="{ current: selector.current }"
                                            >
                                                <div class="selector-line">
                                                    <code>{{ selector.selector }}</code>
                                                    <span v-if="selector.current" class="state-chip success">
                                                        current
                                                    </span>
                                                </div>
                                                <p v-if="selector.description">
                                                    {{ selector.description }}
                                                </p>
                                            </li>
                                        </ul>
                                    </section>
                                </div>
                            </section>
                        </section>

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
.style-guide,
.theme-tools,
.saved-themes,
.guide-block,
.selector-group,
.selector-list,
.variant-list {
    display: grid;
    gap: 1rem;
}

.site-editor-column,
.site-preview-column {
    min-width: 0;
}

.appearance-controls,
.custom-css-workspace,
.style-guide,
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

.section-header h4,
.guide-block h5,
.selector-group-header h5 {
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

.chip-list,
.variant-meta,
.selector-line {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    align-items: center;
}

code {
    font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', monospace;
    font-size: 0.78rem;
    overflow-wrap: anywhere;
}

.code-chip,
.state-chip {
    display: inline-flex;
    min-height: 1.55rem;
    align-items: center;
    border: 1px solid var(--color-border);
    border-radius: 999px;
    padding: 0 0.5rem;
    background: var(--color-surface);
    color: var(--color-muted);
    font-weight: 800;
}

.state-chip {
    font-size: 0.72rem;
    text-transform: uppercase;
}

.state-chip.success {
    border-color: color-mix(in srgb, var(--color-success) 28%, var(--color-border));
    color: var(--color-success);
}

.state-chip.warning {
    border-color: color-mix(in srgb, var(--color-warning) 35%, var(--color-border));
    color: var(--color-warning);
}

.variant-grid {
    display: grid;
    gap: 0.75rem;
}

.variant-row,
.selector-item {
    display: grid;
    gap: 0.45rem;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 0.65rem;
    background: var(--color-surface);
}

.variant-row.current,
.selector-item.current {
    border-color: color-mix(in srgb, var(--color-success) 28%, var(--color-border));
    background: color-mix(in srgb, var(--color-success) 7%, var(--color-surface));
}

.selector-scroll {
    display: grid;
    max-height: 30rem;
    gap: 0.9rem;
    overflow: auto;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 0.7rem;
    background: var(--color-surface);
}

.selector-group {
    gap: 0.5rem;
}

.selector-group-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.75rem;
}

.selector-group-header span {
    color: var(--color-muted);
    font-size: 0.78rem;
    font-weight: 800;
}

.selector-list {
    margin: 0;
    padding: 0;
    list-style: none;
    gap: 0.45rem;
}

.selector-item p {
    margin: 0;
    color: var(--color-muted);
    font-size: 0.82rem;
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

@media (min-width: 860px) {
    .variant-grid {
        grid-template-columns: repeat(2, minmax(0, 1fr));
    }

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
