import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { apiRequest } from '../api/client'
import { usePostsStore } from './posts'
import { useUiStore } from './ui'

export const usePublishingStore = defineStore('publishing', () => {
    const testing = ref(false)
    const rendering = ref(false)
    const publishing = ref(false)
    const verifying = ref(false)
    const lastResult = ref(null)
    const verificationResult = ref(null)
    const verificationCheckedAt = ref('')
    const error = ref('')
    const sshPassphrase = ref('')
    const sshStatus = ref('disabled')
    const sshError = ref('')
    const activeSSHSiteId = ref('')
    const activeSSHConfigSignature = ref('')
    const sshRequiresPassphrase = ref(false)
    const autoVerifiedSiteIds = new Set()
    let verificationRequest = null
    let queuedVerification = null
    let verificationGeneration = 0

    const sshEnabled = computed(() => sshStatus.value !== 'disabled')
    const sshNeedsPassphrase = computed(() => sshStatus.value === 'needs_passphrase')
    const sshBlocksEditing = computed(() => sshEnabled.value && sshStatus.value !== 'ok')

    const sshStatusLabel = computed(() => {
        if (sshStatus.value === 'ok') {
            return 'SSH reachable'
        }
        if (sshStatus.value === 'needs_passphrase') {
            return 'Passphrase required'
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
            return 'danger'
        }
        if (sshStatus.value === 'pending' || sshStatus.value === 'checking' || sshStatus.value === 'needs_passphrase') {
            return 'warning'
        }
        return 'neutral'
    })

    const remoteVerificationByPath = computed(() => {
        const byPath = new Map()
        for (const file of verificationResult.value?.files || []) {
            if (file.relativePath) {
                byPath.set(file.relativePath, file)
            }
        }
        return byPath
    })

    const siteShellVerification = computed(() => {
        if (!verificationResult.value) {
            return 'unknown'
        }
        const files = (verificationResult.value.files || []).filter((file) => !file.relativePath?.startsWith('posts/'))
        return aggregateVerificationStatus(files)
    })

    const verificationSummaryText = computed(() => {
        if (verifying.value && !verificationResult.value) {
            return 'Checking remote files.'
        }
        if (verifying.value) {
            return 'Refreshing remote verification.'
        }
        if (!verificationResult.value) {
            return 'Remote verification has not run.'
        }
        const summary = verificationResult.value.summary || {}
        const total = summary.total || 0
        const published = summary.published || 0
        const pending = (summary.changesPending || 0) + (summary.notOnRemote || 0) + (summary.stillOnRemote || 0)
        if (summary.unknown) {
            return `${published}/${total} files match; ${summary.unknown} could not be checked.`
        }
        if (pending) {
            return `${published}/${total} files match; ${pending} need attention.`
        }
        return `${published}/${total} files match.`
    })

    function prepareSSHForSite(siteId, cfg) {
        const nextSiteId = siteId || ''
        const nextSignature = sshConfigSignature(cfg)
        if (activeSSHSiteId.value !== nextSiteId || activeSSHConfigSignature.value !== nextSignature) {
            sshPassphrase.value = ''
            lastResult.value = null
            clearVerification()
            error.value = ''
            autoVerifiedSiteIds.delete(siteVerificationKey(nextSiteId))
        }
        activeSSHSiteId.value = nextSiteId
        activeSSHConfigSignature.value = nextSignature
        sshError.value = ''
        sshRequiresPassphrase.value = Boolean(cfg?.sshUsePassphrase)
        if (!hasSSHConfig(cfg)) {
            sshStatus.value = 'disabled'
            return
        }
        sshStatus.value = sshRequiresPassphrase.value && !sshPassphrase.value
            ? 'needs_passphrase'
            : 'pending'
    }

    function resetSSH() {
        sshPassphrase.value = ''
        sshStatus.value = 'disabled'
        sshError.value = ''
        sshRequiresPassphrase.value = false
        activeSSHSiteId.value = ''
        activeSSHConfigSignature.value = ''
        lastResult.value = null
        autoVerifiedSiteIds.clear()
        clearVerification()
        error.value = ''
    }

    async function testSSH(passphrase = sshPassphrase.value) {
        const uiStore = useUiStore()
        if (!sshEnabled.value) {
            return true
        }
        const normalizedPassphrase = sshRequiresPassphrase.value ? String(passphrase || '') : ''
        if (sshRequiresPassphrase.value && !normalizedPassphrase.trim()) {
            const message = 'Enter the SSH key passphrase before testing this site.'
            error.value = message
            sshError.value = message
            sshStatus.value = 'needs_passphrase'
            return false
        }
        testing.value = true
        error.value = ''
        sshError.value = ''
        sshStatus.value = 'checking'
        try {
            await apiRequest('/api/test-ssh', {
                method: 'POST',
                body: { passphrase: normalizedPassphrase }
            })
            if (sshRequiresPassphrase.value) {
                sshPassphrase.value = normalizedPassphrase
            }
            sshStatus.value = 'ok'
            uiStore.setNotice('SSH connection succeeded.')
            return true
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
        clearVerification()
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
        const postsStore = usePostsStore()
        publishing.value = true
        error.value = ''
        clearVerification()
        try {
            lastResult.value = await apiRequest(`/api/posts/${encodeURIComponent(slug)}/publish`, {
                method: 'POST',
                body: { passphrase }
            })
            await postsStore.loadPosts()
            await postsStore.selectPost(slug)
            uiStore.setNotice('Post published.')
            void verifyRemoteAfterPublish(passphrase)
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
        clearVerification()
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
        const postsStore = usePostsStore()
        publishing.value = true
        error.value = ''
        clearVerification()
        try {
            lastResult.value = await apiRequest('/api/site/publish', {
                method: 'POST',
                body: { passphrase }
            })
            const selectedSlug = postsStore.selectedSlug
            await postsStore.loadPosts()
            if (selectedSlug) {
                await postsStore.selectPost(selectedSlug)
            }
            uiStore.setNotice('Site published.')
            void verifyRemoteAfterPublish(passphrase)
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            publishing.value = false
        }
    }

    async function verifyRemote(passphrase = sshPassphrase.value, options = {}) {
        const uiStore = useUiStore()
        const postsStore = usePostsStore()
        const notify = options.notify !== false
        const throwOnError = options.throwOnError !== false
        if (verificationRequest) {
            if (options.queueIfBusy) {
                queuedVerification = { passphrase, options }
            }
            return verificationRequest
        }
        if (!sshEnabled.value) {
            const message = 'Configure SSH publishing before verifying the remote site.'
            error.value = message
            if (notify) {
                uiStore.captureError(new Error(message))
            }
            return null
        }
        const normalizedPassphrase = sshRequiresPassphrase.value ? String(passphrase || '') : ''
        if (sshRequiresPassphrase.value && !normalizedPassphrase.trim()) {
            const message = 'Enter the SSH key passphrase before verifying the remote site.'
            error.value = message
            sshError.value = message
            sshStatus.value = 'needs_passphrase'
            return null
        }
        const requestGeneration = ++verificationGeneration
        verifying.value = true
        error.value = ''
        verificationRequest = (async () => {
            const result = normalizeVerificationResult(await apiRequest('/api/site/verify-remote', {
                method: 'POST',
                body: { passphrase: normalizedPassphrase }
            }))
            if (requestGeneration === verificationGeneration) {
                verificationResult.value = result
                verificationCheckedAt.value = new Date().toISOString()
            }
            const selectedSlug = postsStore.selectedSlug
            await postsStore.loadPosts()
            if (selectedSlug) {
                await postsStore.selectPost(selectedSlug)
            }
            if (notify) {
                uiStore.setNotice('Remote verification completed.')
            }
            return result
        })()
        try {
            return await verificationRequest
        } catch (err) {
            error.value = err.message
            if (requestGeneration === verificationGeneration) {
                verificationResult.value = null
                verificationCheckedAt.value = ''
            }
            if (notify) {
                uiStore.captureError(err)
            }
            if (throwOnError) {
                throw err
            }
            return null
        } finally {
            verificationRequest = null
            verifying.value = false
            if (queuedVerification) {
                const nextVerification = queuedVerification
                queuedVerification = null
                void verifyRemote(nextVerification.passphrase, nextVerification.options)
            }
        }
    }

    function verifyRemoteAfterOpen(passphrase = sshPassphrase.value) {
        const key = siteVerificationKey(activeSSHSiteId.value)
        if (autoVerifiedSiteIds.has(key) || !canVerifyRemoteNow(passphrase)) {
            return Promise.resolve(null)
        }
        autoVerifiedSiteIds.add(key)
        return verifyRemote(passphrase, {
            notify: false,
            throwOnError: false,
            queueIfBusy: true
        }).catch(() => null)
    }

    function verifyRemoteAfterPublish(passphrase = sshPassphrase.value) {
        if (!canVerifyRemoteNow(passphrase)) {
            return Promise.resolve(null)
        }
        return verifyRemote(passphrase, {
            notify: false,
            throwOnError: false,
            queueIfBusy: true
        }).catch(() => null)
    }

    function clearVerification() {
        verificationGeneration += 1
        verificationResult.value = null
        verificationCheckedAt.value = ''
    }

    function canVerifyRemoteNow(passphrase = sshPassphrase.value) {
        if (!sshEnabled.value || sshStatus.value !== 'ok') {
            return false
        }
        if (sshRequiresPassphrase.value && !String(passphrase || '').trim()) {
            return false
        }
        return true
    }

    function postRemoteStatus(post) {
        if (!post?.slug) {
            return 'unknown'
        }
        const draftRemotePath = `posts/${post.slug}/index.html`
        const staleDraft = remoteVerificationByPath.value.get(draftRemotePath)?.status === 'still_on_remote'
        if (post.publishStatus === 'draft') {
            return staleDraft ? 'still_on_remote' : 'draft'
        }
        if (post.publishStatus === 'pending_publish') {
            return 'changes_pending'
        }
        if (!verificationResult.value) {
            return 'unknown'
        }
        const prefix = `posts/${post.slug}/`
        const files = (verificationResult.value.files || []).filter((file) => file.relativePath?.startsWith(prefix))
        return aggregateVerificationStatus(files)
    }

    function verificationStatusLabel(status) {
        if (status === 'published') {
            return 'Published'
        }
        if (status === 'changes_pending') {
            return 'Changes pending'
        }
        if (status === 'not_on_remote') {
            return 'Not on remote'
        }
        if (status === 'still_on_remote') {
            return 'Still on remote'
        }
        if (status === 'draft') {
            return 'Draft'
        }
        return 'Unknown'
    }

    function verificationStatusTone(status) {
        if (status === 'published') {
            return 'success'
        }
        if (status === 'changes_pending' || status === 'not_on_remote') {
            return 'warning'
        }
        if (status === 'still_on_remote') {
            return 'danger'
        }
        return 'neutral'
    }

    return {
        testing,
        rendering,
        publishing,
        verifying,
        lastResult,
        verificationResult,
        verificationCheckedAt,
        error,
        sshPassphrase,
        sshStatus,
        sshError,
        sshRequiresPassphrase,
        sshEnabled,
        sshNeedsPassphrase,
        sshBlocksEditing,
        sshStatusLabel,
        sshStatusTone,
        siteShellVerification,
        verificationSummaryText,
        prepareSSHForSite,
        resetSSH,
        testSSH,
        renderPost,
        publishPost,
        renderSite,
        publishSite,
        verifyRemote,
        verifyRemoteAfterOpen,
        verifyRemoteAfterPublish,
        clearVerification,
        postRemoteStatus,
        verificationStatusLabel,
        verificationStatusTone
    }
})

function normalizeVerificationResult(result = {}) {
    return {
        files: Array.isArray(result.files) ? result.files : [],
        summary: {
            total: result.summary?.total || 0,
            published: result.summary?.published || 0,
            changesPending: result.summary?.changesPending || 0,
            notOnRemote: result.summary?.notOnRemote || 0,
            unknown: result.summary?.unknown || 0,
            stillOnRemote: result.summary?.stillOnRemote || 0
        }
    }
}

function aggregateVerificationStatus(files) {
    if (!files.length) {
        return 'not_on_remote'
    }
    if (files.some((file) => file.status === 'still_on_remote')) {
        return 'still_on_remote'
    }
    if (files.some((file) => file.status === 'not_on_remote')) {
        return 'not_on_remote'
    }
    if (files.some((file) => file.status === 'changes_pending')) {
        return 'changes_pending'
    }
    if (files.some((file) => file.status === 'unknown')) {
        return 'unknown'
    }
    if (files.every((file) => file.status === 'published')) {
        return 'published'
    }
    return 'unknown'
}

function hasSSHConfig(cfg = {}) {
    return Boolean(
        cfg.remoteHost?.trim() ||
        cfg.remoteUser?.trim() ||
        cfg.sshKeyPath?.trim() ||
        cfg.remotePublicDir?.trim() ||
        cfg.remoteContentDir?.trim()
    )
}

function sshConfigSignature(cfg = {}) {
    return [
        cfg.contentDir || '',
        cfg.publicDir || '',
        cfg.remoteHost || '',
        cfg.remoteUser || '',
        cfg.sshKeyPath || '',
        cfg.sshUsePassphrase ? 'passphrase' : 'no-passphrase',
        cfg.remotePublicDir || '',
        cfg.remoteContentDir || ''
    ].join('\u001f')
}

function siteVerificationKey(siteId) {
    return siteId || 'single-site'
}
