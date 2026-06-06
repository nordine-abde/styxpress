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
let objectUrls = []

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
    return editor.getMarkdown()
}

function queueImageRefresh() {
    window.clearTimeout(imageRefreshTimer)
    imageRefreshTimer = window.setTimeout(() => {
        refreshEditorImages()
    }, 60)
}

async function refreshEditorImages() {
    if (!editorRoot.value || !props.slug) {
        return
    }
    cleanupObjectUrls()
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
        try {
            const blob = await apiBlobRequest(`/api/posts/${encodeURIComponent(props.slug)}/assets/${encodeAssetPath(assetPath)}`)
            const url = URL.createObjectURL(blob)
            objectUrls.push(url)
            image.dataset.styxpressMarkdownSrc = markdownSrc
            image.src = url
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
    image.dataset.styxpressSize = ['small', 'medium', 'large', 'full'].includes(size) ? size : 'medium'
}

function encodeAssetPath(path) {
    return path.split('/').map((part) => encodeURIComponent(part)).join('/')
}

function cleanupObjectUrls() {
    for (const url of objectUrls) {
        URL.revokeObjectURL(url)
    }
    objectUrls = []
}

onBeforeUnmount(() => {
    window.clearTimeout(imageRefreshTimer)
    imageObserver?.disconnect()
    cleanupObjectUrls()
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
