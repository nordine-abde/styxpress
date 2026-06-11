<script setup>
defineProps({
    title: {
        type: String,
        required: true
    },
    message: {
        type: String,
        default: ''
    }
})
</script>

<template>
    <div class="empty-state">
        <div class="empty-icon" aria-hidden="true">
            <span class="icon-circle icon-circle-lg"></span>
            <span class="icon-circle icon-circle-sm"></span>
            <svg class="icon-tray" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round">
                <path d="M2 12h5l2-3h6l2 3h5" />
                <path d="M19 12v6a2 2 0 0 1-2 2H7a2 2 0 0 1-2-2v-6" />
            </svg>
        </div>
        <div class="empty-content">
            <strong>{{ title }}</strong>
            <p v-if="message">{{ message }}</p>
        </div>
        <div v-if="$slots.default" class="empty-actions">
            <slot />
        </div>
    </div>
</template>

<style scoped>
.empty-state {
    display: grid;
    gap: 0.75rem;
    place-items: center;
    border: 1px dashed color-mix(in srgb, var(--color-border) 80%, var(--color-accent));
    border-radius: 10px;
    padding: 1.8rem 1.2rem;
    background:
        radial-gradient(ellipse at 50% 0%, color-mix(in srgb, var(--color-accent) 4%, transparent) 0%, transparent 70%),
        var(--color-surface-muted);
    text-align: center;
    animation: fade-in 0.4s ease both;
}

/* --- Illustration --- */
.empty-icon {
    position: relative;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 3rem;
    height: 3rem;
}

.icon-circle {
    position: absolute;
    border-radius: 999px;
    background: color-mix(in srgb, var(--color-accent) 10%, transparent);
}

.icon-circle-lg {
    width: 3rem;
    height: 3rem;
    animation: pulse-subtle 3s ease-in-out infinite;
}

.icon-circle-sm {
    width: 2rem;
    height: 2rem;
    background: color-mix(in srgb, var(--color-accent) 16%, transparent);
}

.icon-tray {
    position: relative;
    width: 1.35rem;
    height: 1.35rem;
    color: var(--color-accent-strong);
    opacity: 0.7;
}

/* --- Content --- */
.empty-content {
    display: grid;
    gap: 0.3rem;
    place-items: center;
}

strong {
    color: var(--color-heading);
    font-size: 0.95rem;
    letter-spacing: -0.01em;
}

p {
    margin: 0;
    color: var(--color-muted);
    font-size: 0.88rem;
    line-height: 1.5;
    max-width: 28rem;
}

.empty-actions {
    margin-top: 0.25rem;
}

/* --- Animations --- */
@keyframes fade-in {
    from {
        opacity: 0;
        transform: translateY(4px);
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
}

@keyframes pulse-subtle {
    0%, 100% {
        transform: scale(1);
        opacity: 0.6;
    }
    50% {
        transform: scale(1.08);
        opacity: 1;
    }
}
</style>
