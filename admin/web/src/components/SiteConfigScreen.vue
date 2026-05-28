<script setup>
import { reactive, ref, watch } from 'vue'
import SiteLinkEditor from './SiteLinkEditor.vue'
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

watch(
    () => siteConfigStore.config,
    (value) => Object.assign(form, mergeConfig(value)),
    { deep: true, immediate: true }
)

async function save() {
    await siteConfigStore.saveSiteConfig(form)
}

async function preview() {
    await siteConfigStore.previewSiteConfig(form)
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

        <UiPanel title="Appearance" subtitle="Preview the current draft before saving it to disk.">
            <div class="appearance-layout">
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

                    <UiField
                        v-model="form.theme.customCss"
                        label="Custom CSS"
                        multiline
                        :rows="10"
                        placeholder=":root { --color-accent: #256f63; }"
                    />

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

                <div class="preview-pane">
                    <div class="preview-toolbar">
                        <div>
                            <strong>Preview</strong>
                            <p>Uses the current form values.</p>
                        </div>
                        <UiButton tone="primary" :busy="siteConfigStore.previewing" @click="preview">
                            Preview
                        </UiButton>
                    </div>

                    <div class="iframe-wrap">
                        <iframe
                            v-if="siteConfigStore.previewUrl"
                            :src="siteConfigStore.previewUrl"
                            title="Site style preview"
                        ></iframe>
                        <div v-else class="preview-empty">
                            <p>Run preview to render the current style draft.</p>
                        </div>
                    </div>
                    <p v-if="siteConfigStore.previewError" class="error-text compact-text">
                        {{ siteConfigStore.previewError }}
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
    </form>
</template>

<style scoped>
.site-config {
    display: grid;
    gap: 1rem;
}

.appearance-layout {
    display: grid;
    gap: 1rem;
}

.appearance-controls,
.preview-pane,
.theme-tools,
.saved-themes {
    display: grid;
    gap: 0.85rem;
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
.result strong,
.preview-toolbar strong {
    color: var(--color-heading);
}

.saved-theme p,
.result p,
.preview-toolbar p,
.preview-empty p,
.compact-text {
    margin: 0;
    overflow-wrap: anywhere;
}

.saved-theme p,
.result p,
.preview-toolbar p,
.preview-empty p {
    color: var(--color-muted);
}

.preview-toolbar {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    align-items: center;
    justify-content: space-between;
}

.iframe-wrap {
    min-height: 32rem;
    overflow: hidden;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    background: white;
}

iframe {
    display: block;
    width: 100%;
    height: 32rem;
    border: 0;
}

.preview-empty {
    display: grid;
    min-height: 32rem;
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
    .theme-save,
    .publish-grid {
        grid-template-columns: minmax(14rem, 0.55fr) minmax(0, 1fr);
    }
}

@media (min-width: 1180px) {
    .appearance-layout {
        grid-template-columns: minmax(20rem, 0.85fr) minmax(28rem, 1.15fr);
        align-items: start;
    }
}
</style>
