import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { apiRequest } from '../api/client'
import { useUiStore } from './ui'

const emptyStatus = {
    enabled: false,
    configured: false,
    outOfSync: false,
    secretSet: false,
    summary: null
}

export const useDeployStore = defineStore('deploy', () => {
    const status = ref({ ...emptyStatus })
    const checking = ref(false)
    const deploying = ref(false)
    const savingSecret = ref(false)
    const clearingSecret = ref(false)
    const error = ref('')
    let configSignature = ''

    const enabled = computed(() => status.value.enabled === true)
    const canDeploy = computed(() => enabled.value && status.value.configured && status.value.outOfSync)
    const busy = computed(() => checking.value || deploying.value || savingSecret.value || clearingSecret.value)

    function syncFromConfig(config) {
        const deploy = config?.deploy || {}
        const sftp = deploy.sftp || {}
        const nextSignature = JSON.stringify({
            enabled: deploy.enabled === true,
            host: sftp.host || '',
            port: Number(sftp.port || 22),
            user: sftp.user || '',
            remotePath: sftp.remotePath || '',
            keyPath: sftp.keyPath || '',
            knownHostsPath: sftp.knownHostsPath || '',
            deleteExtra: sftp.deleteExtra === true
        })
        const sameConfig = nextSignature === configSignature
        configSignature = nextSignature
        status.value = {
            ...status.value,
            enabled: deploy.enabled === true,
            configured: Boolean(sftp.host && sftp.user && sftp.remotePath),
            outOfSync: sameConfig && deploy.enabled === true ? status.value.outOfSync : false,
            secretSet: status.value.secretSet,
            summary: sameConfig && deploy.enabled === true ? status.value.summary : null
        }
        if (!status.value.enabled) {
            status.value.summary = null
        }
    }

    async function refreshStatus(options = {}) {
        const uiStore = useUiStore()
        checking.value = true
        error.value = ''
        try {
            const payload = await apiRequest('/api/deploy/status')
            applyStatus(payload)
            return status.value
        } catch (err) {
            error.value = err.message
            if (!options.quiet) {
                uiStore.captureError(err)
            }
            throw err
        } finally {
            checking.value = false
        }
    }

    async function deployNow() {
        const uiStore = useUiStore()
        deploying.value = true
        error.value = ''
        try {
            const summary = await apiRequest('/api/deploy', {
                method: 'POST',
                body: {}
            })
            status.value = {
                ...status.value,
                outOfSync: false,
                summary
            }
            uiStore.setNotice('Deploy completed.')
            return summary
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            deploying.value = false
        }
    }

    async function saveSecret(secret) {
        const uiStore = useUiStore()
        savingSecret.value = true
        error.value = ''
        try {
            const payload = await apiRequest('/api/deploy/secret', {
                method: 'POST',
                body: { secret }
            })
            status.value = {
                ...status.value,
                secretSet: payload?.secretSet === true
            }
            uiStore.setNotice('Deploy secret saved for this session.')
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            savingSecret.value = false
        }
    }

    async function clearSecret() {
        const uiStore = useUiStore()
        clearingSecret.value = true
        error.value = ''
        try {
            const payload = await apiRequest('/api/deploy/secret', {
                method: 'DELETE'
            })
            status.value = {
                ...status.value,
                secretSet: payload?.secretSet === true
            }
            uiStore.setNotice('Deploy secret cleared.')
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            clearingSecret.value = false
        }
    }

    function applyBuildResult() {
        if (!enabled.value) {
            return
        }
        if (status.value.configured) {
            status.value = {
                ...status.value,
                outOfSync: true,
                summary: null
            }
        }
    }

    function applyStatus(payload = {}) {
        status.value = {
            enabled: payload.enabled === true,
            configured: payload.configured === true,
            outOfSync: payload.outOfSync === true,
            secretSet: payload.secretSet === true,
            summary: payload.summary || null
        }
    }

    function reset() {
        status.value = { ...emptyStatus }
        error.value = ''
        configSignature = ''
    }

    return {
        status,
        checking,
        deploying,
        savingSecret,
        clearingSecret,
        error,
        enabled,
        canDeploy,
        busy,
        syncFromConfig,
        refreshStatus,
        deployNow,
        saveSecret,
        clearSecret,
        applyBuildResult,
        reset
    }
})
