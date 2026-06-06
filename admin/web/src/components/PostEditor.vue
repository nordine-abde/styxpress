<script setup>
import { computed, nextTick, ref } from 'vue'
import FileField from './ui/FileField.vue'
import PostAssetPanel from './PostAssetPanel.vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import UiPanel from './ui/UiPanel.vue'
import VisualMarkdownEditor from './VisualMarkdownEditor.vue'
import { useBuildStore } from '../stores/build'
import { usePostsStore } from '../stores/posts'

const buildStore = useBuildStore()
const postsStore = usePostsStore()
const editorMode = ref('compose')
const sourceInput = ref(null)
const visualEditor = ref(null)

const saving = computed(() => postsStore.saving || buildStore.publishing)
const modeLabel = computed(() => editorMode.value === 'markdown' ? 'Markdown' : 'Content')

async function importMarkdown(file) {
    if (!file) {
        return
    }
    postsStore.draft.source = await file.text()
    if (!postsStore.draft.title) {
        postsStore.draft.title = file.name.replace(/\.[^.]+$/, '').replaceAll('-', ' ')
    }
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
    if (editorMode.value === 'compose') {
        visualEditor.value?.insertImage(path, label)
        return
    }
    insertBlock(`![${label}](${path})`)
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
                <div class="field-grid">
                    <div class="editor-actions">
                        <UiButton tone="ghost" @click="backToList">
                            Back to posts
                        </UiButton>
                        <div class="editor-actions-right">
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
                            <UiButton tone="primary" :busy="saving" @click="saveAndRenderPost">
                                Save
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

                    <div class="source-field">
                        <span>{{ modeLabel }}</span>
                        <VisualMarkdownEditor
                            v-if="editorMode === 'compose'"
                            ref="visualEditor"
                            v-model="postsStore.draft.source"
                            :slug="postsStore.selectedSlug"
                            placeholder="# Post title"
                        />
                        <textarea
                            v-else
                            ref="sourceInput"
                            :value="postsStore.draft.source"
                            class="markdown"
                            rows="22"
                            placeholder="# Post title"
                            @input="postsStore.draft.source = $event.target.value"
                        ></textarea>
                    </div>

                    <div class="button-row">
                        <UiButton tone="primary" :busy="saving" @click="saveAndRenderPost">
                            Save
                        </UiButton>
                    </div>
                    <p v-if="buildStore.error" class="error-text compact-text">
                        {{ buildStore.error }}
                    </p>
                </div>
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

.editor-actions-right {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
    align-items: center;
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

@media (min-width: 1120px) {
    .post-editor-layout {
        grid-template-columns: minmax(16rem, 0.36fr) minmax(0, 1fr);
        align-items: start;
    }
}
</style>
