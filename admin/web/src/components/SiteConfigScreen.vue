<script setup>
import { reactive, watch } from 'vue'
import SiteLinkEditor from './SiteLinkEditor.vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import UiPanel from './ui/UiPanel.vue'
import UiSelect from './ui/UiSelect.vue'
import { cloneDefault, mergeConfig, useSiteConfigStore } from '../stores/siteConfig'

const siteConfigStore = useSiteConfigStore()
const form = reactive(cloneDefault())

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
</script>

<template>
    <form class="site-config" @submit.prevent="save">
        <UiPanel title="Site identity" subtitle="These values render into the public blog, feed, and generated pages.">
            <div class="field-grid">
                <UiField v-model="form.title" label="Site title" />
                <UiField v-model="form.description" label="Site description" />
            </div>
        </UiPanel>

        <UiPanel title="Appearance">
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

        <div class="button-row">
            <UiButton tone="primary" type="submit" :busy="siteConfigStore.saving">
                Save site
            </UiButton>
            <UiButton tone="ghost" :busy="siteConfigStore.loading" @click="siteConfigStore.loadSiteConfig">
                Reload
            </UiButton>
        </div>
    </form>
</template>

<style scoped>
.site-config {
    display: grid;
    gap: 1rem;
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
</style>
