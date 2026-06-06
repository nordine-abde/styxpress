<script setup>
import { computed, onBeforeUnmount, reactive, watch } from 'vue'
import DeployPanel from './DeployPanel.vue'
import SiteLinkEditor from './SiteLinkEditor.vue'
import UiBadge from './ui/UiBadge.vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import UiPanel from './ui/UiPanel.vue'
import UiSwitch from './ui/UiSwitch.vue'
import { cloneDefault, mergeConfig, useSiteConfigStore } from '../stores/siteConfig'
import { useBuildStore } from '../stores/build'

const siteConfigStore = useSiteConfigStore()
const buildStore = useBuildStore()
const form = reactive(cloneDefault())
const saving = computed(() => siteConfigStore.saving || buildStore.rendering)
let savedSnapshot = snapshotConfig(form)
let previewTimer = null

watch(
    () => siteConfigStore.config,
    (value) => {
        const merged = mergeConfig(value)
        Object.assign(form, merged)
        savedSnapshot = snapshotConfig(merged)
        siteConfigStore.setDirty(false)
        schedulePreview(0)
    },
    { deep: true, immediate: true }
)

watch(
    form,
    () => {
        siteConfigStore.setDirty(snapshotConfig(form) !== savedSnapshot)
        schedulePreview()
    },
    { deep: true }
)

async function saveAndRender() {
    const saved = await siteConfigStore.saveSiteConfig(form)
    savedSnapshot = snapshotConfig(saved)
    siteConfigStore.setDirty(false)
    await buildStore.renderSite()
    schedulePreview(0)
}

function schedulePreview(delay = 260) {
    if (previewTimer) {
        window.clearTimeout(previewTimer)
    }
    previewTimer = window.setTimeout(async () => {
        try {
            await siteConfigStore.previewSiteConfig(form)
        } catch {
            // The store already captures and exposes preview errors.
        }
    }, delay)
}

function snapshotConfig(value) {
    return JSON.stringify(mergeConfig(value))
}

onBeforeUnmount(() => {
    if (previewTimer) {
        window.clearTimeout(previewTimer)
    }
})
</script>

<template>
    <div class="site-config-layout">
        <section class="site-config-main">
            <form class="site-config-form" @submit.prevent="saveAndRender">
                <div class="site-config-toolbar">
                    <UiBadge :tone="siteConfigStore.isDirty ? 'warning' : 'success'">
                        {{ siteConfigStore.isDirty ? 'Unsaved changes' : 'Saved' }}
                    </UiBadge>
                    <UiButton tone="primary" type="submit" :busy="saving">
                        Save
                    </UiButton>
                </div>

                <UiPanel title="Site">
                    <div class="field-grid">
                        <div class="two-column">
                            <UiField v-model="form.title" label="Title" placeholder="My Blog" />
                            <UiField v-model="form.description" label="Description" placeholder="Latest posts" />
                        </div>

                        <SiteLinkEditor v-model="form.header.links" title="Header links" />

                        <div class="footer-grid">
                            <UiField
                                v-model="form.footer.text"
                                label="Footer text"
                                placeholder="Local notes and writing."
                            />
                            <UiSwitch
                                v-model="form.footer.showWatermark"
                                label="Show Styx Press watermark"
                                description="Adds Published with Styx Press in the generated footer."
                            />
                        </div>

                        <SiteLinkEditor v-model="form.footer.links" title="Footer links" />
                    </div>
                </UiPanel>
            </form>

            <p v-if="siteConfigStore.error" class="error-text compact-text">
                {{ siteConfigStore.error }}
            </p>
            <p v-if="buildStore.error" class="error-text compact-text">
                {{ buildStore.error }}
            </p>
        </section>

        <aside class="site-config-preview">
            <UiPanel title="Preview">
                <iframe
                    v-if="siteConfigStore.previewUrl"
                    class="preview-frame"
                    title="Site preview"
                    :src="siteConfigStore.previewUrl"
                ></iframe>
                <p v-else class="muted">
                    Preview unavailable.
                </p>
                <p v-if="siteConfigStore.previewError" class="error-text compact-text">
                    {{ siteConfigStore.previewError }}
                </p>
            </UiPanel>

            <UiPanel v-if="buildStore.lastResult" title="Last local build">
                <div class="result">
                    <p v-if="buildStore.lastResult.posts">
                        {{ buildStore.lastResult.posts.length }} posts rendered
                    </p>
                    <p v-if="buildStore.lastResult.post">
                        {{ buildStore.lastResult.post.indexPath }}
                    </p>
                    <p v-if="buildStore.lastResult.site">
                        {{ buildStore.lastResult.site.indexPath }}
                    </p>
                </div>
            </UiPanel>

            <DeployPanel compact />
        </aside>
    </div>
</template>

<style scoped>
.site-config-layout {
    display: grid;
    gap: 1rem;
}

.site-config-main,
.site-config-preview,
.site-config-form,
.footer-grid,
.result {
    display: grid;
    gap: 1rem;
}

.site-config-toolbar {
    display: flex;
    flex-wrap: wrap;
    gap: 0.75rem;
    align-items: center;
    justify-content: flex-end;
}

.preview-frame {
    width: 100%;
    min-height: 34rem;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    background: white;
}

.compact-text {
    margin: 0;
}

.result p {
    margin: 0;
    color: var(--color-muted);
    overflow-wrap: anywhere;
}

@media (min-width: 1180px) {
    .site-config-layout {
        grid-template-columns: minmax(0, 1fr) minmax(22rem, 0.72fr);
        align-items: start;
    }

    .site-config-preview {
        position: sticky;
        top: 1.4rem;
    }
}
</style>
