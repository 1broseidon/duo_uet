// Bulma Interactions - Navbar Burger Toggle and Modal Controls

document.addEventListener('DOMContentLoaded', () => {
    // Highlight active navigation tab based on current URL
    const currentPath = window.location.pathname;
    // Navbar burger toggle for mobile menu
    const navbarBurgers = Array.prototype.slice.call(document.querySelectorAll('.navbar-burger'), 0);

    navbarBurgers.forEach(el => {
        el.addEventListener('click', () => {
            // Get the target from the "data-target" attribute
            const target = el.dataset.target;
            const targetElement = document.getElementById(target);

            // Toggle the "is-active" class on both the "navbar-burger" and the "navbar-menu"
            el.classList.toggle('is-active');
            targetElement.classList.toggle('is-active');
        });
    });

    // Generic modal close functionality
    // This handles any modal with .modal class
    const modals = document.querySelectorAll('.modal');
    
    modals.forEach(modal => {
        const closeButtons = modal.querySelectorAll('.modal-background, .modal-close, .delete');
        
        closeButtons.forEach(button => {
            button.addEventListener('click', () => {
                modal.classList.remove('is-active');
            });
        });
    });

    // Close modals on ESC key
    document.addEventListener('keydown', (event) => {
        if (event.key === 'Escape') {
            const activeModals = document.querySelectorAll('.modal.is-active');
            activeModals.forEach(modal => {
                modal.classList.remove('is-active');
            });
        }
    });

    // Notification delete button handler
    // This will work for dynamically created notifications as well
    document.addEventListener('click', (event) => {
        if (event.target.classList.contains('delete') && 
            event.target.closest('.notification')) {
            const notification = event.target.closest('.notification');
            notification.remove();
        }
    });
});

