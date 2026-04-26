<script setup>
import { ref } from 'vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import UiPanel from './ui/UiPanel.vue'
import { usePostsStore } from '../stores/posts'
import { usePublishingStore } from '../stores/publishing'

const postsStore = usePostsStore()
const publishingStore = usePublishingStore()
const passphrase = ref('')
</script>

<template>
    <UiPanel title="Publish" subtitle="Rendering updates the local public directory before upload.">
        <div class="field-grid">
            <UiField v-model="passphrase" label="SSH passphrase" type="password" placeholder="Optional" />

            <div class="button-row">
                <UiButton
                    tone="ghost"
                    :busy="publishingStore.rendering"
                    :disabled="!postsStore.draft.slug"
                    @click="publishingStore.renderPost(postsStore.draft.slug)"
                >
                    Render
                </UiButton>
                <UiButton
                    tone="primary"
                    :busy="publishingStore.publishing"
                    :disabled="!postsStore.draft.slug"
                    @click="publishingStore.publishPost(postsStore.draft.slug, passphrase)"
                >
                    Publish
                </UiButton>
            </div>

            <div v-if="publishingStore.lastResult" class="result">
                <strong>Last result</strong>
                <p v-if="publishingStore.lastResult.post">
                    {{ publishingStore.lastResult.post.outputPath }}
                </p>
                <p v-if="publishingStore.lastResult.site">
                    {{ publishingStore.lastResult.site.indexPath }}
                </p>
                <p v-if="publishingStore.lastResult.publish">
                    {{ publishingStore.lastResult.publish.uploadedPaths.length }} uploaded files
                </p>
            </div>
        </div>
    </UiPanel>
</template>

<style scoped>
.result {
    display: grid;
    gap: 0.35rem;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 0.75rem;
    background: var(--color-surface-muted);
}

strong {
    color: var(--color-heading);
}

p {
    margin: 0;
    color: var(--color-muted);
    overflow-wrap: anywhere;
}
</style>
