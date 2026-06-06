import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { apiRequest } from '../api/client'
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
    const loading = ref(false)
    const saving = ref(false)
    const uploading = ref(false)
    const error = ref('')

    const selectedPost = computed(() => posts.value.find((post) => post.slug === selectedSlug.value))
    const hasDraft = computed(() => draft.value.slug.trim() !== '' || draft.value.title.trim() !== '')

    async function loadPosts() {
        const uiStore = useUiStore()
        loading.value = true
        error.value = ''
        try {
            const payload = await apiRequest('/api/posts')
            posts.value = (payload.posts || []).map(normalizePost)
            if (selectedSlug.value && !posts.value.some((post) => post.slug === selectedSlug.value)) {
                selectedSlug.value = ''
            }
            if (!selectedSlug.value && posts.value.length > 0) {
                await selectPost(posts.value[0].slug)
            } else if (!selectedSlug.value) {
                newPost()
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
    }

    function reset() {
        posts.value = []
        selectedSlug.value = ''
        draft.value = normalizePost()
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

    async function uploadCover(file) {
        const uiStore = useUiStore()
        if (!draft.value.slug || !file) {
            return
        }
        const form = new FormData()
        form.append('file', file)
        uploading.value = true
        try {
            await apiRequest(`/api/posts/${encodeURIComponent(draft.value.slug)}/cover`, {
                method: 'POST',
                body: form
            })
            await loadPosts()
            await selectPost(draft.value.slug)
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
        if (!draft.value.slug) {
            return
        }
        uploading.value = true
        try {
            await apiRequest(`/api/posts/${encodeURIComponent(draft.value.slug)}/cover`, {
                method: 'DELETE'
            })
            await loadPosts()
            await selectPost(draft.value.slug)
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
        if (!draft.value.slug || !file) {
            return
        }
        const form = new FormData()
        form.append('file', file)
        if (path.trim()) {
            form.append('path', path.trim())
        }
        uploading.value = true
        try {
            await apiRequest(`/api/posts/${encodeURIComponent(draft.value.slug)}/assets`, {
                method: 'POST',
                body: form
            })
            await loadPosts()
            await selectPost(draft.value.slug)
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
        if (!draft.value.slug || !assetPath) {
            return
        }
        uploading.value = true
        try {
            const encodedPath = assetPath.split('/').map((part) => encodeURIComponent(part)).join('/')
            await apiRequest(`/api/posts/${encodeURIComponent(draft.value.slug)}/assets/${encodedPath}`, {
                method: 'DELETE'
            })
            await loadPosts()
            await selectPost(draft.value.slug)
            uiStore.setNotice('Asset removed.')
        } catch (err) {
            uiStore.captureError(err)
            throw err
        } finally {
            uploading.value = false
        }
    }

    return {
        posts,
        selectedSlug,
        draft,
        loading,
        saving,
        uploading,
        error,
        selectedPost,
        hasDraft,
        loadPosts,
        selectPost,
        newPost,
        reset,
        saveDraft,
        uploadCover,
        deleteCover,
        uploadAsset,
        deleteAsset
    }
})
