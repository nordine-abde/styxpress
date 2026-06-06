<script setup>
import { computed } from 'vue'
import UiButton from './ui/UiButton.vue'
import UiPanel from './ui/UiPanel.vue'
import { useBuildStore } from '../stores/build'
import { usePostsStore } from '../stores/posts'

const buildStore = useBuildStore()
const postsStore = usePostsStore()

const canRenderPost = computed(() => postsStore.draft.slug && postsStore.draft.publishStatus === 'published')
const canPublishPost = computed(() => Boolean(postsStore.draft.slug))
</script>

<template>
    <UiPanel title="Build" subtitle="Generate local public output from the current content folder.">
        <div class="field-grid">
            <div class="button-row">
                <UiButton
                    tone="ghost"
                    :busy="buildStore.rendering"
                    :disabled="!canRenderPost"
                    @click="buildStore.renderPost(postsStore.draft.slug)"
                >
                    Render post
                </UiButton>
                <UiButton
                    tone="primary"
                    :busy="buildStore.publishing"
                    :disabled="!canPublishPost"
                    @click="buildStore.publishPost(postsStore.draft.slug)"
                >
                    Publish post
                </UiButton>
                <UiButton
                    tone="primary"
                    :busy="buildStore.rendering"
                    @click="buildStore.renderSite"
                >
                    Render site
                </UiButton>
            </div>

            <div v-if="buildStore.lastResult" class="result">
                <p v-if="buildStore.lastResult.post">
                    {{ buildStore.lastResult.post.indexPath }}
                </p>
                <p v-if="buildStore.lastResult.site">
                    {{ buildStore.lastResult.site.indexPath }}
                </p>
                <p v-if="buildStore.lastResult.posts">
                    {{ buildStore.lastResult.posts.length }} posts rendered
                </p>
            </div>

            <p v-if="buildStore.error" class="error-text">
                {{ buildStore.error }}
            </p>
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

.result p {
    margin: 0;
    color: var(--color-muted);
    overflow-wrap: anywhere;
}
</style>
