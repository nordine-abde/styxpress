<script setup>
import { computed, nextTick, ref, watch } from 'vue'
import FileField from './ui/FileField.vue'
import MarkdownToolbar from './MarkdownToolbar.vue'
import PostAssetPanel from './PostAssetPanel.vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import UiPanel from './ui/UiPanel.vue'
import { useBuildStore } from '../stores/build'
import { usePostsStore } from '../stores/posts'
import { usePreviewStore } from '../stores/preview'

const buildStore = useBuildStore()
const postsStore = usePostsStore()
const previewStore = usePreviewStore()
const editorMode = ref('compose')
const previewVisible = ref(false)
const sourceInput = ref(null)

const saving = computed(() => postsStore.saving || buildStore.publishing)
const modeLabel = computed(() => editorMode.value === 'markdown' ? 'Markdown' : 'Content')

watch(
    () => postsStore.selectedSlug,
    () => {
        previewVisible.value = false
        previewStore.clearPreview()
    }
)

async function importMarkdown(file) {
    if (!file) {
        return
    }
    postsStore.draft.source = await file.text()
    if (!postsStore.draft.title) {
        postsStore.draft.title = file.name.replace(/\.[^.]+$/, '').replaceAll('-', ' ')
    }
}

async function preview() {
    await previewStore.renderPreview(postsStore.draft)
    previewVisible.value = true
}

async function saveAndRenderPost() {
    const saved = await postsStore.saveDraft()
    await buildStore.publishPost(saved.slug)
}

function backToList() {
    if (postsStore.isDirty && !window.confirm('You have unsaved changes. Leave without saving?')) {
        return
    }
    if (postsStore.isDirty) {
        postsStore.discardDraftChanges()
    }
    postsStore.clearSelection()
}

function insertImage({ asset, size, alt }) {
    const path = `assets/${encodeMarkdownPath(asset)}#${size || 'medium'}`
    const label = alt || asset.split('/').pop()?.replace(/\.[^.]+$/, '') || 'image'
    insertBlock(`![${label}](${path})`)
}

function formatContent(action) {
    if (action.type === 'block') {
        formatBlock(action.value)
        return
    }
    if (action.type === 'bold') {
        wrapSelection('**', '**', 'bold text')
        return
    }
    if (action.type === 'italic') {
        wrapSelection('_', '_', 'italic text')
        return
    }
    if (action.type === 'link') {
        insertLink()
    }
}

function formatBlock(type) {
    const source = currentSource()
    const selection = currentSelection()
    const lineStart = source.lastIndexOf('\n', Math.max(0, selection.start - 1)) + 1
    const nextBreak = source.indexOf('\n', selection.end)
    const lineEnd = nextBreak === -1 ? source.length : nextBreak
    const block = source.slice(lineStart, lineEnd)
    const prefix = blockPrefix(type)
    const formatted = block.split('\n').map((line) => {
        const stripped = line.replace(/^\s*(#{1,6}\s+|>\s+|[-*]\s+)/, '')
        if (!stripped.trim()) {
            return ''
        }
        return `${prefix}${stripped}`
    }).join('\n')
    setSource(
        source.slice(0, lineStart) + formatted + source.slice(lineEnd),
        lineStart,
        lineStart + formatted.length
    )
}

function blockPrefix(type) {
    const prefixes = {
        h1: '# ',
        h2: '## ',
        h3: '### ',
        quote: '> ',
        list: '- '
    }
    return prefixes[type] || ''
}

function wrapSelection(before, after, fallback) {
    const source = currentSource()
    const selection = currentSelection()
    const selected = source.slice(selection.start, selection.end) || fallback
    const next = `${source.slice(0, selection.start)}${before}${selected}${after}${source.slice(selection.end)}`
    const selectedStart = selection.start + before.length
    setSource(next, selectedStart, selectedStart + selected.length)
}

function insertLink() {
    const source = currentSource()
    const selection = currentSelection()
    const selected = source.slice(selection.start, selection.end) || 'link text'
    const link = `[${selected}](https://example.com)`
    const next = source.slice(0, selection.start) + link + source.slice(selection.end)
    const urlStart = selection.start + selected.length + 3
    setSource(next, urlStart, urlStart + 'https://example.com'.length)
}

function insertBlock(markdown) {
    const source = currentSource()
    const selection = currentSelection()
    const leading = selection.start > 0 && !source.slice(0, selection.start).endsWith('\n\n') ? '\n\n' : ''
    const trailing = selection.end < source.length && !source.slice(selection.end).startsWith('\n\n') ? '\n\n' : '\n'
    const insertion = `${leading}${markdown}${trailing}`
    const next = source.slice(0, selection.start) + insertion + source.slice(selection.end)
    const cursor = selection.start + leading.length + markdown.length
    setSource(next, cursor, cursor)
}

function currentSource() {
    return postsStore.draft.source || ''
}

function currentSelection() {
    const source = currentSource()
    const input = sourceInput.value
    if (!input) {
        return {
            start: source.length,
            end: source.length
        }
    }
    return {
        start: input.selectionStart,
        end: input.selectionEnd
    }
}

function setSource(nextSource, selectionStart, selectionEnd) {
    postsStore.draft.source = nextSource
    nextTick(() => {
        if (!sourceInput.value) {
            return
        }
        sourceInput.value.focus()
        sourceInput.value.setSelectionRange(selectionStart, selectionEnd)
    })
}

function encodeMarkdownPath(path) {
    return path.split('/').map((part) => encodeURIComponent(part)).join('/')
}
</script>

<template>
    <div class="post-editor-layout">
        <PostAssetPanel @insert-image="insertImage" />

        <div class="post-editor-main">
            <UiPanel title="Post">
                <form class="field-grid" @submit.prevent="saveAndRenderPost">
                    <div class="editor-actions">
                        <UiButton tone="ghost" @click="backToList">
                            Back to posts
                        </UiButton>
                        <div class="mode-switch" aria-label="Editor mode">
                            <UiButton
                                :tone="editorMode === 'compose' ? 'primary' : 'ghost'"
                                @click="editorMode = 'compose'"
                            >
                                Compose
                            </UiButton>
                            <UiButton
                                :tone="editorMode === 'markdown' ? 'primary' : 'ghost'"
                                @click="editorMode = 'markdown'"
                            >
                                MD
                            </UiButton>
                        </div>
                    </div>

                    <div class="two-column">
                        <UiField
                            v-model="postsStore.draft.slug"
                            label="Slug"
                            placeholder="hello-world"
                            help="Lowercase letters, numbers, and hyphens."
                        />
                        <UiField v-model="postsStore.draft.title" label="Title" />
                    </div>
                    <UiField v-model="postsStore.draft.description" label="Description" />

                    <FileField
                        v-if="editorMode === 'markdown'"
                        label="Import Markdown"
                        accept=".md,.markdown,text/markdown,text/plain"
                        @selected="importMarkdown"
                    />

                    <MarkdownToolbar v-if="editorMode === 'compose'" @format="formatContent" />

                    <label class="source-field">
                        <span>{{ modeLabel }}</span>
                        <textarea
                            ref="sourceInput"
                            :value="postsStore.draft.source"
                            :class="{ markdown: editorMode === 'markdown' }"
                            rows="22"
                            placeholder="# Post title"
                            @input="postsStore.draft.source = $event.target.value"
                        ></textarea>
                    </label>

                    <div class="button-row">
                        <UiButton tone="primary" type="submit" :busy="saving">
                            Save
                        </UiButton>
                        <UiButton tone="ghost" :busy="previewStore.loading" @click="preview">
                            Preview
                        </UiButton>
                    </div>
                    <p v-if="buildStore.error" class="error-text compact-text">
                        {{ buildStore.error }}
                    </p>
                </form>
            </UiPanel>

            <UiPanel v-if="previewVisible" title="Preview">
                <iframe class="preview-frame" title="Rendered preview" :srcdoc="previewStore.html"></iframe>
            </UiPanel>
        </div>
    </div>
</template>

<style scoped>
.post-editor-layout {
    display: grid;
    gap: 1rem;
    min-width: 0;
}

.post-editor-main {
    display: grid;
    gap: 1rem;
    min-width: 0;
}

.editor-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 0.7rem;
    align-items: center;
    justify-content: space-between;
}

.mode-switch {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
    align-items: center;
}

.source-field {
    display: grid;
    gap: 0.35rem;
}

.source-field span {
    color: var(--color-heading);
    font-size: 0.82rem;
    font-weight: 800;
}

textarea {
    width: 100%;
    min-height: 36rem;
    border: 1px solid color-mix(in srgb, var(--color-border) 86%, white);
    border-radius: 8px;
    padding: 0.9rem 1rem;
    background: color-mix(in srgb, var(--color-surface) 92%, white);
    color: var(--color-text);
    font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    line-height: 1.65;
    resize: vertical;
}

textarea.markdown {
    font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', monospace;
    line-height: 1.55;
}

textarea:focus {
    border-color: color-mix(in srgb, var(--color-accent) 72%, var(--color-accent-strong));
    outline: 3px solid color-mix(in srgb, var(--color-accent) 30%, transparent);
}

.compact-text {
    margin: 0;
}

.preview-frame {
    width: 100%;
    min-height: 32rem;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    background: white;
}

@media (min-width: 1120px) {
    .post-editor-layout {
        grid-template-columns: minmax(16rem, 0.36fr) minmax(0, 1fr);
        align-items: start;
    }
}
</style>
