import { defineStore } from 'pinia'
import { computed, ref, watch } from 'vue'
import { apiRequest } from '../api/client'
import { useDeployStore } from './deploy'
import { useUiStore } from './ui'

const emptyPost = {
    slug: '',
    title: '',
    description: '',
    source: '# New post\n',
    cover: '',
    assets: [],
    publishedAt: '',
    updatedAt: '',
    publishStatus: 'draft'
}

function normalizePost(post = {}) {
    const normalized = {
        ...emptyPost,
        ...post,
        assets: Array.isArray(post.assets) ? post.assets : []
    }
    return {
        ...normalized,
        publishStatus: normalizePublishStatus(normalized)
    }
}

function normalizePublishStatus(post) {
    if (['draft', 'published'].includes(post.publishStatus)) {
        return post.publishStatus
    }
    if (!post.publishedAt) {
        return 'draft'
    }
    return 'published'
}

export const usePostsStore = defineStore('posts', () => {
    const posts = ref([])
    const selectedSlug = ref('')
    const draft = ref({ ...emptyPost })
    const editorOpen = ref(false)
    const postSetupOpen = ref(false)
    const loading = ref(false)
    const saving = ref(false)
    const deleting = ref(false)
    const uploading = ref(false)
    const error = ref('')
    const isDirty = ref(false)
    let cleanDraftSnapshot = snapshotPost(draft.value)

    const selectedPost = computed(() => posts.value.find((post) => post.slug === selectedSlug.value))
    const canUploadMedia = computed(() => Boolean(selectedSlug.value || draft.value.slug.trim()))
    const mediaUploadNeedsSave = computed(() => canUploadMedia.value && (!selectedSlug.value || isDirty.value))

    watch(
        draft,
        () => {
            isDirty.value = snapshotPost(draft.value) !== cleanDraftSnapshot
        },
        { deep: true }
    )

    async function loadPosts() {
        const uiStore = useUiStore()
        loading.value = true
        error.value = ''
        try {
            const payload = await apiRequest('/api/posts')
            posts.value = (payload.posts || []).map(normalizePost)
            if (selectedSlug.value && !posts.value.some((post) => post.slug === selectedSlug.value)) {
                clearSelection()
            }
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
        } finally {
            loading.value = false
        }
    }

    async function selectPost(slug) {
        const uiStore = useUiStore()
        loading.value = true
        error.value = ''
        try {
            draft.value = normalizePost(await apiRequest(`/api/posts/${encodeURIComponent(slug)}`))
            selectedSlug.value = slug
            editorOpen.value = true
            postSetupOpen.value = false
            markDraftClean()
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
        } finally {
            loading.value = false
        }
    }

    function newPost() {
        selectedSlug.value = ''
        draft.value = normalizePost()
        editorOpen.value = false
        postSetupOpen.value = true
        error.value = ''
        markDraftClean()
    }

    function clearSelection() {
        selectedSlug.value = ''
        draft.value = normalizePost()
        editorOpen.value = false
        postSetupOpen.value = false
        markDraftClean()
    }

    function discardDraftChanges() {
        draft.value = JSON.parse(cleanDraftSnapshot)
        isDirty.value = false
    }

    function reset() {
        posts.value = []
        clearSelection()
        error.value = ''
    }

    async function saveDraft() {
        const uiStore = useUiStore()
        saving.value = true
        error.value = ''
        const slug = draft.value.slug.trim()
        const path = selectedSlug.value ? `/api/posts/${encodeURIComponent(selectedSlug.value)}` : '/api/posts'
        try {
            const saved = await apiRequest(path, {
                method: 'POST',
                body: {
                    ...draft.value,
                    slug
                }
            })
            draft.value = normalizePost(saved)
            selectedSlug.value = saved.slug
            editorOpen.value = true
            markDraftClean()
            await loadPosts()
            await selectPost(saved.slug)
            uiStore.setNotice('Post saved.')
            return saved
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            saving.value = false
        }
    }

    async function createPreparedPost() {
        const title = draft.value.title.trim()
        draft.value = normalizePost({
            ...draft.value,
            slug: draft.value.slug.trim(),
            title,
            source: draft.value.source === emptyPost.source ? `# ${title}\n` : draft.value.source
        })
        const saved = await saveDraft()
        postSetupOpen.value = false
        editorOpen.value = true
        return saved
    }

    async function deletePost(slug = selectedSlug.value) {
        const uiStore = useUiStore()
        const deployStore = useDeployStore()
        const postSlug = typeof slug === 'string' ? slug.trim() : ''
        if (!postSlug) {
            return null
        }
        deleting.value = true
        error.value = ''
        try {
            const result = await apiRequest(`/api/posts/${encodeURIComponent(postSlug)}`, {
                method: 'DELETE'
            })
            if (selectedSlug.value === postSlug) {
                clearSelection()
            }
            await loadPosts()
            deployStore.applyBuildResult(result)
            uiStore.setNotice('Post deleted.')
            return result
        } catch (err) {
            error.value = err.message
            uiStore.captureError(err)
            throw err
        } finally {
            deleting.value = false
        }
    }

    async function uploadCover(file) {
        const uiStore = useUiStore()
        if (!canUploadMedia.value || !file) {
            return
        }
        const form = new FormData()
        form.append('file', file)
        uploading.value = true
        try {
            const slug = await ensureMediaPostSaved()
            await apiRequest(`/api/posts/${encodeURIComponent(slug)}/cover`, {
                method: 'POST',
                body: form
            })
            await loadPosts()
            await selectPost(slug)
            uiStore.setNotice('Cover uploaded.')
        } catch (err) {
            uiStore.captureError(err)
            throw err
        } finally {
            uploading.value = false
        }
    }

    async function deleteCover() {
        const uiStore = useUiStore()
        if (!canUploadMedia.value) {
            return
        }
        uploading.value = true
        try {
            const slug = await ensureMediaPostSaved()
            await apiRequest(`/api/posts/${encodeURIComponent(slug)}/cover`, {
                method: 'DELETE'
            })
            await loadPosts()
            await selectPost(slug)
            uiStore.setNotice('Cover removed.')
        } catch (err) {
            uiStore.captureError(err)
            throw err
        } finally {
            uploading.value = false
        }
    }

    async function uploadAsset(file, path) {
        const uiStore = useUiStore()
        if (!canUploadMedia.value || !file) {
            return
        }
        const cleanPath = typeof path === 'string' ? path.trim() : ''
        const form = new FormData()
        form.append('file', file)
        if (cleanPath) {
            form.append('path', cleanPath)
        }
        uploading.value = true
        try {
            const slug = await ensureMediaPostSaved()
            await apiRequest(`/api/posts/${encodeURIComponent(slug)}/assets`, {
                method: 'POST',
                body: form
            })
            await loadPosts()
            await selectPost(slug)
            uiStore.setNotice('Asset uploaded.')
        } catch (err) {
            uiStore.captureError(err)
            throw err
        } finally {
            uploading.value = false
        }
    }

    async function deleteAsset(assetPath) {
        const uiStore = useUiStore()
        if (!canUploadMedia.value || !assetPath) {
            return
        }
        uploading.value = true
        try {
            const slug = await ensureMediaPostSaved()
            const encodedPath = assetPath.split('/').map((part) => encodeURIComponent(part)).join('/')
            await apiRequest(`/api/posts/${encodeURIComponent(slug)}/assets/${encodedPath}`, {
                method: 'DELETE'
            })
            await loadPosts()
            await selectPost(slug)
            uiStore.setNotice('Asset removed.')
        } catch (err) {
            uiStore.captureError(err)
            throw err
        } finally {
            uploading.value = false
        }
    }

    async function ensureMediaPostSaved() {
        if (!selectedSlug.value || isDirty.value) {
            const saved = await saveDraft()
            return saved.slug
        }
        return selectedSlug.value
    }

    function markDraftClean() {
        cleanDraftSnapshot = snapshotPost(draft.value)
        isDirty.value = false
    }

    return {
        posts,
        selectedSlug,
        draft,
        editorOpen,
        postSetupOpen,
        loading,
        saving,
        deleting,
        uploading,
        error,
        isDirty,
        selectedPost,
        canUploadMedia,
        mediaUploadNeedsSave,
        loadPosts,
        selectPost,
        newPost,
        clearSelection,
        discardDraftChanges,
        reset,
        saveDraft,
        deletePost,
        createPreparedPost,
        uploadCover,
        deleteCover,
        uploadAsset,
        deleteAsset
    }
})

function snapshotPost(post) {
    return JSON.stringify(normalizePost(post))
}
