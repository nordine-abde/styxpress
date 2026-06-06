import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { apiRequest } from '../api/client'
import { useUiStore } from './ui'

const emptyStatus = {
    enabled: false,
    configured: false,
    mode: 'manual',
    outOfSync: false,
    summary: null
}

export const useDeployStore = defineStore('deploy', () => {
    const status = ref({ ...emptyStatus })
    const checking = ref(false)
    const deploying = ref(false)
    const error = ref('')
    let configSignature = ''

    const enabled = computed(() => status.value.enabled === true)
    const automatic = computed(() => enabled.value && status.value.mode === 'auto')
    const manual = computed(() => enabled.value && status.value.mode === 'manual')
    const canDeploy = computed(() => manual.value && status.value.configured && status.value.outOfSync)
    const busy = computed(() => checking.value || deploying.value)

    function syncFromConfig(config) {
        const deploy = config?.deploy || {}
        const sftp = deploy.sftp || {}
        const nextSignature = JSON.stringify({
            enabled: deploy.enabled === true,
            mode: deploy.mode === 'auto' ? 'auto' : 'manual',
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
            mode: deploy.mode === 'auto' ? 'auto' : 'manual',
            outOfSync: sameConfig && deploy.enabled === true ? status.value.outOfSync : false,
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

    function applyBuildResult(result) {
        if (!enabled.value) {
            return
        }
        if (result?.deploy) {
            status.value = {
                ...status.value,
                configured: true,
                outOfSync: false,
                summary: result.deploy
            }
            return
        }
        if (manual.value && status.value.configured) {
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
            mode: payload.mode === 'auto' ? 'auto' : 'manual',
            outOfSync: payload.outOfSync === true,
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
        error,
        enabled,
        automatic,
        manual,
        canDeploy,
        busy,
        syncFromConfig,
        refreshStatus,
        deployNow,
        applyBuildResult,
        reset
    }
})
