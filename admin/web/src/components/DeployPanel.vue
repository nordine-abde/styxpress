<script setup>
import { computed, onMounted } from 'vue'
import UiBadge from './ui/UiBadge.vue'
import UiButton from './ui/UiButton.vue'
import UiPanel from './ui/UiPanel.vue'
import { useDeployStore } from '../stores/deploy'

defineProps({
    compact: {
        type: Boolean,
        default: false
    }
})

const deployStore = useDeployStore()

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

onMounted(() => {
    if (deployStore.enabled && deployStore.status.configured) {
        deployStore.refreshStatus({ quiet: true }).catch(() => {})
    }
})
</script>

<template>
    <UiPanel
        v-if="deployStore.manual"
        title="Deploy"
        :subtitle="compact ? '' : 'Manual SFTP deployment for the generated public folder.'"
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
.deploy-status {
    display: grid;
    gap: 0.65rem;
}

.compact-text {
    margin: 0;
}
</style>
