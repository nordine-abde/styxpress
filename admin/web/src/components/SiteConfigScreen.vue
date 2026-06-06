<script setup>
import { reactive, watch } from 'vue'
import SiteLinkEditor from './SiteLinkEditor.vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import UiPanel from './ui/UiPanel.vue'
import UiSwitch from './ui/UiSwitch.vue'
import { cloneDefault, mergeConfig, useSiteConfigStore } from '../stores/siteConfig'
import { useBuildStore } from '../stores/build'

const siteConfigStore = useSiteConfigStore()
const buildStore = useBuildStore()
const form = reactive(cloneDefault())

watch(
    () => siteConfigStore.config,
    (value) => {
        Object.assign(form, mergeConfig(value))
    },
    { deep: true, immediate: true }
)

async function save() {
    await siteConfigStore.saveSiteConfig(form)
}

async function reload() {
    await siteConfigStore.loadSiteConfig()
}

async function preview() {
    await siteConfigStore.previewSiteConfig(form)
}

async function saveAndRender() {
    await save()
    await buildStore.renderSite()
}
</script>

<template>
    <div class="site-config-layout">
        <section class="site-config-main">
            <UiPanel title="Site" subtitle="Basic identity used by the generated homepage, posts, feed, and sitemap.">
                <form class="field-grid" @submit.prevent="save">
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

                    <div class="button-row">
                        <UiButton tone="primary" type="submit" :busy="siteConfigStore.saving">
                            Save site
                        </UiButton>
                        <UiButton tone="ghost" :busy="siteConfigStore.loading" @click="reload">
                            Reload
                        </UiButton>
                        <UiButton tone="ghost" :busy="siteConfigStore.previewing" @click="preview">
                            Preview
                        </UiButton>
                        <UiButton tone="primary" :busy="siteConfigStore.saving || buildStore.rendering" @click="saveAndRender">
                            Save and render site
                        </UiButton>
                    </div>
                </form>
            </UiPanel>

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
                    Generate a preview to inspect the current site settings.
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
.footer-grid,
.result {
    display: grid;
    gap: 1rem;
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
