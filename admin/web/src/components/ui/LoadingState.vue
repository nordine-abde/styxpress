<template>
    <div class="loading-state">
        <div class="spinner-wrap" aria-hidden="true">
            <span class="ring ring-outer"></span>
            <span class="ring ring-inner"></span>
        </div>
        <span class="loading-text">
            <slot>Loading</slot>
        </span>
    </div>
</template>

<style scoped>
.loading-state {
    display: inline-flex;
    align-items: center;
    gap: 0.6rem;
    color: var(--color-muted);
    font-size: 0.9rem;
    animation: fade-in 0.35s ease both;
}

.spinner-wrap {
    position: relative;
    width: 1.1rem;
    height: 1.1rem;
    flex-shrink: 0;
}

.ring {
    position: absolute;
    inset: 0;
    border-radius: 999px;
}

.ring-outer {
    border: 2px solid color-mix(in srgb, var(--color-muted) 22%, transparent);
    border-top-color: var(--color-accent);
    animation: spin 0.7s linear infinite;
}

.ring-inner {
    inset: 3px;
    border: 1.5px solid color-mix(in srgb, var(--color-accent) 18%, transparent);
    animation: pulse-ring 1.6s ease-in-out infinite;
}

.loading-text {
    font-weight: 600;
    letter-spacing: 0.01em;
}

@keyframes spin {
    to {
        transform: rotate(360deg);
    }
}

@keyframes pulse-ring {
    0%, 100% {
        opacity: 0.35;
        transform: scale(0.92);
    }
    50% {
        opacity: 0.7;
        transform: scale(1);
    }
}

@keyframes fade-in {
    from {
        opacity: 0;
        transform: translateY(2px);
    }
    to {
        opacity: 1;
        transform: translateY(0);
    }
}
</style>
