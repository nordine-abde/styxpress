import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { apiRequest } from '../api/client'
import { useUiStore } from './ui'

export const usePublishingStore = defineStore('publishing', () => {
    const testing = ref(false)
    const rendering = ref(false)
    const publishing = ref(false)
    const lastResult = ref(null)
    const error = ref('')
    const sshPassphrase = ref('')
    const sshStatus = ref('disabled')
    const sshError = ref('')
    const activeSSHSiteId = ref('')

    const sshStatusLabel = computed(() => {
        if (sshStatus.value === 'ok') {
            return 'SSH reachable'
        }
        if (sshStatus.value === 'checking') {
            return 'Checking SSH'
        }
        if (sshStatus.value === 'error') {
            return 'SSH error'
        }
        if (sshStatus.value === 'pending') {
            return 'SSH not tested'
        }
        return 'SSH disabled'
    })

    const sshStatusTone = computed(() => {
        if (sshStatus.value === 'ok') {
            return 'success'
        }
        if (sshStatus.value === 'error') {
            return 'warning'
        }
        if (sshStatus.value === 'pending' || sshStatus.value === 'checking') {
            return 'warning'
        }
        return 'neutral'
    })

    function prepareSSHForSite(siteId, cfg) {
        if (activeSSHSiteId.value !== siteId) {
            sshPassphrase.value = ''
        }
        activeSSHSiteId.value = siteId || ''
        sshError.value = ''
        sshStatus.value = hasSSHConfig(cfg) ? 'pending' : 'disabled'
    }

    async function testSSH(passphrase = sshPassphrase.value) {
        const uiStore = useUiStore()
        testing.value = true
        error.value = ''
        sshError.value = ''
        sshStatus.value = 'checking'
        try {
            await apiRequest('/api/test-ssh', {
                method: 'POST',
                body: { passphrase }
            })
            sshStatus.value = 'ok'
            uiStore.setNotice('SSH connection succeeded.')
        } catch (err) {
            error.value = err.message
            sshError.value = err.message
            sshStatus.value = 'error'
            uiStore.captureError(err)
            throw err
        } finally {
            testing.value = false
        }
    }

    async function renderPost(slug) {
        const uiStore = useUiStore()
        rendering.value = true
        error.value = ''
        try {
            lastResult.value = await apiRequest(`/api/posts/${encodeURIComponent(slug)}/render`, {
                method: 'POST',
                body: {}
            })
            uiStore.setNotice('Post rendered locally.')
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            rendering.value = false
        }
    }

    async function publishPost(slug, passphrase) {
        const uiStore = useUiStore()
        publishing.value = true
        error.value = ''
        try {
            lastResult.value = await apiRequest(`/api/posts/${encodeURIComponent(slug)}/publish`, {
                method: 'POST',
                body: { passphrase }
            })
            uiStore.setNotice('Post published.')
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            publishing.value = false
        }
    }

    async function renderSite() {
        const uiStore = useUiStore()
        rendering.value = true
        error.value = ''
        try {
            lastResult.value = await apiRequest('/api/site/render', {
                method: 'POST',
                body: {}
            })
            uiStore.setNotice('Site rendered locally.')
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            rendering.value = false
        }
    }

    async function publishSite(passphrase) {
        const uiStore = useUiStore()
        publishing.value = true
        error.value = ''
        try {
            lastResult.value = await apiRequest('/api/site/publish', {
                method: 'POST',
                body: { passphrase }
            })
            uiStore.setNotice('Site published.')
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            publishing.value = false
        }
    }

    return {
        testing,
        rendering,
        publishing,
        lastResult,
        error,
        sshPassphrase,
        sshStatus,
        sshError,
        sshStatusLabel,
        sshStatusTone,
        prepareSSHForSite,
        testSSH,
        renderPost,
        publishPost,
        renderSite,
        publishSite
    }
})

function hasSSHConfig(cfg = {}) {
    return Boolean(
        cfg.remoteHost?.trim() ||
        cfg.remoteUser?.trim() ||
        cfg.sshKeyPath?.trim() ||
        cfg.remotePublicDir?.trim() ||
        cfg.remoteContentDir?.trim()
    )
}
