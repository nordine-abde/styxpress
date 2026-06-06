<script setup>
import { reactive, watch } from 'vue'
import DeployPanel from './DeployPanel.vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import UiPanel from './ui/UiPanel.vue'
import UiSelect from './ui/UiSelect.vue'
import UiSwitch from './ui/UiSwitch.vue'
import { useConfigStore } from '../stores/config'
import { useDeployStore } from '../stores/deploy'
import { useSiteWorkspaceStore } from '../stores/siteWorkspace'

const configStore = useConfigStore()
const deployStore = useDeployStore()
const siteWorkspaceStore = useSiteWorkspaceStore()
const deployModeOptions = [
    { value: 'manual', label: 'Manual' },
    { value: 'auto', label: 'Automatic' }
]
const form = reactive(cloneConfig(configStore.config))

watch(
    () => configStore.config,
    (value) => {
        Object.assign(form, cloneConfig(value))
    },
    { deep: true, immediate: true }
)

async function save() {
    await configStore.saveConfig({ ...form })
    await siteWorkspaceStore.loadCurrentSite({ force: true })
    if (deployStore.enabled && deployStore.status.configured) {
        await deployStore.refreshStatus({ quiet: true }).catch(() => {})
    }
}

async function reload() {
    await configStore.loadConfig()
}

function cloneConfig(value) {
    return JSON.parse(JSON.stringify(value))
}
</script>

<template>
    <div class="config-layout">
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

                <section class="config-section">
                    <h3>SFTP</h3>
                    <div class="field-grid">
                        <div class="two-column">
                            <UiSwitch
                                v-model="form.deploy.enabled"
                                label="Enable SFTP deploy"
                                description="Syncs the generated public folder to a remote host."
                            />
                            <UiSelect v-model="form.deploy.mode" label="Deploy mode" :options="deployModeOptions" />
                        </div>

                        <div class="two-column">
                            <UiField v-model="form.deploy.sftp.host" label="Host" placeholder="example.com" />
                            <UiField v-model="form.deploy.sftp.port" label="Port" type="number" />
                        </div>

                        <div class="two-column">
                            <UiField v-model="form.deploy.sftp.user" label="User" placeholder="deploy" />
                            <UiField v-model="form.deploy.sftp.remotePath" label="Remote folder" placeholder="/public_html" />
                        </div>

                        <div class="two-column">
                            <UiField
                                v-model="form.deploy.sftp.keyPath"
                                label="Key file"
                                placeholder="~/.ssh/id_ed25519"
                            />
                            <UiField
                                v-model="form.deploy.sftp.knownHostsPath"
                                label="Known hosts"
                                placeholder="~/.ssh/known_hosts"
                            />
                        </div>

                        <UiSwitch
                            v-model="form.deploy.sftp.deleteExtra"
                            label="Delete remote extras"
                            description="Removes remote files that are not present in the local public folder."
                        />
                    </div>
                </section>

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

        <DeployPanel />
    </div>
</template>

<style scoped>
.config-layout {
    display: grid;
    gap: 1rem;
}

.config-section {
    display: grid;
    gap: 0.75rem;
    border-top: 1px solid var(--color-border);
    padding-top: 1rem;
}

h3 {
    margin: 0;
    color: var(--color-heading);
    font-size: 0.95rem;
}
</style>
