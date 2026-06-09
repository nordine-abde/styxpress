<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import UiBadge from './ui/UiBadge.vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import UiPanel from './ui/UiPanel.vue'
import UiSwitch from './ui/UiSwitch.vue'
import { useConfigStore } from '../stores/config'
import { useDeployStore } from '../stores/deploy'
import { useSiteWorkspaceStore } from '../stores/siteWorkspace'
import { useUiStore } from '../stores/ui'

const configStore = useConfigStore()
const deployStore = useDeployStore()
const siteWorkspaceStore = useSiteWorkspaceStore()
const uiStore = useUiStore()
const form = reactive(cloneConfig(configStore.config))
const deploySecret = ref('')
const showKeyInput = ref(false)
const setupWarning = ref(null)
const operationError = ref('')
const savedSnapshot = ref(snapshotConfig(form))

const savedKeyPath = computed(() => configStore.config?.deploy?.sftp?.keyPath?.trim() || '')
const keyInputHidden = computed(() => (
    Boolean(savedKeyPath.value) &&
    !showKeyInput.value &&
    form.deploy.sftp.keyPath === savedKeyPath.value
))
const firstDeploySetup = computed(() => requiresDeploySetup(form))
const saveBusy = computed(() => configStore.saving || siteWorkspaceStore.loading)
const configDirty = computed(() => snapshotConfig(form) !== savedSnapshot.value)

watch(
    () => configStore.config,
    (value) => {
        const cloned = cloneConfig(value)
        Object.assign(form, cloned)
        savedSnapshot.value = snapshotConfig(cloned)
        deploySecret.value = ''
        setupWarning.value = null
        operationError.value = ''
        showKeyInput.value = !value?.deploy?.sftp?.keyPath
    },
    { deep: true, immediate: true }
)

async function save() {
    await persistConfig(false)
}

async function confirmRemoteOverwrite() {
    await persistConfig(true)
}

async function persistConfig(confirmRemoteOverwrite) {
    setupWarning.value = null
    operationError.value = ''
    const nextConfig = cloneConfig(form)
    try {
        if (requiresDeploySetup(nextConfig)) {
            const payload = await configStore.setupDeployConfig(nextConfig, {
                secret: deploySecret.value,
                confirmRemoteOverwrite
            })
            if (payload?.requiresConfirmation) {
                setupWarning.value = payload
                return
            }
            deploySecret.value = ''
        } else {
            await configStore.saveConfig(nextConfig)
        }
        await siteWorkspaceStore.loadCurrentSite({ force: true })
        if (deployStore.enabled && deployStore.status.configured) {
            await deployStore.refreshStatus({ quiet: true }).catch(() => {})
        }
        const savedConfig = cloneConfig(configStore.config)
        Object.assign(form, savedConfig)
        savedSnapshot.value = snapshotConfig(savedConfig)
    } catch (err) {
        operationError.value = err?.message || 'Configuration could not be saved.'
    }
}

async function reload() {
    await configStore.loadConfig()
}

function cloneConfig(value) {
    return JSON.parse(JSON.stringify(value))
}

function snapshotConfig(value) {
    return JSON.stringify(cloneConfig(value))
}

function changeKey() {
    showKeyInput.value = true
}

function removeKey() {
    form.deploy.sftp.keyPath = ''
    showKeyInput.value = true
}

function deployConfigured(value) {
    const sftp = value?.deploy?.sftp || {}
    return Boolean(
        String(sftp.host || '').trim() &&
        String(sftp.user || '').trim() &&
        String(sftp.remotePath || '').trim()
    )
}

function requiresDeploySetup(value) {
    return value?.deploy?.enabled === true && deployConfigured(value) && !deployConfigured(configStore.config)
}

onMounted(() => {
    uiStore.registerHeaderSaveAction('config', {
        isAvailable: () => true,
        isDirty: () => configDirty.value,
        isBusy: () => saveBusy.value,
        run: save
    })
})

onBeforeUnmount(() => {
    uiStore.unregisterHeaderSaveAction('config')
})
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
                        <UiSwitch
                            v-model="form.deploy.enabled"
                            label="Enable SFTP deploy"
                            description="Syncs the generated public folder to a remote host."
                        />

                        <div v-if="form.deploy.enabled" class="field-grid">
                            <div class="managed-warning">
                                <strong>Remote folder is managed by Styxpress.</strong>
                                <p>Manual deploy overwrites matching files and removes remote files that are not in the local public folder.</p>
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
                                <div v-if="keyInputHidden" class="key-configured">
                                    <div>
                                        <span>Key file</span>
                                        <UiBadge tone="success">configured</UiBadge>
                                    </div>
                                    <div class="inline-actions">
                                        <UiButton tone="ghost" @click="changeKey">Change</UiButton>
                                        <UiButton tone="ghost" @click="removeKey">Remove</UiButton>
                                    </div>
                                </div>
                                <UiField
                                    v-else
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

                            <UiField
                                v-if="firstDeploySetup"
                                v-model="deploySecret"
                                type="password"
                                label="Password or key passphrase"
                                autocomplete="current-password"
                                help="Optional when ssh-agent or an unencrypted key can authenticate. If provided, it must pass an SSH test before saving."
                            />

                            <div v-if="setupWarning" class="setup-warning" role="alert">
                                <strong>Remote folder is not empty.</strong>
                                <p>{{ setupWarning.remoteFiles }} remote files will be overwritten or removed when this setup is saved.</p>
                                <div class="inline-actions">
                                    <UiButton tone="danger" :busy="configStore.saving" @click="confirmRemoteOverwrite">
                                        Save and replace remote files
                                    </UiButton>
                                    <UiButton tone="ghost" @click="setupWarning = null">
                                        Cancel
                                    </UiButton>
                                </div>
                            </div>
                        </div>
                    </div>
                </section>

                <div class="button-row">
                    <UiButton tone="ghost" :busy="configStore.loading" @click="reload">
                        Reload
                    </UiButton>
                </div>
                <p v-if="configStore.saving && firstDeploySetup" class="progress-text">
                    Testing SFTP connection...
                </p>
                <p v-if="operationError" class="form-error" role="alert">
                    {{ operationError }}
                </p>
            </form>
        </UiPanel>
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

.managed-warning,
.setup-warning,
.key-configured {
    display: grid;
    gap: 0.55rem;
    border: 1px solid color-mix(in srgb, var(--color-warning) 34%, var(--color-border));
    border-radius: 8px;
    padding: 0.8rem;
    background: color-mix(in srgb, var(--color-warning) 9%, var(--color-surface));
}

.key-configured {
    border-color: color-mix(in srgb, var(--color-border) 85%, white);
    background: color-mix(in srgb, var(--color-surface-muted) 72%, white);
}

.managed-warning p,
.setup-warning p {
    margin: 0;
    color: var(--color-muted);
}

.managed-warning strong,
.setup-warning strong,
.key-configured span {
    color: var(--color-heading);
    font-size: 0.82rem;
    font-weight: 800;
}

.key-configured > div:first-child,
.inline-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.55rem;
    align-items: center;
}

.progress-text,
.form-error {
    margin: 0;
    font-size: 0.86rem;
}

.progress-text {
    color: var(--color-muted);
}

.form-error {
    color: var(--color-danger);
    font-weight: 700;
    overflow-wrap: anywhere;
}
</style>
