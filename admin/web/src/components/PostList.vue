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

function shortDate(post) {
    const raw = post.publishedAt || post.updatedAt
    if (!raw) {
        return ''
    }
    const d = new Date(raw)
    return new Intl.DateTimeFormat(undefined, { month: 'short', day: 'numeric' }).format(d)
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
            <li
                v-for="(post, index) in postsStore.posts"
                :key="post.slug"
                :style="{ animationDelay: `${index * 45}ms` }"
                class="post-card-wrapper"
            >
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
                        <div class="post-title-row">
                            <strong>{{ post.title }}</strong>
                            <span v-if="shortDate(post)" class="date-chip">{{ shortDate(post) }}</span>
                        </div>
                        <span class="post-slug">{{ post.slug }}</span>
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
@keyframes post-enter {
    from {
        opacity: 0;
        transform: translateY(8px);
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
}

.post-card-wrapper {
    animation: post-enter 0.32s ease both;
}

.post-item {
    display: grid;
    gap: 0.35rem;
    width: 100%;
    border: 1px solid var(--color-border);
    border-radius: 10px;
    padding: 0.9rem 1rem;
    background: linear-gradient(135deg, var(--color-surface) 0%, color-mix(in srgb, var(--color-surface-muted) 18%, var(--color-surface)) 100%);
    color: var(--color-text);
    text-align: left;
    box-shadow: 0 1px 4px rgb(15 23 42 / 3%);
    transition: transform 0.2s ease, box-shadow 0.2s ease, border-color 0.2s ease;
}

.post-item.has-cover {
    grid-template-columns: 4.5rem minmax(0, 1fr);
    align-items: center;
}

.post-item:hover {
    transform: translateY(-2px);
    border-color: color-mix(in srgb, var(--color-accent) 36%, var(--color-border));
    box-shadow: 0 6px 20px rgb(15 23 42 / 7%), 0 0 0 1px color-mix(in srgb, var(--color-accent) 8%, transparent);
}

.post-item.active {
    border-color: var(--color-accent);
    background: color-mix(in srgb, var(--color-accent) 5%, var(--color-surface));
    box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-accent) 18%, transparent), 0 4px 16px rgb(15 23 42 / 6%);
}

.post-title-row {
    display: flex;
    align-items: center;
    gap: 0.5rem;
}

strong {
    color: var(--color-heading);
    font-size: 0.98rem;
    font-weight: 700;
    letter-spacing: -0.01em;
}

.date-chip {
    flex: 0 0 auto;
    border-radius: 5px;
    padding: 0.15rem 0.5rem;
    background: color-mix(in srgb, var(--color-surface-muted) 72%, var(--color-surface));
    color: var(--color-muted);
    font-size: 0.7rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.03em;
}

.post-details {
    display: grid;
    gap: 0.3rem;
    min-width: 0;
}

.post-slug {
    color: var(--color-muted);
    font-size: 0.82rem;
    overflow-wrap: anywhere;
}

small {
    color: var(--color-muted);
    font-size: 0.78rem;
    overflow-wrap: anywhere;
}

.post-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 0.4rem;
}
</style>
