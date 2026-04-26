<script setup>
import { computed, ref } from 'vue'
import FileField from './ui/FileField.vue'
import ConfirmPrompt from './ui/ConfirmPrompt.vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import UiPanel from './ui/UiPanel.vue'
import { usePostsStore } from '../stores/posts'
import { usePreviewStore } from '../stores/preview'

const postsStore = usePostsStore()
const previewStore = usePreviewStore()
const assetPath = ref('')
const previewVisible = ref(false)

const canUpload = computed(() => Boolean(postsStore.draft.slug))

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
</script>

<template>
    <UiPanel title="Editor" subtitle="Save a post before uploading cover images or assets.">
        <form class="field-grid" @submit.prevent="postsStore.saveDraft">
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
            <FileField label="Import Markdown" accept=".md,.markdown,text/markdown,text/plain" @selected="importMarkdown" />
            <UiField
                v-model="postsStore.draft.source"
                label="Markdown"
                multiline
                :rows="18"
                placeholder="# Post title"
            />
            <div class="button-row">
                <UiButton tone="primary" type="submit" :busy="postsStore.saving">
                    Save
                </UiButton>
                <UiButton tone="ghost" :busy="previewStore.loading" @click="preview">
                    Preview
                </UiButton>
            </div>
        </form>
    </UiPanel>

    <UiPanel title="Media">
        <div class="field-grid">
            <div class="inline-row">
                <span class="media-label">Cover</span>
                <strong>{{ postsStore.draft.cover || 'none' }}</strong>
            </div>
            <FileField
                label="Upload cover"
                accept=".jpg,.jpeg,.png,.webp,.avif,image/jpeg,image/png,image/webp,image/avif"
                @selected="postsStore.uploadCover"
            />
            <ConfirmPrompt
                v-if="postsStore.draft.cover"
                label="Remove cover"
                confirm-label="Remove"
                @confirm="postsStore.deleteCover"
            />

            <div class="asset-upload">
                <UiField v-model="assetPath" label="Asset path" placeholder="images/diagram.png" />
                <FileField label="Upload asset" @selected="(file) => postsStore.uploadAsset(file, assetPath)" />
            </div>

            <p v-if="!canUpload" class="muted">
                Save the post before uploading files.
            </p>

            <ul v-if="postsStore.draft.assets.length > 0" class="list">
                <li v-for="asset in postsStore.draft.assets" :key="asset" class="asset-item">
                    <code>{{ asset }}</code>
                    <ConfirmPrompt label="Remove" confirm-label="Remove" @confirm="postsStore.deleteAsset(asset)" />
                </li>
            </ul>
        </div>
    </UiPanel>

    <UiPanel v-if="previewVisible" title="Preview">
        <iframe class="preview-frame" title="Rendered preview" :srcdoc="previewStore.html"></iframe>
    </UiPanel>
</template>

<style scoped>
.media-label {
    color: var(--color-muted);
    font-size: 0.86rem;
    font-weight: 800;
}

.asset-upload {
    display: grid;
    gap: 0.75rem;
}

.asset-item {
    display: flex;
    flex-wrap: wrap;
    gap: 0.5rem;
    align-items: center;
    justify-content: space-between;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 0.6rem;
}

code {
    overflow-wrap: anywhere;
}

.preview-frame {
    width: 100%;
    min-height: 32rem;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    background: white;
}
</style>
