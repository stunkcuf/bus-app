// Service Worker for Offline Capabilities
// ======================================

const CACHE_NAME = 'fleet-management-v1';
const OFFLINE_URL = '/offline.html';

// Critical assets to cache for offline use
const CRITICAL_ASSETS = [
    '/',
    '/static/enhanced_ui.css',
    '/static/dark_theme_text.css',
    '/static/touch_friendly.css',
    '/static/touch_friendly.js',
    '/static/contextual_help.js',
    '/static/realtime_notifications.js',
    'https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/css/bootstrap.min.css',
    'https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.0/font/bootstrap-icons.css',
    'https://cdn.jsdelivr.net/npm/bootstrap@5.3.0/dist/js/bootstrap.bundle.min.js'
];

// Critical pages for offline access
const OFFLINE_PAGES = [
    '/driver-dashboard',
    '/trip-log',
    '/students',
    '/help-center',
    '/offline.html'
];

// Install event - cache critical assets
self.addEventListener('install', event => {
    event.waitUntil(
        caches.open(CACHE_NAME)
            .then(cache => {
                console.log('Caching critical assets');
                return cache.addAll(CRITICAL_ASSETS.concat(OFFLINE_PAGES));
            })
            .then(() => self.skipWaiting())
    );
});

// Activate event - clean up old caches
self.addEventListener('activate', event => {
    event.waitUntil(
        caches.keys()
            .then(cacheNames => {
                return Promise.all(
                    cacheNames
                        .filter(cacheName => cacheName !== CACHE_NAME)
                        .map(cacheName => caches.delete(cacheName))
                );
            })
            .then(() => self.clients.claim())
    );
});

// Fetch event - serve from cache when offline
self.addEventListener('fetch', event => {
    // Skip non-GET requests
    if (event.request.method !== 'GET') {
        return;
    }

    // Handle navigation requests
    if (event.request.mode === 'navigate') {
        event.respondWith(
            fetch(event.request)
                .catch(() => {
                    return caches.match(event.request)
                        .then(cachedResponse => {
                            if (cachedResponse) {
                                return cachedResponse;
                            }
                            // Return offline page for navigation requests
                            return caches.match(OFFLINE_URL);
                        });
                })
        );
        return;
    }

    // Handle other requests with network-first strategy
    event.respondWith(
        fetch(event.request)
            .then(response => {
                // Don't cache non-successful responses
                if (!response || response.status !== 200 || response.type !== 'basic') {
                    return response;
                }

                // Clone the response
                const responseToCache = response.clone();

                caches.open(CACHE_NAME)
                    .then(cache => {
                        cache.put(event.request, responseToCache);
                    });

                return response;
            })
            .catch(() => {
                // Try to return cached version
                return caches.match(event.request);
            })
    );
});

// Background sync for offline data submission
self.addEventListener('sync', event => {
    if (event.tag === 'sync-trip-logs') {
        event.waitUntil(syncTripLogs());
    } else if (event.tag === 'sync-attendance') {
        event.waitUntil(syncAttendance());
    }
});

// Sync functions
async function syncTripLogs() {
    const db = await openIndexedDB();
    const pendingLogs = await getPendingTripLogs(db);
    
    for (const log of pendingLogs) {
        try {
            const response = await fetch('/api/trip-log', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(log)
            });
            
            if (response.ok) {
                await markLogAsSynced(db, log.id);
            }
        } catch (error) {
            console.error('Failed to sync trip log:', error);
        }
    }
}

async function syncAttendance() {
    const db = await openIndexedDB();
    const pendingAttendance = await getPendingAttendance(db);
    
    for (const record of pendingAttendance) {
        try {
            const response = await fetch('/api/attendance', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(record)
            });
            
            if (response.ok) {
                await markAttendanceAsSynced(db, record.id);
            }
        } catch (error) {
            console.error('Failed to sync attendance:', error);
        }
    }
}

// IndexedDB functions
function openIndexedDB() {
    return new Promise((resolve, reject) => {
        const request = indexedDB.open('FleetManagementDB', 1);
        
        request.onerror = () => reject(request.error);
        request.onsuccess = () => resolve(request.result);
        
        request.onupgradeneeded = event => {
            const db = event.target.result;
            
            // Create object stores
            if (!db.objectStoreNames.contains('tripLogs')) {
                const tripStore = db.createObjectStore('tripLogs', { keyPath: 'id', autoIncrement: true });
                tripStore.createIndex('synced', 'synced', { unique: false });
            }
            
            if (!db.objectStoreNames.contains('attendance')) {
                const attendanceStore = db.createObjectStore('attendance', { keyPath: 'id', autoIncrement: true });
                attendanceStore.createIndex('synced', 'synced', { unique: false });
            }
            
            if (!db.objectStoreNames.contains('students')) {
                db.createObjectStore('students', { keyPath: 'student_id' });
            }
            
            if (!db.objectStoreNames.contains('routes')) {
                db.createObjectStore('routes', { keyPath: 'route_id' });
            }
        };
    });
}

function getPendingTripLogs(db) {
    return new Promise((resolve, reject) => {
        const transaction = db.transaction(['tripLogs'], 'readonly');
        const store = transaction.objectStore('tripLogs');
        const index = store.index('synced');
        const request = index.getAll(false);
        
        request.onsuccess = () => resolve(request.result);
        request.onerror = () => reject(request.error);
    });
}

function getPendingAttendance(db) {
    return new Promise((resolve, reject) => {
        const transaction = db.transaction(['attendance'], 'readonly');
        const store = transaction.objectStore('attendance');
        const index = store.index('synced');
        const request = index.getAll(false);
        
        request.onsuccess = () => resolve(request.result);
        request.onerror = () => reject(request.error);
    });
}

function markLogAsSynced(db, id) {
    return new Promise((resolve, reject) => {
        const transaction = db.transaction(['tripLogs'], 'readwrite');
        const store = transaction.objectStore('tripLogs');
        const request = store.get(id);
        
        request.onsuccess = () => {
            const log = request.result;
            log.synced = true;
            const updateRequest = store.put(log);
            updateRequest.onsuccess = () => resolve();
            updateRequest.onerror = () => reject(updateRequest.error);
        };
        
        request.onerror = () => reject(request.error);
    });
}

function markAttendanceAsSynced(db, id) {
    return new Promise((resolve, reject) => {
        const transaction = db.transaction(['attendance'], 'readwrite');
        const store = transaction.objectStore('attendance');
        const request = store.get(id);
        
        request.onsuccess = () => {
            const record = request.result;
            record.synced = true;
            const updateRequest = store.put(record);
            updateRequest.onsuccess = () => resolve();
            updateRequest.onerror = () => reject(updateRequest.error);
        };
        
        request.onerror = () => reject(request.error);
    });
}

// Push notifications
self.addEventListener('push', event => {
    const options = {
        body: event.data ? event.data.text() : 'New notification',
        icon: '/static/images/icon-192.png',
        badge: '/static/images/badge-72.png',
        vibrate: [200, 100, 200],
        data: {
            dateOfArrival: Date.now(),
            primaryKey: 1
        }
    };
    
    event.waitUntil(
        self.registration.showNotification('Fleet Management System', options)
    );
});

// Notification click handler
self.addEventListener('notificationclick', event => {
    event.notification.close();
    
    event.waitUntil(
        clients.openWindow('/')
    );
});

// Message handler for communication with main app
self.addEventListener('message', event => {
    if (event.data.action === 'skipWaiting') {
        self.skipWaiting();
    }
});