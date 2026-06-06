<script setup>
import { onBeforeUnmount, ref, watch } from 'vue'
import { apiBlobRequest } from '../api/client'

const props = defineProps({
    slug: {
        type: String,
        required: true
    },
    cover: {
        type: String,
        required: true
    },
    alt: {
        type: String,
        default: ''
    },
    compact: {
        type: Boolean,
        default: false
    }
})

const coverUrl = ref('')
const failed = ref(false)
let requestId = 0

watch(
    () => [props.slug, props.cover],
    () => {
        loadCover()
    },
    { immediate: true }
)

async function loadCover() {
    const currentRequest = requestId + 1
    requestId = currentRequest
    failed.value = false
    setCoverUrl('')
    if (!props.slug || !props.cover) {
        return
    }

    try {
        const blob = await apiBlobRequest(`/api/posts/${encodeURIComponent(props.slug)}/cover`)
        if (currentRequest !== requestId) {
            return
        }
        setCoverUrl(URL.createObjectURL(blob))
    } catch {
        if (currentRequest === requestId) {
            failed.value = true
        }
    }
}

function setCoverUrl(nextUrl) {
    if (coverUrl.value) {
        URL.revokeObjectURL(coverUrl.value)
    }
    coverUrl.value = nextUrl
}

onBeforeUnmount(() => {
    requestId += 1
    setCoverUrl('')
})
</script>

<template>
    <div class="post-cover" :class="{ compact, failed }">
        <img v-if="coverUrl" :src="coverUrl" :alt="alt">
    </div>
</template>

<style scoped>
.post-cover {
    width: 100%;
    aspect-ratio: 16 / 9;
    overflow: hidden;
    border: 1px solid var(--color-border);
    border-radius: 8px;
    background: var(--color-surface-muted);
}

.post-cover.compact {
    aspect-ratio: 1;
}

.post-cover.failed {
    border-style: dashed;
}

img {
    display: block;
    width: 100%;
    height: 100%;
    object-fit: cover;
}
</style>
