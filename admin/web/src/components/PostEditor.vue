<script setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref } from 'vue'
import ConfirmPrompt from './ui/ConfirmPrompt.vue'
import FileField from './ui/FileField.vue'
import PostAssetPanel from './PostAssetPanel.vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import UiPanel from './ui/UiPanel.vue'
import VisualMarkdownEditor from './VisualMarkdownEditor.vue'
import { useBuildStore } from '../stores/build'
import { usePostsStore } from '../stores/posts'
import { useUiStore } from '../stores/ui'

const buildStore = useBuildStore()
const postsStore = usePostsStore()
const uiStore = useUiStore()
const editorMode = ref('compose')
const sourceInput = ref(null)
const visualEditor = ref(null)

const saving = computed(() => postsStore.saving || postsStore.deleting || buildStore.publishing)
const modeLabel = computed(() => editorMode.value === 'markdown' ? 'Markdown' : 'Content')
const deleteMessage = computed(() => `Delete "${postsStore.draft.title || postsStore.selectedSlug}" now. The generated site will need deploy afterward.`)

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

async function deletePost() {
    await postsStore.deletePost(postsStore.selectedSlug)
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

onMounted(() => {
    uiStore.registerHeaderSaveAction('post-editor', {
        isAvailable: () => true,
        isDirty: () => postsStore.isDirty,
        isBusy: () => saving.value,
        run: saveAndRenderPost
    })
})

onBeforeUnmount(() => {
    uiStore.unregisterHeaderSaveAction('post-editor')
})
</script>

<template>
    <div class="post-editor-layout">
        <PostAssetPanel @insert-image="insertImage" />

        <div class="post-editor-main">
            <UiPanel title="Post">
                <div class="field-grid">
                    <div class="editor-actions">
                        <div class="editor-actions-left">
                            <UiButton tone="ghost" @click="backToList">
                                Back
                            </UiButton>
                            <ConfirmPrompt
                                v-if="postsStore.selectedSlug"
                                label="Delete"
                                confirm-label="Delete post"
                                :message="deleteMessage"
                                :disabled="saving || postsStore.deleting"
                                @confirm="deletePost"
                            />
                        </div>
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
                        </div>
                    </div>

                    <section class="metadata-section">
                        <h4 class="metadata-heading">
                            <span class="metadata-icon">
                                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M11 4H4a2 2 0 0 0-2 2v14a2 2 0 0 0 2 2h14a2 2 0 0 0 2-2v-7"/><path d="M18.5 2.5a2.121 2.121 0 0 1 3 3L12 15l-4 1 1-4 9.5-9.5z"/></svg>
                            </span>
                            Post metadata
                        </h4>
                        <div class="metadata-fields">
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
                        </div>
                    </section>

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
                            placeholder="Write the post body"
                        />
                        <textarea
                            v-else
                            ref="sourceInput"
                            :value="postsStore.draft.source"
                            class="markdown"
                            rows="22"
                            placeholder="Write the post body"
                            @input="postsStore.draft.source = $event.target.value"
                        ></textarea>
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

.editor-actions-left,
.editor-actions-right {
    display: flex;
    flex-wrap: wrap;
    gap: 1rem;
    align-items: center;
}

.mode-switch {
    display: flex;
    gap: 0.25rem;
    align-items: center;
    border-radius: 8px;
    padding: 0.2rem;
    background: color-mix(in srgb, var(--color-surface-muted) 60%, var(--color-surface));
}

.metadata-section {
    display: grid;
    gap: 0.75rem;
    border: 1px solid color-mix(in srgb, var(--color-border) 72%, white);
    border-radius: 10px;
    padding: 1rem 1.1rem;
    background: linear-gradient(135deg, color-mix(in srgb, var(--color-surface-muted) 24%, var(--color-surface)) 0%, var(--color-surface) 100%);
}

.metadata-heading {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin: 0;
    color: var(--color-heading);
    font-size: 0.82rem;
    font-weight: 800;
    text-transform: uppercase;
    letter-spacing: 0.04em;
}

.metadata-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 1.6rem;
    height: 1.6rem;
    border-radius: 5px;
    background: color-mix(in srgb, var(--color-accent) 12%, var(--color-surface));
    color: var(--color-accent-strong);
}

.metadata-fields {
    display: grid;
    gap: 0.85rem;
}

.source-field {
    display: grid;
    gap: 0.45rem;
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
    border-radius: 10px;
    padding: 1rem 1.1rem;
    background: color-mix(in srgb, var(--color-surface) 92%, white);
    color: var(--color-text);
    font-family: ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    line-height: 1.65;
    resize: vertical;
    transition: border-color 0.2s ease, box-shadow 0.2s ease;
}

textarea.markdown {
    font-family: 'SFMono-Regular', Consolas, 'Liberation Mono', monospace;
    line-height: 1.55;
}

textarea:focus {
    border-color: color-mix(in srgb, var(--color-accent) 72%, var(--color-accent-strong));
    outline: none;
    box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-accent) 16%, transparent);
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

@media (max-width: 1119px) {
    .post-editor-layout > :first-child {
        order: 2;
    }

    .post-editor-main {
        order: 1;
    }
}
</style>
