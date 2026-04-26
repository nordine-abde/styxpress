<script setup>
import { computed, ref, watch } from 'vue'
import EmptyState from './ui/EmptyState.vue'
import UiBadge from './ui/UiBadge.vue'
import UiButton from './ui/UiButton.vue'
import UiPanel from './ui/UiPanel.vue'
import { usePostsStore } from '../stores/posts'

const postsStore = usePostsStore()
const selected = ref([])

const orderedPosts = computed(() => postsStore.posts)

watch(
    () => postsStore.featuredSlugs,
    (slugs) => {
        selected.value = [...slugs]
    },
    { immediate: true }
)

function toggle(slug) {
    if (selected.value.includes(slug)) {
        selected.value = selected.value.filter((value) => value !== slug)
        return
    }
    selected.value = [...selected.value, slug]
}
</script>

<template>
    <UiPanel title="Featured posts" subtitle="The saved order is the order shown on the generated homepage.">
        <EmptyState
            v-if="orderedPosts.length === 0"
            title="No posts available"
            message="Create posts before selecting featured entries."
        />

        <ul v-else class="list">
            <li v-for="post in orderedPosts" :key="post.slug" class="featured-item">
                <label>
                    <input
                        type="checkbox"
                        :checked="selected.includes(post.slug)"
                        @change="toggle(post.slug)"
                    />
                    <span>
                        <strong>{{ post.title }}</strong>
                        <small>{{ post.slug }}</small>
                    </span>
                </label>
                <UiBadge v-if="selected.includes(post.slug)" tone="success">
                    selected
                </UiBadge>
            </li>
        </ul>

        <div class="button-row">
            <UiButton tone="primary" @click="postsStore.saveFeatured(selected)">
                Save featured
            </UiButton>
            <UiButton tone="ghost" @click="postsStore.loadFeatured">
                Reload
            </UiButton>
        </div>
    </UiPanel>
</template>

<style scoped>
.featured-item {
    display: flex;
    flex-wrap: wrap;
    gap: 0.7rem;
    align-items: center;
    justify-content: space-between;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    padding: 0.75rem;
}

label {
    display: flex;
    gap: 0.7rem;
    align-items: center;
}

label span {
    display: grid;
    gap: 0.2rem;
}

strong {
    color: var(--color-heading);
}

small {
    color: var(--color-muted);
}
</style>
