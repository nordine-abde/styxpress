<script setup>
import { computed } from 'vue'
import UiButton from './ui/UiButton.vue'
import UiBadge from './ui/UiBadge.vue'
import UiPanel from './ui/UiPanel.vue'
import { usePostsStore } from '../stores/posts'
import { usePublishingStore } from '../stores/publishing'

const postsStore = usePostsStore()
const publishingStore = usePublishingStore()

const canRenderPost = computed(() => postsStore.draft.slug && postsStore.draft.publishStatus !== 'draft')
const canVerifyRemote = computed(() => publishingStore.sshStatus === 'ok' && !publishingStore.rendering && !publishingStore.publishing)

function formatDate(value) {
    if (!value) {
        return ''
    }
    return new Intl.DateTimeFormat(undefined, {
        dateStyle: 'medium',
        timeStyle: 'short'
    }).format(new Date(value))
}
</script>

<template>
    <UiPanel title="Publish" subtitle="Drafts publish first; rendering updates public output for published posts.">
        <div class="field-grid">
            <div class="publish-connection">
                <UiBadge :tone="publishingStore.sshStatusTone">
                    {{ publishingStore.sshStatusLabel }}
                </UiBadge>
                <p>SSH passphrase and connection test live in Configuration.</p>
            </div>

            <div class="button-row">
                <UiButton
                    tone="ghost"
                    :busy="publishingStore.rendering"
                    :disabled="!canRenderPost"
                    @click="publishingStore.renderPost(postsStore.draft.slug)"
                >
                    Render
                </UiButton>
                <UiButton
                    tone="primary"
                    :busy="publishingStore.publishing"
                    :disabled="!postsStore.draft.slug"
                    @click="publishingStore.publishPost(postsStore.draft.slug, publishingStore.sshPassphrase)"
                >
                    Publish
                </UiButton>
                <UiButton
                    tone="ghost"
                    :busy="publishingStore.verifying"
                    :disabled="!canVerifyRemote"
                    @click="publishingStore.verifyRemote(publishingStore.sshPassphrase)"
                >
                    Verify remote
                </UiButton>
            </div>

            <div class="result">
                <div class="result-heading">
                    <strong>Remote verification</strong>
                    <UiBadge :tone="publishingStore.verificationStatusTone(publishingStore.siteShellVerification)">
                        {{ publishingStore.verificationStatusLabel(publishingStore.siteShellVerification) }}
                    </UiBadge>
                </div>
                <p>{{ publishingStore.verificationSummaryText }}</p>
                <p v-if="publishingStore.verificationCheckedAt">
                    Checked {{ formatDate(publishingStore.verificationCheckedAt) }}
                </p>
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

.result-heading {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    gap: 0.5rem;
}

p {
    margin: 0;
    color: var(--color-muted);
    overflow-wrap: anywhere;
}

.publish-connection {
    display: grid;
    gap: 0.45rem;
}
</style>
