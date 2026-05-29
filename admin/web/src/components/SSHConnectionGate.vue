<script setup>
import { computed } from 'vue'
import UiBadge from './ui/UiBadge.vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import { usePublishingStore } from '../stores/publishing'
import { useSiteWorkspaceStore } from '../stores/siteWorkspace'
import { useUiStore } from '../stores/ui'

const publishingStore = usePublishingStore()
const siteWorkspaceStore = useSiteWorkspaceStore()
const uiStore = useUiStore()

const title = computed(() => {
    if (publishingStore.sshStatus === 'error') {
        return 'SSH connection failed'
    }
    if (publishingStore.sshStatus === 'checking') {
        return 'Checking SSH connection'
    }
    if (publishingStore.sshNeedsPassphrase) {
        return 'This site needs an SSH passphrase'
    }
    return 'SSH verification required'
})

const message = computed(() => {
    if (publishingStore.sshStatus === 'error') {
        return publishingStore.sshError || 'The configured SSH server is not reachable.'
    }
    if (publishingStore.sshStatus === 'checking') {
        return 'Styxpress is testing the saved SSH settings before opening this site.'
    }
    if (publishingStore.sshNeedsPassphrase) {
        return 'Enter the SSH key passphrase to continue. Content, styles, and featured posts stay locked until verification succeeds.'
    }
    return 'Verify the saved SSH connection before editing this site.'
})

const showPassphraseField = computed(() => {
    return publishingStore.sshRequiresPassphrase && publishingStore.sshStatus !== 'checking'
})
const canTest = computed(() => {
    return publishingStore.sshStatus !== 'checking' &&
        (!publishingStore.sshRequiresPassphrase || Boolean(publishingStore.sshPassphrase.trim()))
})
const primaryActionLabel = computed(() => {
    if (publishingStore.sshStatus === 'error') {
        return 'Retry connection'
    }
    if (publishingStore.sshRequiresPassphrase) {
        return 'Verify and open site'
    }
    return 'Verify connection'
})

async function testConnection() {
    let ok = false
    try {
        ok = await publishingStore.testSSH()
    } catch {
        ok = false
    }
    if (ok) {
        await siteWorkspaceStore.loadCurrentSite({ force: true })
        uiStore.setActiveView('config')
    }
}

function editConfiguration() {
    uiStore.setActiveView('config')
}
</script>

<template>
    <section v-if="publishingStore.sshBlocksEditing" class="ssh-gate" role="alert">
        <div class="ssh-gate-panel">
            <div class="ssh-gate-copy">
                <div class="status-row">
                    <strong>{{ title }}</strong>
                    <UiBadge :tone="publishingStore.sshStatusTone">
                        {{ publishingStore.sshStatusLabel }}
                    </UiBadge>
                </div>
                <p>{{ message }}</p>
            </div>

            <div v-if="publishingStore.sshStatus === 'checking'" class="checking-row" aria-live="polite">
                <span class="spinner" aria-hidden="true"></span>
                <span>Checking the connection...</span>
            </div>

            <form v-else class="ssh-gate-actions" @submit.prevent="testConnection">
                <UiField
                    v-if="showPassphraseField"
                    v-model="publishingStore.sshPassphrase"
                    label="SSH key passphrase"
                    type="password"
                    placeholder="Required for this session"
                />
                <UiButton
                    tone="primary"
                    type="submit"
                    :busy="publishingStore.testing || siteWorkspaceStore.loading"
                    :disabled="!canTest"
                >
                    {{ primaryActionLabel }}
                </UiButton>
            </form>

            <button class="configuration-link" type="button" @click="editConfiguration">
                Edit configuration
            </button>
        </div>
    </section>
</template>

<style scoped>
.ssh-gate {
    display: grid;
    min-height: min(32rem, calc(100vh - 10rem));
    align-items: center;
}

.ssh-gate-panel {
    display: grid;
    gap: 1rem;
    width: min(100%, 42rem);
    margin: 0 auto;
    border: 1px solid color-mix(in srgb, var(--color-danger) 42%, var(--color-border));
    border-radius: 8px;
    padding: 1rem;
    background: color-mix(in srgb, var(--color-danger) 10%, var(--color-surface));
}

.ssh-gate-copy,
.ssh-gate-actions {
    display: grid;
    gap: 0.65rem;
}

strong {
    color: var(--color-heading);
}

p {
    margin: 0;
    color: var(--color-text);
    overflow-wrap: anywhere;
}

.checking-row,
.configuration-link {
    color: var(--color-muted);
    font-weight: 700;
}

.checking-row {
    display: inline-flex;
    gap: 0.55rem;
    align-items: center;
}

.configuration-link {
    width: fit-content;
    border: 0;
    padding: 0;
    background: transparent;
    text-align: left;
    text-decoration: underline;
    text-underline-offset: 0.18rem;
}

.configuration-link:hover {
    color: var(--color-heading);
}

.spinner {
    width: 0.9rem;
    height: 0.9rem;
    border: 2px solid currentColor;
    border-right-color: transparent;
    border-radius: 999px;
    animation: spin 0.7s linear infinite;
}

@keyframes spin {
    to {
        transform: rotate(360deg);
    }
}
</style>
