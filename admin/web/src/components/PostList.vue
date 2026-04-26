<script setup>
import EmptyState from './ui/EmptyState.vue'
import LoadingState from './ui/LoadingState.vue'
import UiBadge from './ui/UiBadge.vue'
import UiButton from './ui/UiButton.vue'
import UiPanel from './ui/UiPanel.vue'
import { usePostsStore } from '../stores/posts'

const postsStore = usePostsStore()

function formatDate(value) {
    if (!value) {
        return 'unpublished'
    }
    return new Intl.DateTimeFormat(undefined, {
        dateStyle: 'medium',
        timeStyle: 'short'
    }).format(new Date(value))
}
</script>

<template>
    <UiPanel title="Posts" subtitle="Create, import, and select Markdown posts.">
        <div class="button-row">
            <UiButton tone="primary" @click="postsStore.newPost">
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
            message="Create the first post from the editor."
        />

        <ul v-else class="list">
            <li v-for="post in postsStore.posts" :key="post.slug">
                <button
                    type="button"
                    class="post-item"
                    :class="{ active: post.slug === postsStore.selectedSlug }"
                    @click="postsStore.selectPost(post.slug)"
                >
                    <strong>{{ post.title }}</strong>
                    <span>{{ post.slug }}</span>
                    <small>{{ formatDate(post.publishedAt) }}</small>
                    <UiBadge v-if="postsStore.featuredSlugs.includes(post.slug)" tone="success">
                        featured
                    </UiBadge>
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

.post-item:hover,
.post-item.active {
    border-color: var(--color-accent);
    background: color-mix(in srgb, var(--color-accent) 6%, var(--color-surface));
}

strong {
    color: var(--color-heading);
}

span,
small {
    color: var(--color-muted);
    overflow-wrap: anywhere;
}
</style>
