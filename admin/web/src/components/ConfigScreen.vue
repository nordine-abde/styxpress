<script setup>
import { computed, reactive, ref, watch } from 'vue'
import UiBadge from './ui/UiBadge.vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import UiPanel from './ui/UiPanel.vue'
import UiSelect from './ui/UiSelect.vue'
import { useConfigStore } from '../stores/config'
import { usePublishingStore } from '../stores/publishing'

const configStore = useConfigStore()
const publishingStore = usePublishingStore()
const sshMode = ref('disabled')

const form = reactive({ ...configStore.config })
const storageOptions = [
    { value: 'local', label: 'Local content' },
    { value: 'server', label: 'Server-backed content' }
]
const sshOptions = [
    { value: 'disabled', label: 'No SSH publishing' },
    { value: 'enabled', label: 'Use SSH publishing' }
]
const sshEnabled = computed(() => sshMode.value === 'enabled')
const sshBannerText = computed(() => {
    if (!sshEnabled.value) {
        return 'Publishing over SSH is disabled for this site.'
    }
    if (publishingStore.sshStatus === 'ok') {
        return 'Connection test succeeded for the saved SSH settings.'
    }
    if (publishingStore.sshStatus === 'error') {
        return publishingStore.sshError || 'The configured SSH server is not reachable.'
    }
    if (publishingStore.sshStatus === 'checking') {
        return 'Testing the saved SSH settings.'
    }
    return 'This site uses SSH. Add the key settings, enter the passphrase if needed, then test the connection before publishing.'
})

watch(
    () => configStore.config,
    (value) => {
        Object.assign(form, value)
        sshMode.value = hasSSHFields(value) ? 'enabled' : 'disabled'
    },
    { deep: true, immediate: true }
)

async function save() {
    await configStore.saveConfig(configForSave())
    publishingStore.prepareSSHForSite(configStore.activeSiteId, configStore.config)
}

async function saveAndTestSSH() {
    await save()
    await publishingStore.testSSH()
}

function configForSave() {
    if (sshEnabled.value) {
        return { ...form }
    }
    return {
        ...form,
        remoteHost: '',
        remoteUser: '',
        sshKeyPath: '',
        remotePublicDir: '',
        remoteContentDir: ''
    }
}

function hasSSHFields(value = {}) {
    return Boolean(
        value.remoteHost?.trim() ||
        value.remoteUser?.trim() ||
        value.sshKeyPath?.trim() ||
        value.remotePublicDir?.trim() ||
        value.remoteContentDir?.trim()
    )
}
</script>

<template>
    <UiPanel title="Site configuration" subtitle="Local paths are resolved by the Go server. Passphrases are never saved.">
        <form class="field-grid" @submit.prevent="save">
            <div class="two-column">
                <UiField v-model="form.name" label="Site name" placeholder="My site" />
                <UiField v-model="form.siteBaseUrl" label="Site URL" placeholder="https://blog.example.com" />
                <UiSelect v-model="form.contentStorageMode" label="Content storage" :options="storageOptions" />
                <UiField v-model="form.contentDir" label="Content directory" />
                <UiField v-model="form.publicDir" label="Public directory" />
            </div>

            <section class="connection-section">
                <div class="connection-header">
                    <div>
                        <h4>Publishing connection</h4>
                        <p>Choose whether this site publishes to a remote host.</p>
                    </div>
                    <UiBadge :tone="publishingStore.sshStatusTone">
                        {{ publishingStore.sshStatusLabel }}
                    </UiBadge>
                </div>

                <UiSelect v-model="sshMode" label="SSH publishing" :options="sshOptions" />

                <div class="connection-banner" :class="publishingStore.sshStatus">
                    <p>{{ sshBannerText }}</p>
                </div>

                <div v-if="sshEnabled" class="field-grid">
                    <div class="two-column">
                        <UiField v-model="form.remoteHost" label="SSH host" placeholder="example.com:22" />
                        <UiField v-model="form.remoteUser" label="SSH user" placeholder="deploy" />
                        <UiField v-model="form.sshKeyPath" label="SSH key path" placeholder="/home/user/.ssh/id_ed25519" />
                        <UiField v-model="form.remotePublicDir" label="Remote public directory" placeholder="/srv/site/public" />
                        <UiField v-model="form.remoteContentDir" label="Remote content directory" placeholder="/srv/site/content" />
                    </div>

                    <div class="ssh-test-row">
                        <UiField
                            v-model="publishingStore.sshPassphrase"
                            label="SSH passphrase"
                            type="password"
                            placeholder="Optional"
                            help="Used for this admin session only."
                        />
                        <UiButton
                            tone="primary"
                            :busy="configStore.saving || publishingStore.testing"
                            @click="saveAndTestSSH"
                        >
                            Save and test SSH
                        </UiButton>
                    </div>
                </div>
            </section>

            <div class="button-row">
                <UiButton tone="primary" type="submit" :busy="configStore.saving">
                    Save config
                </UiButton>
                <UiButton tone="ghost" :busy="configStore.loading" @click="configStore.loadConfig">
                    Reload
                </UiButton>
            </div>
        </form>
    </UiPanel>
</template>

<style scoped>
.connection-section,
.connection-header,
.ssh-test-row {
    display: grid;
    gap: 0.85rem;
}

.connection-header {
    align-items: start;
}

.connection-header h4,
.connection-header p,
.connection-banner p {
    margin: 0;
}

.connection-header h4 {
    color: var(--color-heading);
    font-size: 0.95rem;
}

.connection-header p,
.connection-banner p {
    color: var(--color-muted);
    font-size: 0.88rem;
}

.connection-banner {
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 0.75rem;
    background: var(--color-surface-muted);
}

.connection-banner.error {
    border-color: color-mix(in srgb, var(--color-danger) 45%, var(--color-border));
}

.connection-banner.ok {
    border-color: color-mix(in srgb, var(--color-success) 45%, var(--color-border));
}

@media (min-width: 720px) {
    .connection-header,
    .ssh-test-row {
        grid-template-columns: minmax(0, 1fr) auto;
        align-items: end;
    }
}
</style>
