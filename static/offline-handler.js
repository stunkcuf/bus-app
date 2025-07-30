// Offline Handler
// ==============
// Manages service worker registration and offline functionality

class OfflineHandler {
    constructor() {
        this.isOnline = navigator.onLine;
        this.db = null;
        this.syncQueue = [];
        
        this.init();
    }
    
    async init() {
        // Register service worker
        if ('serviceWorker' in navigator) {
            try {
                const registration = await navigator.serviceWorker.register('/static/service-worker.js');
                console.log('Service Worker registered:', registration);
                
                // Check for updates
                registration.addEventListener('updatefound', () => {
                    const newWorker = registration.installing;
                    newWorker.addEventListener('statechange', () => {
                        if (newWorker.state === 'installed' && navigator.serviceWorker.controller) {
                            this.showUpdateNotification();
                        }
                    });
                });
            } catch (error) {
                console.error('Service Worker registration failed:', error);
            }
        }
        
        // Initialize IndexedDB
        this.db = await this.openDatabase();
        
        // Set up event listeners
        this.setupEventListeners();
        
        // Check initial connection status
        this.updateConnectionStatus();
        
        // Sync any pending data
        if (this.isOnline) {
            this.syncPendingData();
        }
    }
    
    setupEventListeners() {
        // Online/offline events
        window.addEventListener('online', () => {
            this.isOnline = true;
            this.updateConnectionStatus();
            this.syncPendingData();
        });
        
        window.addEventListener('offline', () => {
            this.isOnline = false;
            this.updateConnectionStatus();
        });
        
        // Intercept form submissions
        document.addEventListener('submit', (e) => {
            const form = e.target;
            if (form.dataset.offlineCapable === 'true') {
                e.preventDefault();
                this.handleOfflineForm(form);
            }
        });
    }
    
    updateConnectionStatus() {
        const statusElement = document.getElementById('connection-status');
        if (!statusElement) {
            this.createStatusIndicator();
            return;
        }
        
        if (this.isOnline) {
            statusElement.className = 'connection-status online';
            statusElement.innerHTML = '<i class="bi bi-wifi"></i> Online';
        } else {
            statusElement.className = 'connection-status offline';
            statusElement.innerHTML = '<i class="bi bi-wifi-off"></i> Offline';
        }
    }
    
    createStatusIndicator() {
        const status = document.createElement('div');
        status.id = 'connection-status';
        status.className = 'connection-status';
        
        const style = document.createElement('style');
        style.textContent = `
            .connection-status {
                position: fixed;
                top: 10px;
                right: 10px;
                background: rgba(255, 255, 255, 0.1);
                backdrop-filter: blur(10px);
                padding: 8px 16px;
                border-radius: 20px;
                font-size: 14px;
                display: flex;
                align-items: center;
                gap: 8px;
                z-index: 1000;
                transition: all 0.3s ease;
            }
            
            .connection-status.online {
                background: rgba(16, 185, 129, 0.2);
                color: #10b981;
                border: 1px solid rgba(16, 185, 129, 0.5);
            }
            
            .connection-status.offline {
                background: rgba(239, 68, 68, 0.2);
                color: #ef4444;
                border: 1px solid rgba(239, 68, 68, 0.5);
            }
            
            .sync-indicator {
                position: fixed;
                bottom: 20px;
                right: 20px;
                background: rgba(102, 126, 234, 0.9);
                color: white;
                padding: 12px 20px;
                border-radius: 25px;
                display: none;
                align-items: center;
                gap: 10px;
                box-shadow: 0 4px 20px rgba(0, 0, 0, 0.3);
                z-index: 1000;
            }
            
            .sync-indicator.show {
                display: flex;
            }
            
            .sync-spinner {
                width: 16px;
                height: 16px;
                border: 2px solid rgba(255, 255, 255, 0.3);
                border-radius: 50%;
                border-top-color: white;
                animation: spin 0.8s linear infinite;
            }
            
            @keyframes spin {
                to { transform: rotate(360deg); }
            }
        `;
        
        document.head.appendChild(style);
        document.body.appendChild(status);
        
        this.updateConnectionStatus();
    }
    
    async openDatabase() {
        return new Promise((resolve, reject) => {
            const request = indexedDB.open('FleetManagementDB', 1);
            
            request.onerror = () => reject(request.error);
            request.onsuccess = () => resolve(request.result);
            
            request.onupgradeneeded = (event) => {
                const db = event.target.result;
                
                // Create object stores
                if (!db.objectStoreNames.contains('tripLogs')) {
                    const tripStore = db.createObjectStore('tripLogs', { keyPath: 'id', autoIncrement: true });
                    tripStore.createIndex('synced', 'synced', { unique: false });
                    tripStore.createIndex('timestamp', 'timestamp', { unique: false });
                }
                
                if (!db.objectStoreNames.contains('attendance')) {
                    const attendanceStore = db.createObjectStore('attendance', { keyPath: 'id', autoIncrement: true });
                    attendanceStore.createIndex('synced', 'synced', { unique: false });
                    attendanceStore.createIndex('tripId', 'tripId', { unique: false });
                }
                
                if (!db.objectStoreNames.contains('students')) {
                    db.createObjectStore('students', { keyPath: 'student_id' });
                }
                
                if (!db.objectStoreNames.contains('routes')) {
                    db.createObjectStore('routes', { keyPath: 'route_id' });
                }
                
                if (!db.objectStoreNames.contains('formData')) {
                    const formStore = db.createObjectStore('formData', { keyPath: 'id', autoIncrement: true });
                    formStore.createIndex('synced', 'synced', { unique: false });
                    formStore.createIndex('formType', 'formType', { unique: false });
                }
            };
        });
    }
    
    async handleOfflineForm(form) {
        const formData = new FormData(form);
        const data = Object.fromEntries(formData.entries());
        data.timestamp = new Date().toISOString();
        data.synced = false;
        data.formType = form.dataset.formType || 'generic';
        
        if (this.isOnline) {
            // Try to submit directly
            try {
                await this.submitForm(form.action, data);
                this.showNotification('Data saved successfully', 'success');
            } catch (error) {
                // Save for later sync
                await this.saveOfflineData('formData', data);
                this.showNotification('Saved offline. Will sync when connected.', 'info');
            }
        } else {
            // Save for later sync
            await this.saveOfflineData('formData', data);
            this.showNotification('Saved offline. Will sync when connected.', 'info');
        }
    }
    
    async saveOfflineData(storeName, data) {
        const transaction = this.db.transaction([storeName], 'readwrite');
        const store = transaction.objectStore(storeName);
        
        return new Promise((resolve, reject) => {
            const request = store.add(data);
            request.onsuccess = () => resolve(request.result);
            request.onerror = () => reject(request.error);
        });
    }
    
    async submitForm(url, data) {
        const response = await fetch(url, {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify(data)
        });
        
        if (!response.ok) {
            throw new Error('Network response was not ok');
        }
        
        return response.json();
    }
    
    async syncPendingData() {
        if (!this.isOnline || !this.db) return;
        
        this.showSyncIndicator();
        
        // Sync different types of data
        await this.syncFormData();
        await this.syncTripLogs();
        await this.syncAttendance();
        
        this.hideSyncIndicator();
        
        // Request background sync if available
        if ('sync' in self.registration) {
            try {
                await self.registration.sync.register('sync-all-data');
            } catch (error) {
                console.error('Background sync registration failed:', error);
            }
        }
    }
    
    async syncFormData() {
        const transaction = this.db.transaction(['formData'], 'readonly');
        const store = transaction.objectStore('formData');
        const index = store.index('synced');
        const pendingData = await this.getAll(index.getAll(false));
        
        for (const data of pendingData) {
            try {
                // Determine the correct endpoint based on form type
                const endpoint = this.getEndpointForFormType(data.formType);
                await this.submitForm(endpoint, data);
                
                // Mark as synced
                await this.markAsSynced('formData', data.id);
            } catch (error) {
                console.error('Failed to sync form data:', error);
            }
        }
    }
    
    async syncTripLogs() {
        const transaction = this.db.transaction(['tripLogs'], 'readonly');
        const store = transaction.objectStore('tripLogs');
        const index = store.index('synced');
        const pendingLogs = await this.getAll(index.getAll(false));
        
        for (const log of pendingLogs) {
            try {
                await this.submitForm('/api/trip-log', log);
                await this.markAsSynced('tripLogs', log.id);
            } catch (error) {
                console.error('Failed to sync trip log:', error);
            }
        }
    }
    
    async syncAttendance() {
        const transaction = this.db.transaction(['attendance'], 'readonly');
        const store = transaction.objectStore('attendance');
        const index = store.index('synced');
        const pendingAttendance = await this.getAll(index.getAll(false));
        
        for (const record of pendingAttendance) {
            try {
                await this.submitForm('/api/attendance', record);
                await this.markAsSynced('attendance', record.id);
            } catch (error) {
                console.error('Failed to sync attendance:', error);
            }
        }
    }
    
    getEndpointForFormType(formType) {
        const endpoints = {
            'trip-log': '/api/trip-log',
            'attendance': '/api/attendance',
            'student': '/api/student',
            'maintenance': '/api/maintenance',
            'generic': '/api/form-submit'
        };
        
        return endpoints[formType] || endpoints.generic;
    }
    
    async markAsSynced(storeName, id) {
        const transaction = this.db.transaction([storeName], 'readwrite');
        const store = transaction.objectStore(storeName);
        
        const request = store.get(id);
        request.onsuccess = () => {
            const data = request.result;
            data.synced = true;
            store.put(data);
        };
    }
    
    getAll(request) {
        return new Promise((resolve, reject) => {
            request.onsuccess = () => resolve(request.result);
            request.onerror = () => reject(request.error);
        });
    }
    
    showSyncIndicator() {
        let indicator = document.getElementById('sync-indicator');
        if (!indicator) {
            indicator = document.createElement('div');
            indicator.id = 'sync-indicator';
            indicator.className = 'sync-indicator';
            indicator.innerHTML = `
                <div class="sync-spinner"></div>
                <span>Syncing data...</span>
            `;
            document.body.appendChild(indicator);
        }
        indicator.classList.add('show');
    }
    
    hideSyncIndicator() {
        const indicator = document.getElementById('sync-indicator');
        if (indicator) {
            indicator.classList.remove('show');
        }
    }
    
    showNotification(message, type = 'info') {
        const notification = document.createElement('div');
        notification.className = `notification notification-${type}`;
        notification.textContent = message;
        
        const style = document.createElement('style');
        style.textContent = `
            .notification {
                position: fixed;
                top: 80px;
                right: 20px;
                background: rgba(255, 255, 255, 0.9);
                color: #333;
                padding: 16px 24px;
                border-radius: 8px;
                box-shadow: 0 4px 20px rgba(0, 0, 0, 0.2);
                transform: translateX(400px);
                transition: transform 0.3s ease;
                z-index: 1001;
            }
            
            .notification.show {
                transform: translateX(0);
            }
            
            .notification-success {
                background: rgba(16, 185, 129, 0.9);
                color: white;
            }
            
            .notification-info {
                background: rgba(102, 126, 234, 0.9);
                color: white;
            }
            
            .notification-error {
                background: rgba(239, 68, 68, 0.9);
                color: white;
            }
        `;
        
        if (!document.getElementById('notification-styles')) {
            style.id = 'notification-styles';
            document.head.appendChild(style);
        }
        
        document.body.appendChild(notification);
        
        // Show notification
        setTimeout(() => notification.classList.add('show'), 10);
        
        // Hide and remove after 3 seconds
        setTimeout(() => {
            notification.classList.remove('show');
            setTimeout(() => notification.remove(), 300);
        }, 3000);
    }
    
    showUpdateNotification() {
        const notification = document.createElement('div');
        notification.className = 'update-notification';
        notification.innerHTML = `
            <p>A new version of the app is available!</p>
            <button onclick="offlineHandler.updateApp()">Update Now</button>
            <button onclick="this.parentElement.remove()">Later</button>
        `;
        
        document.body.appendChild(notification);
    }
    
    updateApp() {
        if ('serviceWorker' in navigator && navigator.serviceWorker.controller) {
            navigator.serviceWorker.controller.postMessage({ action: 'skipWaiting' });
            window.location.reload();
        }
    }
}

// Initialize offline handler
const offlineHandler = new OfflineHandler();

// Export for use in other modules
window.offlineHandler = offlineHandler;