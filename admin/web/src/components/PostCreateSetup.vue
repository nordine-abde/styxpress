<script setup>
import { computed, ref, watch } from 'vue'
import UiButton from './ui/UiButton.vue'
import UiField from './ui/UiField.vue'
import UiPanel from './ui/UiPanel.vue'
import { usePostsStore } from '../stores/posts'

const postsStore = usePostsStore()
const slugEdited = ref(false)
const submitted = ref(false)

const title = computed({
    get: () => postsStore.draft.title,
    set: (value) => {
        postsStore.draft.title = value
    }
})

const slug = computed({
    get: () => postsStore.draft.slug,
    set: (value) => {
        slugEdited.value = true
        postsStore.draft.slug = slugFromTitle(value)
    }
})

const titleError = computed(() => {
    return title.value.trim() ? '' : 'Title is required.'
})
const slugError = computed(() => {
    if (!slug.value.trim()) {
        return 'Slug is required.'
    }
    if (!/^[a-z0-9-]+$/.test(slug.value)) {
        return 'Use lowercase letters, numbers, and hyphens.'
    }
    return ''
})
const canCreate = computed(() => !titleError.value && !slugError.value && !postsStore.saving)

watch(
    title,
    (value) => {
        if (!slugEdited.value) {
            postsStore.draft.slug = slugFromTitle(value)
        }
    },
    { immediate: true }
)

async function createPost() {
    submitted.value = true
    if (!canCreate.value) {
        return
    }
    await postsStore.createPreparedPost()
}

function cancel() {
    if (postsStore.isDirty && !window.confirm('Discard this new post?')) {
        return
    }
    postsStore.clearSelection()
}

function slugFromTitle(value) {
    return String(value || '')
        .normalize('NFKD')
        .replace(/[\u0300-\u036f]/g, '')
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, '-')
        .replace(/^-+|-+$/g, '')
        .replace(/-{2,}/g, '-')
}
</script>

<template>
    <div class="post-create-setup">
        <UiPanel title="New post" subtitle="Set the post identity before opening the editor.">
            <form class="field-grid" @submit.prevent="createPost">
                <UiField v-model="title" label="Title" placeholder="Launch notes" />
                <p v-if="submitted && titleError" class="error-text compact-text">
                    {{ titleError }}
                </p>

                <UiField v-model="slug" label="Slug" placeholder="launch-notes" />
                <p v-if="submitted && slugError" class="error-text compact-text">
                    {{ slugError }}
                </p>

                <p v-if="postsStore.error" class="error-text compact-text">
                    {{ postsStore.error }}
                </p>

                <div class="button-row">
                    <UiButton tone="ghost" :disabled="postsStore.saving" @click="cancel">
                        Cancel
                    </UiButton>
                    <UiButton tone="primary" type="submit" :busy="postsStore.saving" :disabled="!canCreate">
                        Start writing
                    </UiButton>
                </div>
            </form>
        </UiPanel>
    </div>
</template>

<style scoped>
.post-create-setup {
    max-width: 44rem;
}

.compact-text {
    margin: 0;
}
</style>
