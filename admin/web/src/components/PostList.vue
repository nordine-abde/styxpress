<script setup>
import EmptyState from './ui/EmptyState.vue'
import LoadingState from './ui/LoadingState.vue'
import PostCoverImage from './PostCoverImage.vue'
import UiBadge from './ui/UiBadge.vue'
import UiButton from './ui/UiButton.vue'
import UiPanel from './ui/UiPanel.vue'
import { usePostsStore } from '../stores/posts'

const postsStore = usePostsStore()

function formatDate(value) {
    if (!value) {
        return ''
    }
    return new Intl.DateTimeFormat(undefined, {
        dateStyle: 'medium',
        timeStyle: 'short'
    }).format(new Date(value))
}

function statusTone(post) {
    return post.publishStatus === 'published' ? 'success' : 'neutral'
}

function statusLabel(post) {
    return post.publishStatus === 'published' ? 'Published' : 'Draft'
}

function dateLabel(post) {
    if (post.publishedAt) {
        return `Published ${formatDate(post.publishedAt)}`
    }
    if (post.updatedAt) {
        return `Updated ${formatDate(post.updatedAt)}`
    }
    return 'Not published'
}

function startNewPost() {
    if (!confirmDiscardDraft()) {
        return
    }
    postsStore.newPost()
}

function selectPost(slug) {
    if (slug === postsStore.selectedSlug) {
        return
    }
    if (!confirmDiscardDraft()) {
        return
    }
    postsStore.selectPost(slug)
}

function confirmDiscardDraft() {
    if (!postsStore.isDirty) {
        return true
    }
    const confirmed = window.confirm('You have unsaved changes. Leave without saving?')
    if (confirmed) {
        postsStore.discardDraftChanges()
    }
    return confirmed
}
</script>

<template>
    <UiPanel title="Posts" subtitle="Create, import, and select Markdown posts.">
        <div class="button-row">
            <UiButton tone="primary" @click="startNewPost">
                New
            </UiButton>
            <UiButton tone="ghost" :busy="postsStore.loading" @click="postsStore.loadPosts">
                Reload
            </UiButton>
        </div>

        <LoadingState v-if="postsStore.loading">Loading posts</LoadingState>

        <EmptyState
            v-else-if="postsStore.posts.length === 0"
            title="No posts yet"
            message="Create the first post from this list."
        />

        <ul v-else class="list">
            <li v-for="post in postsStore.posts" :key="post.slug">
                <button
                    type="button"
                    class="post-item"
                    :class="{ active: post.slug === postsStore.selectedSlug, 'has-cover': post.cover }"
                    @click="selectPost(post.slug)"
                >
                    <PostCoverImage
                        v-if="post.cover"
                        :slug="post.slug"
                        :cover="post.cover"
                        :alt="`${post.title} cover`"
                        compact
                    />
                    <div class="post-details">
                        <strong>{{ post.title }}</strong>
                        <span>{{ post.slug }}</span>
                        <div class="post-meta">
                            <UiBadge :tone="statusTone(post)">
                                {{ statusLabel(post) }}
                            </UiBadge>
                        </div>
                        <small>{{ dateLabel(post) }}</small>
                    </div>
                </button>
            </li>
        </ul>
    </UiPanel>
</template>

<style scoped>
.post-item {
    display: grid;
    gap: 0.25rem;
    width: 100%;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 0.8rem;
    background: var(--color-surface);
    color: var(--color-text);
    text-align: left;
}

.post-item.has-cover {
    grid-template-columns: 4.5rem minmax(0, 1fr);
    align-items: center;
}

.post-item:hover,
.post-item.active {
    border-color: var(--color-accent);
    background: color-mix(in srgb, var(--color-accent) 6%, var(--color-surface));
}

strong {
    color: var(--color-heading);
}

.post-details {
    display: grid;
    gap: 0.25rem;
    min-width: 0;
}

span,
small {
    color: var(--color-muted);
    overflow-wrap: anywhere;
}

.post-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
}
</style>
