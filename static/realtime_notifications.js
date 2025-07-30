// Real-time Notification System
class RealtimeNotifications {
    constructor() {
        this.socket = null;
        this.reconnectInterval = 5000;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 10;
        this.notifications = [];
        this.unreadCount = 0;
        this.isConnected = false;
        
        // UI elements
        this.notificationBell = null;
        this.notificationPanel = null;
        this.notificationList = null;
        this.unreadBadge = null;
        
        this.init();
    }
    
    init() {
        // Create notification UI
        this.createNotificationUI();
        
        // Connect to WebSocket
        this.connect();
        
        // Load existing notifications
        this.loadNotifications();
        
        // Set up event listeners
        this.setupEventListeners();
    }
    
    createNotificationUI() {
        // Create notification bell in navbar
        const navbar = document.querySelector('.navbar-nav') || document.querySelector('.navbar');
        if (!navbar) return;
        
        const notificationItem = document.createElement('div');
        notificationItem.className = 'notification-container';
        notificationItem.innerHTML = `
            <button class="btn btn-link position-relative" id="notificationBell">
                <i class="bi bi-bell" style="font-size: 1.5rem; color: white;"></i>
                <span class="position-absolute top-0 start-100 translate-middle badge rounded-pill bg-danger" 
                      id="unreadBadge" style="display: none;">
                    <span id="unreadCount">0</span>
                </span>
            </button>
            
            <div class="notification-panel" id="notificationPanel" style="display: none;">
                <div class="notification-header">
                    <h5>Notifications</h5>
                    <button class="btn btn-sm btn-link" onclick="notificationSystem.markAllAsRead()">
                        Mark all as read
                    </button>
                </div>
                <div class="notification-list" id="notificationList">
                    <div class="notification-empty">
                        <i class="bi bi-bell-slash"></i>
                        <p>No notifications</p>
                    </div>
                </div>
                <div class="notification-footer">
                    <a href="/notification-history" class="btn btn-sm btn-link">
                        View all notifications
                    </a>
                </div>
            </div>
        `;
        
        // Insert before the last nav item
        const lastItem = navbar.lastElementChild;
        navbar.insertBefore(notificationItem, lastItem);
        
        // Store references
        this.notificationBell = document.getElementById('notificationBell');
        this.notificationPanel = document.getElementById('notificationPanel');
        this.notificationList = document.getElementById('notificationList');
        this.unreadBadge = document.getElementById('unreadBadge');
        
        // Add styles
        this.addStyles();
    }
    
    addStyles() {
        const style = document.createElement('style');
        style.textContent = `
            .notification-container {
                position: relative;
                margin-right: 1rem;
            }
            
            .notification-panel {
                position: absolute;
                top: 100%;
                right: 0;
                width: 380px;
                max-height: 500px;
                background: white;
                border-radius: 10px;
                box-shadow: 0 10px 30px rgba(0, 0, 0, 0.2);
                z-index: 1000;
                margin-top: 10px;
                overflow: hidden;
            }
            
            .notification-header {
                padding: 1rem;
                border-bottom: 1px solid #e9ecef;
                display: flex;
                justify-content: space-between;
                align-items: center;
            }
            
            .notification-header h5 {
                margin: 0;
                font-size: 1.1rem;
                font-weight: 600;
            }
            
            .notification-list {
                max-height: 350px;
                overflow-y: auto;
                background: #f8f9fa;
            }
            
            .notification-item {
                padding: 1rem;
                border-bottom: 1px solid #e9ecef;
                background: white;
                cursor: pointer;
                transition: background 0.2s;
            }
            
            .notification-item:hover {
                background: #f8f9fa;
            }
            
            .notification-item.unread {
                background: #e8f0fe;
                border-left: 4px solid #1976d2;
            }
            
            .notification-item.unread:hover {
                background: #daebff;
            }
            
            .notification-type {
                display: inline-block;
                width: 40px;
                height: 40px;
                border-radius: 50%;
                text-align: center;
                line-height: 40px;
                font-size: 1.2rem;
                margin-right: 1rem;
                float: left;
            }
            
            .notification-type.maintenance {
                background: #fff3cd;
                color: #856404;
            }
            
            .notification-type.route {
                background: #d1ecf1;
                color: #0c5460;
            }
            
            .notification-type.emergency {
                background: #f8d7da;
                color: #721c24;
            }
            
            .notification-type.system {
                background: #e2e3e5;
                color: #383d41;
            }
            
            .notification-content {
                overflow: hidden;
            }
            
            .notification-title {
                font-weight: 600;
                margin-bottom: 0.25rem;
                color: #2c3e50;
            }
            
            .notification-message {
                font-size: 0.875rem;
                color: #6c757d;
                margin-bottom: 0.25rem;
            }
            
            .notification-time {
                font-size: 0.75rem;
                color: #adb5bd;
            }
            
            .notification-empty {
                padding: 3rem;
                text-align: center;
                color: #6c757d;
            }
            
            .notification-empty i {
                font-size: 3rem;
                margin-bottom: 1rem;
                display: block;
            }
            
            .notification-footer {
                padding: 0.75rem;
                text-align: center;
                border-top: 1px solid #e9ecef;
                background: white;
            }
            
            @keyframes notificationSlideIn {
                from {
                    transform: translateX(100%);
                    opacity: 0;
                }
                to {
                    transform: translateX(0);
                    opacity: 1;
                }
            }
            
            .notification-toast {
                position: fixed;
                top: 80px;
                right: 20px;
                background: white;
                border-radius: 10px;
                box-shadow: 0 5px 20px rgba(0, 0, 0, 0.2);
                padding: 1rem 1.5rem;
                min-width: 300px;
                max-width: 400px;
                animation: notificationSlideIn 0.3s ease-out;
                z-index: 2000;
            }
            
            .notification-toast.success {
                border-left: 4px solid #28a745;
            }
            
            .notification-toast.warning {
                border-left: 4px solid #ffc107;
            }
            
            .notification-toast.error {
                border-left: 4px solid #dc3545;
            }
            
            .notification-toast.info {
                border-left: 4px solid #17a2b8;
            }
            
            .notification-toast-close {
                position: absolute;
                top: 0.5rem;
                right: 0.5rem;
                background: none;
                border: none;
                font-size: 1.25rem;
                cursor: pointer;
                color: #6c757d;
            }
            
            .notification-toast-close:hover {
                color: #495057;
            }
        `;
        document.head.appendChild(style);
    }
    
    setupEventListeners() {
        // Toggle notification panel
        if (this.notificationBell) {
            this.notificationBell.addEventListener('click', (e) => {
                e.stopPropagation();
                this.togglePanel();
            });
        }
        
        // Close panel when clicking outside
        document.addEventListener('click', (e) => {
            if (this.notificationPanel && 
                this.notificationPanel.style.display !== 'none' &&
                !this.notificationPanel.contains(e.target)) {
                this.closePanel();
            }
        });
    }
    
    connect() {
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;
        
        try {
            this.socket = new WebSocket(wsUrl);
            
            this.socket.onopen = () => {
                console.log('WebSocket connected for notifications');
                this.isConnected = true;
                this.reconnectAttempts = 0;
                
                // Subscribe to notifications
                this.socket.send(JSON.stringify({
                    type: 'subscribe',
                    data: { channel: 'notifications' }
                }));
            };
            
            this.socket.onmessage = (event) => {
                try {
                    const message = JSON.parse(event.data);
                    this.handleMessage(message);
                } catch (error) {
                    console.error('Failed to parse WebSocket message:', error);
                }
            };
            
            this.socket.onclose = () => {
                console.log('WebSocket disconnected');
                this.isConnected = false;
                this.reconnect();
            };
            
            this.socket.onerror = (error) => {
                console.error('WebSocket error:', error);
            };
            
        } catch (error) {
            console.error('Failed to connect WebSocket:', error);
            this.reconnect();
        }
    }
    
    reconnect() {
        if (this.reconnectAttempts >= this.maxReconnectAttempts) {
            console.error('Max reconnection attempts reached');
            return;
        }
        
        this.reconnectAttempts++;
        setTimeout(() => {
            console.log(`Attempting to reconnect... (${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
            this.connect();
        }, this.reconnectInterval);
    }
    
    handleMessage(message) {
        switch (message.type) {
            case 'notification':
                this.handleNotification(message.data);
                break;
            case 'unread_count':
                this.updateUnreadCount(message.data.count);
                break;
            case 'notification_read':
                this.markNotificationAsRead(message.data.notification_id);
                break;
        }
    }
    
    handleNotification(notification) {
        // Add to notifications array
        this.notifications.unshift(notification);
        
        // Update UI
        this.addNotificationToList(notification);
        
        // Update unread count
        this.unreadCount++;
        this.updateUnreadBadge();
        
        // Show toast notification
        this.showToast(notification);
        
        // Play notification sound
        this.playNotificationSound();
        
        // Request browser notification permission
        this.showBrowserNotification(notification);
    }
    
    addNotificationToList(notification) {
        if (!this.notificationList) return;
        
        // Remove empty state if present
        const emptyState = this.notificationList.querySelector('.notification-empty');
        if (emptyState) {
            emptyState.remove();
        }
        
        // Create notification element
        const notificationEl = document.createElement('div');
        notificationEl.className = 'notification-item unread';
        notificationEl.dataset.notificationId = notification.id;
        
        const typeIcon = this.getNotificationIcon(notification.type);
        const timeAgo = this.formatTimeAgo(notification.created_at);
        
        notificationEl.innerHTML = `
            <div class="notification-type ${notification.type}">
                <i class="${typeIcon}"></i>
            </div>
            <div class="notification-content">
                <div class="notification-title">${notification.subject}</div>
                <div class="notification-message">${notification.message}</div>
                <div class="notification-time">${timeAgo}</div>
            </div>
        `;
        
        // Add click handler
        notificationEl.addEventListener('click', () => {
            this.handleNotificationClick(notification);
        });
        
        // Insert at the beginning
        this.notificationList.insertBefore(notificationEl, this.notificationList.firstChild);
        
        // Limit displayed notifications
        const maxDisplayed = 10;
        const items = this.notificationList.querySelectorAll('.notification-item');
        if (items.length > maxDisplayed) {
            items[items.length - 1].remove();
        }
    }
    
    getNotificationIcon(type) {
        const icons = {
            'maintenance': 'bi-tools',
            'route': 'bi-map',
            'emergency': 'bi-exclamation-triangle-fill',
            'system': 'bi-gear',
            'message': 'bi-chat-dots',
            'alert': 'bi-bell-fill',
            'driver': 'bi-person-badge'
        };
        return icons[type] || 'bi-bell';
    }
    
    formatTimeAgo(timestamp) {
        const date = new Date(timestamp);
        const now = new Date();
        const seconds = Math.floor((now - date) / 1000);
        
        if (seconds < 60) return 'Just now';
        if (seconds < 3600) return `${Math.floor(seconds / 60)} minutes ago`;
        if (seconds < 86400) return `${Math.floor(seconds / 3600)} hours ago`;
        if (seconds < 604800) return `${Math.floor(seconds / 86400)} days ago`;
        
        return date.toLocaleDateString();
    }
    
    showToast(notification) {
        const toast = document.createElement('div');
        toast.className = `notification-toast ${notification.priority || 'info'}`;
        
        const typeIcon = this.getNotificationIcon(notification.type);
        
        toast.innerHTML = `
            <button class="notification-toast-close" onclick="this.parentElement.remove()">
                <i class="bi bi-x"></i>
            </button>
            <div style="display: flex; align-items: start;">
                <div style="margin-right: 1rem; font-size: 1.5rem;">
                    <i class="${typeIcon}"></i>
                </div>
                <div>
                    <div style="font-weight: 600; margin-bottom: 0.25rem;">
                        ${notification.subject}
                    </div>
                    <div style="font-size: 0.875rem; color: #6c757d;">
                        ${notification.message}
                    </div>
                </div>
            </div>
        `;
        
        document.body.appendChild(toast);
        
        // Auto-remove after 5 seconds
        setTimeout(() => {
            toast.remove();
        }, 5000);
    }
    
    playNotificationSound() {
        // Create and play notification sound
        try {
            const audio = new Audio('/static/notification.mp3');
            audio.volume = 0.5;
            audio.play().catch(e => console.log('Could not play notification sound:', e));
        } catch (e) {
            console.log('Notification sound not available');
        }
    }
    
    showBrowserNotification(notification) {
        // Check if browser supports notifications
        if (!('Notification' in window)) return;
        
        // Request permission if needed
        if (Notification.permission === 'default') {
            Notification.requestPermission();
        }
        
        // Show notification if permitted
        if (Notification.permission === 'granted') {
            const options = {
                body: notification.message,
                icon: '/static/images/notification-icon.png',
                badge: '/static/images/notification-badge.png',
                tag: notification.id,
                requireInteraction: notification.priority === 'high'
            };
            
            const browserNotif = new Notification(notification.subject, options);
            
            browserNotif.onclick = () => {
                window.focus();
                this.handleNotificationClick(notification);
                browserNotif.close();
            };
        }
    }
    
    handleNotificationClick(notification) {
        // Mark as read
        this.markAsRead(notification.id);
        
        // Navigate based on notification type
        switch (notification.type) {
            case 'maintenance':
                if (notification.data && notification.data.vehicle_id) {
                    window.location.href = `/vehicle-maintenance/${notification.data.vehicle_id}`;
                }
                break;
            case 'route':
                if (notification.data && notification.data.route_id) {
                    window.location.href = `/route/${notification.data.route_id}`;
                }
                break;
            case 'emergency':
                window.location.href = '/emergency-alerts';
                break;
            case 'message':
                this.closePanel();
                // Could open a messaging interface
                break;
        }
    }
    
    markAsRead(notificationId) {
        // Update UI
        const notificationEl = document.querySelector(`[data-notification-id="${notificationId}"]`);
        if (notificationEl) {
            notificationEl.classList.remove('unread');
        }
        
        // Update count
        if (this.unreadCount > 0) {
            this.unreadCount--;
            this.updateUnreadBadge();
        }
        
        // Send to server
        fetch('/api/notifications/mark-read', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
                'X-CSRF-Token': document.querySelector('[name="csrf_token"]')?.value || ''
            },
            body: JSON.stringify({ notification_id: notificationId })
        }).catch(error => console.error('Failed to mark notification as read:', error));
    }
    
    markAllAsRead() {
        // Update UI
        const unreadItems = this.notificationList.querySelectorAll('.notification-item.unread');
        unreadItems.forEach(item => item.classList.remove('unread'));
        
        // Update count
        this.unreadCount = 0;
        this.updateUnreadBadge();
        
        // Send to server
        fetch('/api/notifications/mark-all-read', {
            method: 'POST',
            headers: {
                'X-CSRF-Token': document.querySelector('[name="csrf_token"]')?.value || ''
            }
        }).catch(error => console.error('Failed to mark all notifications as read:', error));
    }
    
    updateUnreadBadge() {
        if (!this.unreadBadge) return;
        
        if (this.unreadCount > 0) {
            this.unreadBadge.style.display = 'block';
            document.getElementById('unreadCount').textContent = 
                this.unreadCount > 99 ? '99+' : this.unreadCount;
        } else {
            this.unreadBadge.style.display = 'none';
        }
    }
    
    updateUnreadCount(count) {
        this.unreadCount = count;
        this.updateUnreadBadge();
    }
    
    togglePanel() {
        if (this.notificationPanel.style.display === 'none') {
            this.openPanel();
        } else {
            this.closePanel();
        }
    }
    
    openPanel() {
        this.notificationPanel.style.display = 'block';
        // Load latest notifications if needed
        if (this.notifications.length === 0) {
            this.loadNotifications();
        }
    }
    
    closePanel() {
        this.notificationPanel.style.display = 'none';
    }
    
    async loadNotifications() {
        try {
            const response = await fetch('/api/notifications/recent');
            if (response.ok) {
                const data = await response.json();
                
                // Clear current list
                this.notificationList.innerHTML = '';
                this.notifications = [];
                
                if (data.notifications && data.notifications.length > 0) {
                    data.notifications.forEach(notification => {
                        this.notifications.push(notification);
                        this.addNotificationToList(notification);
                    });
                } else {
                    this.showEmptyState();
                }
                
                // Update unread count
                if (data.unread_count !== undefined) {
                    this.updateUnreadCount(data.unread_count);
                }
            }
        } catch (error) {
            console.error('Failed to load notifications:', error);
            this.showEmptyState();
        }
    }
    
    showEmptyState() {
        this.notificationList.innerHTML = `
            <div class="notification-empty">
                <i class="bi bi-bell-slash"></i>
                <p>No notifications</p>
            </div>
        `;
    }
    
    // Public API
    send(type, data) {
        if (this.socket && this.socket.readyState === WebSocket.OPEN) {
            this.socket.send(JSON.stringify({ type, data }));
        }
    }
    
    disconnect() {
        if (this.socket) {
            this.socket.close();
            this.socket = null;
        }
    }
}

// Initialize notification system when DOM is ready
let notificationSystem;

document.addEventListener('DOMContentLoaded', () => {
    // Only initialize for authenticated users
    if (document.querySelector('.navbar-nav') || document.querySelector('.navbar')) {
        notificationSystem = new RealtimeNotifications();
        
        // Make it globally accessible
        window.notificationSystem = notificationSystem;
    }
});

// Export for use in other modules
if (typeof module !== 'undefined' && module.exports) {
    module.exports = RealtimeNotifications;
}