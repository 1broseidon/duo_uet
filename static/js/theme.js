// Theme toggle functionality with system preference detection
(function () {
    const STORAGE_KEY = 'uet-theme';
    const THEME_LIGHT = 'light';
    const THEME_DARK = 'dark';

    // Icon SVGs
    const moonIcon = `<path d="M6 .278a.768.768 0 0 1 .08.858 7.208 7.208 0 0 0-.878 3.46c0 4.021 3.278 7.277 7.318 7.277.527 0 1.04-.055 1.533-.16a.787.787 0 0 1 .81.316.733.733 0 0 1-.031.893A8.349 8.349 0 0 1 8.344 16C3.734 16 0 12.286 0 7.71 0 4.266 2.114 1.312 5.124.06A.752.752 0 0 1 6 .278z"/>`;
    const sunIcon = `<path d="M8 11a3 3 0 1 1 0-6 3 3 0 0 1 0 6zm0 1a4 4 0 1 0 0-8 4 4 0 0 0 0 8zM8 0a.5.5 0 0 1 .5.5v2a.5.5 0 0 1-1 0v-2A.5.5 0 0 1 8 0zm0 13a.5.5 0 0 1 .5.5v2a.5.5 0 0 1-1 0v-2A.5.5 0 0 1 8 13zm8-5a.5.5 0 0 1-.5.5h-2a.5.5 0 0 1 0-1h2a.5.5 0 0 1 .5.5zM3 8a.5.5 0 0 1-.5.5h-2a.5.5 0 0 1 0-1h2A.5.5 0 0 1 3 8zm10.657-5.657a.5.5 0 0 1 0 .707l-1.414 1.415a.5.5 0 1 1-.707-.708l1.414-1.414a.5.5 0 0 1 .707 0zm-9.193 9.193a.5.5 0 0 1 0 .707L3.05 13.657a.5.5 0 0 1-.707-.707l1.414-1.414a.5.5 0 0 1 .707 0zm9.193 2.121a.5.5 0 0 1-.707 0l-1.414-1.414a.5.5 0 0 1 .707-.707l1.414 1.414a.5.5 0 0 1 0 .707zM4.464 4.465a.5.5 0 0 1-.707 0L2.343 3.05a.5.5 0 1 1 .707-.707l1.414 1.414a.5.5 0 0 1 0 .708z"/>`;

    function getSystemTheme() {
        return window.matchMedia('(prefers-color-scheme: dark)').matches ? THEME_DARK : THEME_LIGHT;
    }

    function getStoredTheme() {
        const stored = localStorage.getItem(STORAGE_KEY);
        // Clear legacy "auto" value
        if (stored === 'auto') {
            localStorage.removeItem(STORAGE_KEY);
            return null;
        }
        return stored;
    }

    function setStoredTheme(theme) {
        localStorage.setItem(STORAGE_KEY, theme);
    }

    function getActiveTheme() {
        const stored = getStoredTheme();
        return stored ? stored : getSystemTheme();
    }

    function hasUserPreference() {
        return getStoredTheme() !== null;
    }

    function updateIcon(button, mode, userSelected) {
        const icon = button.querySelector('.theme-icon');
        if (!icon) return;
        
        if (mode === THEME_LIGHT) {
            icon.innerHTML = sunIcon;
            button.title = userSelected ? 'Theme: Light' : 'Theme: Light (system default)';
        } else {
            icon.innerHTML = moonIcon;
            button.title = userSelected ? 'Theme: Dark' : 'Theme: Dark (system default)';
        }
    }

    function applyTheme(theme) {
        document.documentElement.setAttribute('data-theme', theme);
    }

    function initTheme() {
        const button = document.getElementById('theme-toggle');
        if (button) {
            const activeTheme = getActiveTheme();
            applyTheme(activeTheme);
            updateIcon(button, activeTheme, hasUserPreference());
            
            button.addEventListener('click', () => {
                const currentTheme = getActiveTheme();
                const nextTheme = currentTheme === THEME_LIGHT ? THEME_DARK : THEME_LIGHT;

                setStoredTheme(nextTheme);
                applyTheme(nextTheme);
                updateIcon(button, nextTheme, true);
            });
        }
    }

    // Listen for system theme changes when in auto mode
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
        if (!hasUserPreference()) {
            const systemTheme = getSystemTheme();
            applyTheme(systemTheme);
            const button = document.getElementById('theme-toggle');
            if (button) {
                updateIcon(button, systemTheme, false);
            }
        }
    });

    // Initialize on DOMContentLoaded
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initTheme);
    } else {
        initTheme();
    }

    // Apply theme immediately to prevent flash
    applyTheme(getActiveTheme());
})();

