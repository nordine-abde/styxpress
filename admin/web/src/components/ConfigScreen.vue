<script setup>
import { reactive, ref, watch } from 'vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import UiPanel from './ui/UiPanel.vue'
import UiSelect from './ui/UiSelect.vue'
import { useConfigStore } from '../stores/config'
import { usePublishingStore } from '../stores/publishing'

const configStore = useConfigStore()
const publishingStore = usePublishingStore()
const sshPassphrase = ref('')

const form = reactive({ ...configStore.config })
const storageOptions = [
    { value: 'local', label: 'Local content' },
    { value: 'server', label: 'Server-backed content' }
]

watch(
    () => configStore.config,
    (value) => Object.assign(form, value),
    { deep: true, immediate: true }
)

async function save() {
    await configStore.saveConfig(form)
}
</script>

<template>
    <UiPanel title="Site configuration" subtitle="Local paths are resolved by the Go server. Passphrases are never saved.">
        <form class="field-grid" @submit.prevent="save">
            <div class="two-column">
                <UiField v-model="form.siteBaseUrl" label="Site URL" placeholder="https://blog.example.com" />
                <UiSelect v-model="form.contentStorageMode" label="Content storage" :options="storageOptions" />
                <UiField v-model="form.contentDir" label="Content directory" />
                <UiField v-model="form.publicDir" label="Public directory" />
            </div>

            <div class="two-column">
                <UiField v-model="form.remoteHost" label="SSH host" placeholder="example.com:22" />
                <UiField v-model="form.remoteUser" label="SSH user" placeholder="deploy" />
                <UiField v-model="form.sshKeyPath" label="SSH key path" placeholder="/home/user/.ssh/id_ed25519" />
                <UiField v-model="form.remotePublicDir" label="Remote public directory" placeholder="/srv/site/public" />
                <UiField v-model="form.remoteContentDir" label="Remote content directory" placeholder="/srv/site/content" />
            </div>

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

    <UiPanel title="SSH check">
        <div class="field-grid">
            <UiField
                v-model="sshPassphrase"
                label="SSH passphrase"
                type="password"
                placeholder="Optional"
            />
            <div class="button-row">
                <UiButton
                    tone="primary"
                    :busy="publishingStore.testing"
                    @click="publishingStore.testSSH(sshPassphrase)"
                >
                    Test SSH
                </UiButton>
            </div>
        </div>
    </UiPanel>
</template>
