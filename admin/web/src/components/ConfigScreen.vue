<script setup>
import { reactive, watch } from 'vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import UiPanel from './ui/UiPanel.vue'
import { useConfigStore } from '../stores/config'
import { useSiteWorkspaceStore } from '../stores/siteWorkspace'

const configStore = useConfigStore()
const siteWorkspaceStore = useSiteWorkspaceStore()
const form = reactive({ ...configStore.config })

watch(
    () => configStore.config,
    (value) => {
        Object.assign(form, value)
    },
    { deep: true, immediate: true }
)

async function save() {
    await configStore.saveConfig({ ...form })
    await siteWorkspaceStore.loadCurrentSite({ force: true })
}

async function reload() {
    await configStore.loadConfig()
}
</script>

<template>
    <UiPanel title="Configuration" subtitle="Styxpress reads content locally and writes generated public files locally.">
        <form class="field-grid" @submit.prevent="save">
            <UiField v-model="form.name" label="Site name" placeholder="My blog" />
            <div class="two-column">
                <UiField
                    v-model="form.contentDir"
                    label="Content folder"
                    help="Source Markdown, metadata, media, and site.toml live here."
                />
                <UiField
                    v-model="form.publicDir"
                    label="Public folder"
                    help="Generated HTML, feed, sitemap, stylesheet, and assets are written here."
                />
            </div>

            <div class="button-row">
                <UiButton tone="primary" type="submit" :busy="configStore.saving || siteWorkspaceStore.loading">
                    Save config
                </UiButton>
                <UiButton tone="ghost" :busy="configStore.loading" @click="reload">
                    Reload
                </UiButton>
            </div>
        </form>
    </UiPanel>
</template>
