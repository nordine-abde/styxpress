<script setup>
import { computed, onMounted, ref } from 'vue'
import UiBadge from './ui/UiBadge.vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import UiPanel from './ui/UiPanel.vue'
import { useDeployStore } from '../stores/deploy'

defineProps({
    compact: {
        type: Boolean,
        default: false
    }
})

const deployStore = useDeployStore()
const secret = ref('')

const summaryText = computed(() => {
    const summary = deployStore.status.summary
    if (!summary) {
        return deployStore.status.outOfSync ? 'Local output changed.' : 'Status not checked.'
    }
    return `${summary.localFiles} local files, ${summary.uploaded} new, ${summary.updated} changed, ${summary.deleted} deleted.`
})

const statusTone = computed(() => {
    if (deployStore.status.outOfSync) {
        return 'warning'
    }
    return deployStore.status.summary ? 'success' : 'neutral'
})

const statusLabel = computed(() => {
    if (deployStore.status.outOfSync) {
        return 'Changes ready'
    }
    return deployStore.status.summary ? 'Synced' : 'Not checked'
})

const subtitle = computed(() => {
    return 'Manual SFTP deployment for the generated public folder.'
})

async function saveSecret() {
    if (!secret.value) {
        return
    }
    await deployStore.saveSecret(secret.value)
    secret.value = ''
}

async function clearSecret() {
    await deployStore.clearSecret()
    secret.value = ''
}

onMounted(() => {
    if (deployStore.enabled && deployStore.status.configured) {
        deployStore.refreshStatus({ quiet: true }).catch(() => {})
    }
})
</script>

<template>
    <UiPanel
        v-if="deployStore.enabled"
        title="Deploy"
        :subtitle="compact ? '' : subtitle"
    >
        <div class="deploy-panel">
            <div class="deploy-status">
                <UiBadge :tone="statusTone">
                    {{ statusLabel }}
                </UiBadge>
                <p class="muted compact-text">
                    {{ summaryText }}
                </p>
            </div>

            <section class="secret-box">
                <div class="secret-status">
                    <UiBadge :tone="deployStore.status.secretSet ? 'success' : 'warning'">
                        {{ deployStore.status.secretSet ? 'Session secret set' : 'Session secret missing' }}
                    </UiBadge>
                    <p class="muted compact-text">
                        Verified as SFTP password or encrypted key passphrase. Not saved to config.
                    </p>
                </div>
                <UiField
                    v-model="secret"
                    type="password"
                    label="Password / passphrase"
                    autocomplete="current-password"
                    placeholder="Only kept in this server session"
                />
                <div class="button-row">
                    <UiButton tone="primary" :busy="deployStore.savingSecret" :disabled="!secret" @click="saveSecret">
                        Verify for session
                    </UiButton>
                    <UiButton
                        v-if="deployStore.status.secretSet"
                        tone="ghost"
                        :busy="deployStore.clearingSecret"
                        @click="clearSecret"
                    >
                        Clear
                    </UiButton>
                </div>
            </section>

            <div class="button-row">
                <UiButton
                    v-if="deployStore.canDeploy"
                    tone="primary"
                    :busy="deployStore.deploying"
                    @click="deployStore.deployNow"
                >
                    Deploy
                </UiButton>
                <UiButton tone="ghost" :busy="deployStore.checking" @click="deployStore.refreshStatus()">
                    Check
                </UiButton>
            </div>
            <p v-if="deployStore.error" class="error-text compact-text">
                {{ deployStore.error }}
            </p>
        </div>
    </UiPanel>
</template>

<style scoped>
.deploy-panel,
.deploy-status,
.secret-box,
.secret-status {
    display: grid;
    gap: 0.65rem;
}

.secret-box {
    border-top: 1px solid var(--color-border);
    padding-top: 0.85rem;
}

.compact-text {
    margin: 0;
}
</style>
