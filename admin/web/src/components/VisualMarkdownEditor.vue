<script setup>
import '@toast-ui/editor/dist/toastui-editor.css'
import Editor from '@toast-ui/editor'
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { apiBlobRequest } from '../api/client'

const props = defineProps({
    modelValue: {
        type: String,
        default: ''
    },
    slug: {
        type: String,
        default: ''
    },
    placeholder: {
        type: String,
        default: ''
    }
})

const emit = defineEmits(['update:modelValue', 'ready'])

const editorRoot = ref(null)
let editor = null
let syncingFromParent = false
let imageObserver = null
let imageRefreshTimer = 0
let toolbarStateSyncFrame = 0
const imageUrlCache = new Map()
const pendingImageRequests = new Map()
let staleObjectUrls = []

const imageRefreshInterval = 1500

const toolbarButtonStates = [
    { buttonClass: 'heading', stateKey: 'heading' },
    { buttonClass: 'bold', stateKey: 'strong' },
    { buttonClass: 'italic', stateKey: 'emph' },
    { buttonClass: 'bullet-list', stateKey: 'bulletList' },
    { buttonClass: 'ordered-list', stateKey: 'orderedList' },
    { buttonClass: 'quote', stateKey: 'blockQuote' }
]

onMounted(() => {
    editor = new Editor({
        el: editorRoot.value,
        height: 'auto',
        minHeight: '36rem',
        initialEditType: 'wysiwyg',
        initialValue: props.modelValue || '',
        previewStyle: 'tab',
        hideModeSwitch: true,
        usageStatistics: false,
        placeholder: props.placeholder,
        toolbarItems: [
            ['heading', 'bold', 'italic'],
            ['ul', 'ol', 'quote'],
            ['link']
        ],
        events: {
            change: () => {
                if (syncingFromParent) {
                    return
                }
                emitMarkdown()
                queueImageRefresh()
            }
        },
        hooks: {
            addImageBlobHook: () => false
        }
    })
    imageObserver = new MutationObserver(() => {
        queueImageRefresh()
    })
    imageObserver.observe(editorRoot.value, {
        childList: true,
        subtree: true
    })
    attachToolbarStateSync()
    queueImageRefresh()
    emit('ready', {
        insertMarkdown,
        insertImage,
        focus
    })
})

watch(
    () => props.modelValue,
    (value) => {
        if (!editor || value === currentMarkdown()) {
            return
        }
        syncingFromParent = true
        editor.setMarkdown(value || '')
        syncingFromParent = false
        queueImageRefresh()
    }
)

watch(
    () => props.slug,
    () => {
        clearImageUrlCache()
        queueImageRefresh()
    }
)

function insertMarkdown(markdown) {
    if (!editor) {
        return
    }
    restoreMarkdownImageSources()
    editor.replaceSelection(markdown)
    emitMarkdown()
    queueImageRefresh()
}

function insertImage(imageUrl, altText) {
    if (!editor) {
        return
    }
    appendMarkdownBlock(`![${altText || 'image'}](${imageUrl})`)
}

function appendMarkdownBlock(markdown) {
    const current = currentMarkdown().trimEnd()
    syncingFromParent = true
    editor.setMarkdown(`${current}${current ? '\n\n' : ''}${markdown}\n`, true)
    syncingFromParent = false
    emitMarkdown()
    queueImageRefresh()
}

function focus() {
    editor?.focus()
}

function emitMarkdown() {
    emit('update:modelValue', currentMarkdown())
}

function currentMarkdown() {
    if (!editor) {
        return props.modelValue || ''
    }
    restoreMarkdownImageSources()
    const markdown = editor.getMarkdown()
    reapplyCachedImageSources()
    return markdown
}

function queueImageRefresh() {
    window.clearTimeout(imageRefreshTimer)
    imageRefreshTimer = window.setTimeout(() => {
        refreshEditorImages()
    }, 60)
}

function attachToolbarStateSync() {
    const queueFromPayload = ({ toolbarState = {} } = {}) => {
        queueToolbarStateSync(toolbarState)
    }
    const queueCurrentState = () => {
        queueToolbarStateSync()
    }

    editor.eventEmitter.listen('changeToolbarState', queueFromPayload)
    editor.eventEmitter.listen('command', queueCurrentState)
    editor.on('caretChange', queueCurrentState)
    editor.on('change', queueCurrentState)
    editor.on('focus', queueCurrentState)
    editor.on('keyup', queueCurrentState)
    queueToolbarStateSync()
}

function queueToolbarStateSync(toolbarState = {}) {
    window.cancelAnimationFrame(toolbarStateSyncFrame)
    toolbarStateSyncFrame = window.requestAnimationFrame(() => {
        syncToolbarButtonStates(toolbarState)
    })
}

function syncToolbarButtonStates(toolbarState = {}) {
    if (!editorRoot.value) {
        return
    }
    const currentState = {
        ...toolbarState,
        ...currentWysiwygToolbarState()
    }

    for (const { buttonClass, stateKey } of toolbarButtonStates) {
        const active = Boolean(currentState[stateKey]?.active)
        const buttons = editorRoot.value.querySelectorAll(`.toastui-editor-toolbar-icons.${buttonClass}`)
        for (const button of buttons) {
            button.classList.toggle('active', active)
            button.setAttribute('aria-pressed', active ? 'true' : 'false')
        }
    }
}

function currentWysiwygToolbarState() {
    const state = editor?.wwEditor?.view?.state
    if (!state || !editor.isWysiwygMode()) {
        return {}
    }

    // Toast UI omits ProseMirror stored marks from its toolbar state.
    const markNames = activeMarkNames(state)
    const nodeNames = activeNodeNames(state.selection.$from)

    return {
        strong: { active: markNames.has('strong') },
        emph: { active: markNames.has('emph') },
        heading: { active: nodeNames.has('heading') },
        bulletList: { active: nodeNames.has('bulletList') },
        orderedList: { active: nodeNames.has('orderedList') },
        blockQuote: { active: nodeNames.has('blockQuote') }
    }
}

function activeMarkNames(state) {
    const { selection, storedMarks } = state
    const marks = selection.empty
        ? storedMarks || selection.$from.marks()
        : selection.$from.marksAcross(selection.$to) || []

    return new Set(marks.map((mark) => mark.type.name))
}

function activeNodeNames(resolvedPosition) {
    const names = new Set()

    for (let depth = resolvedPosition.depth; depth > 0; depth -= 1) {
        const node = resolvedPosition.node(depth)
        const parent = depth > 0 ? resolvedPosition.node(depth - 1) : null
        const nodeName = node.type.name

        names.add(nodeName)

        if (nodeName === 'listItem' && parent?.type?.name) {
            names.add(node.attrs?.task ? 'taskList' : parent.type.name)
        }
    }

    return names
}

async function refreshEditorImages() {
    if (!editorRoot.value || !props.slug) {
        return
    }
    const images = Array.from(editorRoot.value.querySelectorAll('.toastui-editor-contents img'))
    for (const image of images) {
        const markdownSrc = originalImageSrc(image)
        if (!markdownSrc) {
            continue
        }
        applyImageSize(image, markdownSrc)
        const assetPath = assetPathFromMarkdownSrc(markdownSrc)
        if (!assetPath) {
            continue
        }
        const key = imageCacheKey(assetPath)
        image.dataset.styxpressMarkdownSrc = markdownSrc
        image.dataset.styxpressCacheKey = key
        const cached = imageUrlCache.get(key)
        if (cached?.url) {
            setImageSource(image, cached.url)
            if (Date.now() - cached.fetchedAt < imageRefreshInterval) {
                continue
            }
        } else {
            reserveImageBox(image)
        }
        try {
            const url = await fetchEditorImageURL(key, assetPath)
            applyCachedImageToMatchingImages(key, url)
        } catch {
            image.dataset.styxpressFailed = 'true'
        }
    }
}

function restoreMarkdownImageSources() {
    if (!editorRoot.value) {
        return
    }
    const images = editorRoot.value.querySelectorAll('img[data-styxpress-markdown-src]')
    for (const image of images) {
        image.src = image.dataset.styxpressMarkdownSrc
    }
}

function reapplyCachedImageSources() {
    if (!editorRoot.value || !props.slug) {
        return
    }
    const images = editorRoot.value.querySelectorAll('img[data-styxpress-markdown-src]')
    for (const image of images) {
        const markdownSrc = image.dataset.styxpressMarkdownSrc
        const assetPath = assetPathFromMarkdownSrc(markdownSrc)
        if (!assetPath) {
            continue
        }
        const cached = imageUrlCache.get(imageCacheKey(assetPath))
        if (cached?.url) {
            setImageSource(image, cached.url)
        }
    }
}

function originalImageSrc(image) {
    const stored = image.dataset.styxpressMarkdownSrc
    if (stored) {
        return stored
    }
    const raw = image.getAttribute('src') || ''
    if (raw.startsWith('blob:')) {
        return ''
    }
    return raw
}

function assetPathFromMarkdownSrc(src) {
    const [path] = src.split('#')
    const normalized = decodeURI(path).replace(/^\.\//, '')
    if (!normalized.startsWith('assets/')) {
        return ''
    }
    return normalized.slice('assets/'.length)
}

function applyImageSize(image, src) {
    const size = src.includes('#') ? src.split('#').pop() : 'medium'
    const normalized = ['small', 'medium', 'large', 'full'].includes(size) ? size : 'medium'
    image.dataset.styxpressSize = normalized
    image.style.setProperty('--styxpress-image-fallback-height', fallbackImageHeight(normalized))
}

function encodeAssetPath(path) {
    return path.split('/').map((part) => encodeURIComponent(part)).join('/')
}

function imageCacheKey(assetPath) {
    return `${props.slug}:${assetPath}`
}

async function fetchEditorImageURL(key, assetPath) {
    const pending = pendingImageRequests.get(key)
    if (pending) {
        return pending
    }
    const slug = props.slug
    const request = apiBlobRequest(`/api/posts/${encodeURIComponent(slug)}/assets/${encodeAssetPath(assetPath)}`)
        .then(async (blob) => {
            const url = URL.createObjectURL(blob)
            await preloadImage(url)
            if (slug !== props.slug) {
                URL.revokeObjectURL(url)
                return ''
            }
            const previous = imageUrlCache.get(key)
            imageUrlCache.set(key, {
                url,
                fetchedAt: Date.now()
            })
            if (previous?.url && previous.url !== url) {
                staleObjectUrls.push(previous.url)
                scheduleStaleObjectUrlCleanup()
            }
            return url
        })
        .finally(() => {
            pendingImageRequests.delete(key)
        })
    pendingImageRequests.set(key, request)
    return request
}

function applyCachedImageToMatchingImages(key, url) {
    if (!editorRoot.value || !url) {
        return
    }
    const images = Array.from(editorRoot.value.querySelectorAll('img[data-styxpress-cache-key]'))
        .filter((image) => image.dataset.styxpressCacheKey === key)
    for (const image of images) {
        setImageSource(image, url)
    }
}

function setImageSource(image, url) {
    if (image.getAttribute('src') === url) {
        releaseImageBox(image)
        return
    }
    reserveImageBox(image)
    image.addEventListener('load', () => releaseImageBox(image), { once: true })
    image.src = url
    if (image.complete) {
        releaseImageBox(image)
    }
}

function reserveImageBox(image) {
    const rect = image.getBoundingClientRect()
    if (rect.height > 24) {
        image.style.setProperty('--styxpress-image-reserved-height', `${Math.ceil(rect.height)}px`)
    }
    image.dataset.styxpressLoading = 'true'
}

function releaseImageBox(image) {
    if (!image.complete && image.naturalWidth === 0) {
        return
    }
    const rect = image.getBoundingClientRect()
    if (rect.height > 24) {
        image.style.setProperty('--styxpress-image-reserved-height', `${Math.ceil(rect.height)}px`)
    }
    delete image.dataset.styxpressLoading
    delete image.dataset.styxpressFailed
}

function fallbackImageHeight(size) {
    return {
        small: '10rem',
        medium: '14rem',
        large: '18rem',
        full: '20rem'
    }[size] || '14rem'
}

function preloadImage(url) {
    return new Promise((resolve, reject) => {
        const image = new Image()
        image.onload = () => resolve()
        image.onerror = reject
        image.src = url
    })
}

function scheduleStaleObjectUrlCleanup() {
    window.setTimeout(() => {
        for (const url of staleObjectUrls) {
            URL.revokeObjectURL(url)
        }
        staleObjectUrls = []
    }, 1200)
}

function clearImageUrlCache() {
    for (const { url } of imageUrlCache.values()) {
        URL.revokeObjectURL(url)
    }
    for (const url of staleObjectUrls) {
        URL.revokeObjectURL(url)
    }
    imageUrlCache.clear()
    pendingImageRequests.clear()
    staleObjectUrls = []
}

onBeforeUnmount(() => {
    window.clearTimeout(imageRefreshTimer)
    window.cancelAnimationFrame(toolbarStateSyncFrame)
    imageObserver?.disconnect()
    clearImageUrlCache()
    editor?.destroy()
})

defineExpose({
    insertMarkdown,
    insertImage,
    focus
})
</script>

<template>
    <div ref="editorRoot" class="visual-markdown-editor"></div>
</template>

<style scoped>
.visual-markdown-editor {
    min-width: 0;
}

.visual-markdown-editor :deep(.toastui-editor-defaultUI) {
    overflow: hidden;
    border-color: color-mix(in srgb, var(--color-border) 86%, white);
    border-radius: 8px;
    background: color-mix(in srgb, var(--color-surface) 92%, white);
}

.visual-markdown-editor :deep(.toastui-editor-toolbar) {
    background: color-mix(in srgb, var(--color-surface-muted) 62%, white);
}

.visual-markdown-editor :deep(.toastui-editor-toolbar-icons) {
    border-radius: 6px;
}

.visual-markdown-editor :deep(.toastui-editor-toolbar-icons.active),
.visual-markdown-editor :deep(.toastui-editor-toolbar-icons:not(:disabled).active) {
    background-color: color-mix(in srgb, var(--color-accent) 20%, transparent);
    box-shadow: inset 0 0 0 2px var(--color-accent);
}

.visual-markdown-editor :deep(.toastui-editor-contents) {
    color: var(--color-text);
    font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    font-size: 1rem;
}

.visual-markdown-editor :deep(.toastui-editor-contents img) {
    display: block;
    width: auto;
    max-width: min(100%, 32rem);
    max-height: 30rem;
    margin-right: auto;
    margin-left: auto;
    border-radius: 8px;
    object-fit: contain;
}

.visual-markdown-editor :deep(.toastui-editor-contents img[data-styxpress-loading='true']) {
    min-height: var(--styxpress-image-reserved-height, var(--styxpress-image-fallback-height, 14rem));
    background: color-mix(in srgb, var(--color-surface-muted) 72%, white);
}

.visual-markdown-editor :deep(.toastui-editor-contents img[data-styxpress-size='small']) {
    max-width: min(100%, 18rem);
    max-height: 18rem;
}

.visual-markdown-editor :deep(.toastui-editor-contents img[data-styxpress-size='large']) {
    max-width: min(100%, 44rem);
    max-height: 36rem;
}

.visual-markdown-editor :deep(.toastui-editor-contents img[data-styxpress-size='full']) {
    width: 100%;
    max-width: 100%;
    max-height: 42rem;
}
</style>
