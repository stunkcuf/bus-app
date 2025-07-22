// Fleet Management Charts and Analytics

class FleetCharts {
    constructor() {
        this.charts = {};
        this.initChartDefaults();
    }

    initChartDefaults() {
        // Set default chart options
        Chart.defaults.color = '#fff';
        Chart.defaults.borderColor = 'rgba(255, 255, 255, 0.1)';
        Chart.defaults.font.family = '-apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif';
    }

    // Fleet Status Donut Chart
    createFleetStatusChart(elementId, data) {
        const ctx = document.getElementById(elementId).getContext('2d');
        
        this.charts.fleetStatus = new Chart(ctx, {
            type: 'doughnut',
            data: {
                labels: ['Active', 'Maintenance', 'Out of Service'],
                datasets: [{
                    data: [data.active || 0, data.maintenance || 0, data.outOfService || 0],
                    backgroundColor: [
                        'rgba(67, 233, 123, 0.8)',
                        'rgba(250, 112, 154, 0.8)',
                        'rgba(245, 87, 108, 0.8)'
                    ],
                    borderColor: [
                        'rgba(67, 233, 123, 1)',
                        'rgba(250, 112, 154, 1)',
                        'rgba(245, 87, 108, 1)'
                    ],
                    borderWidth: 2
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                plugins: {
                    legend: {
                        position: 'bottom',
                        labels: {
                            padding: 20,
                            font: { size: 14 }
                        }
                    },
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                const label = context.label || '';
                                const value = context.parsed || 0;
                                const total = context.dataset.data.reduce((a, b) => a + b, 0);
                                const percentage = ((value / total) * 100).toFixed(1);
                                return `${label}: ${value} (${percentage}%)`;
                            }
                        }
                    }
                },
                animation: {
                    animateRotate: true,
                    animateScale: true
                }
            }
        });
    }

    // Route Efficiency Line Chart
    createRouteEfficiencyChart(elementId, data) {
        const ctx = document.getElementById(elementId).getContext('2d');
        
        this.charts.routeEfficiency = new Chart(ctx, {
            type: 'line',
            data: {
                labels: data.labels || ['Mon', 'Tue', 'Wed', 'Thu', 'Fri'],
                datasets: [{
                    label: 'On-Time Performance',
                    data: data.onTimePerformance || [95, 92, 98, 94, 96],
                    borderColor: 'rgba(79, 172, 254, 1)',
                    backgroundColor: 'rgba(79, 172, 254, 0.1)',
                    borderWidth: 3,
                    tension: 0.4,
                    fill: true
                }, {
                    label: 'Student Attendance',
                    data: data.attendance || [88, 90, 85, 92, 89],
                    borderColor: 'rgba(102, 126, 234, 1)',
                    backgroundColor: 'rgba(102, 126, 234, 0.1)',
                    borderWidth: 3,
                    tension: 0.4,
                    fill: true
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: true,
                        max: 100,
                        ticks: {
                            callback: function(value) {
                                return value + '%';
                            }
                        },
                        grid: {
                            color: 'rgba(255, 255, 255, 0.1)'
                        }
                    },
                    x: {
                        grid: {
                            color: 'rgba(255, 255, 255, 0.1)'
                        }
                    }
                },
                plugins: {
                    legend: {
                        position: 'top',
                        labels: {
                            padding: 20,
                            font: { size: 14 }
                        }
                    },
                    tooltip: {
                        mode: 'index',
                        intersect: false,
                        callbacks: {
                            label: function(context) {
                                return `${context.dataset.label}: ${context.parsed.y}%`;
                            }
                        }
                    }
                },
                interaction: {
                    mode: 'nearest',
                    axis: 'x',
                    intersect: false
                }
            }
        });
    }

    // Maintenance Schedule Gantt Chart
    createMaintenanceChart(elementId, data) {
        const ctx = document.getElementById(elementId).getContext('2d');
        
        // Convert maintenance data to chart format
        const chartData = {
            labels: data.vehicles || ['Bus 24', 'Bus 25', 'Bus 26', 'Vehicle 101', 'Vehicle 102'],
            datasets: [{
                label: 'Oil Change',
                data: data.oilChanges || [
                    { x: ['2024-01-15', '2024-01-16'], y: 0 },
                    { x: ['2024-01-20', '2024-01-21'], y: 1 },
                    { x: ['2024-01-25', '2024-01-26'], y: 2 },
                    { x: ['2024-02-01', '2024-02-02'], y: 3 },
                    { x: ['2024-02-05', '2024-02-06'], y: 4 }
                ],
                backgroundColor: 'rgba(102, 126, 234, 0.8)'
            }, {
                label: 'Tire Service',
                data: data.tireServices || [
                    { x: ['2024-01-18', '2024-01-19'], y: 0 },
                    { x: ['2024-01-23', '2024-01-24'], y: 1 },
                    { x: ['2024-01-28', '2024-01-29'], y: 2 },
                    { x: ['2024-02-03', '2024-02-04'], y: 3 },
                    { x: ['2024-02-08', '2024-02-09'], y: 4 }
                ],
                backgroundColor: 'rgba(67, 233, 123, 0.8)'
            }]
        };

        this.charts.maintenance = new Chart(ctx, {
            type: 'bar',
            data: chartData,
            options: {
                responsive: true,
                maintainAspectRatio: false,
                indexAxis: 'y',
                scales: {
                    x: {
                        type: 'time',
                        time: {
                            unit: 'day',
                            displayFormats: {
                                day: 'MMM D'
                            }
                        },
                        grid: {
                            color: 'rgba(255, 255, 255, 0.1)'
                        }
                    },
                    y: {
                        grid: {
                            color: 'rgba(255, 255, 255, 0.1)'
                        }
                    }
                },
                plugins: {
                    legend: {
                        position: 'top',
                        labels: {
                            padding: 20,
                            font: { size: 14 }
                        }
                    },
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                const label = context.dataset.label;
                                const vehicle = context.chart.data.labels[context.dataIndex];
                                return `${vehicle} - ${label}`;
                            }
                        }
                    }
                }
            }
        });
    }

    // Fuel Efficiency Bar Chart
    createFuelEfficiencyChart(elementId, data) {
        const ctx = document.getElementById(elementId).getContext('2d');
        
        // Create gradient
        const gradient = ctx.createLinearGradient(0, 0, 0, 400);
        gradient.addColorStop(0, 'rgba(79, 172, 254, 0.8)');
        gradient.addColorStop(1, 'rgba(0, 242, 254, 0.3)');

        this.charts.fuelEfficiency = new Chart(ctx, {
            type: 'bar',
            data: {
                labels: data.vehicles || ['Bus 24', 'Bus 25', 'Bus 26', 'Bus 27', 'Bus 28'],
                datasets: [{
                    label: 'Miles per Gallon',
                    data: data.mpg || [8.5, 7.2, 9.1, 6.8, 8.0],
                    backgroundColor: gradient,
                    borderColor: 'rgba(79, 172, 254, 1)',
                    borderWidth: 2,
                    borderRadius: 10
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    y: {
                        beginAtZero: true,
                        ticks: {
                            callback: function(value) {
                                return value + ' MPG';
                            }
                        },
                        grid: {
                            color: 'rgba(255, 255, 255, 0.1)'
                        }
                    },
                    x: {
                        grid: {
                            display: false
                        }
                    }
                },
                plugins: {
                    legend: {
                        display: false
                    },
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                return `Efficiency: ${context.parsed.y} MPG`;
                            }
                        }
                    }
                },
                animation: {
                    delay: (context) => {
                        return context.dataIndex * 100;
                    }
                }
            }
        });
    }

    // Student Ridership Heat Map
    createRidershipHeatmap(elementId, data) {
        const ctx = document.getElementById(elementId).getContext('2d');
        
        // Process data for heatmap
        const heatmapData = [];
        const times = ['6:00', '6:30', '7:00', '7:30', '8:00'];
        const days = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri'];
        
        days.forEach((day, dayIndex) => {
            times.forEach((time, timeIndex) => {
                heatmapData.push({
                    x: day,
                    y: time,
                    v: data.ridership?.[dayIndex]?.[timeIndex] || Math.floor(Math.random() * 100)
                });
            });
        });

        this.charts.ridership = new Chart(ctx, {
            type: 'bubble',
            data: {
                datasets: [{
                    label: 'Student Count',
                    data: heatmapData.map(item => ({
                        x: item.x,
                        y: item.y,
                        r: item.v / 5 // Scale bubble size
                    })),
                    backgroundColor: function(context) {
                        const value = context.raw.r * 5;
                        const alpha = value / 100;
                        return `rgba(102, 126, 234, ${alpha})`;
                    },
                    borderColor: 'rgba(102, 126, 234, 1)',
                    borderWidth: 1
                }]
            },
            options: {
                responsive: true,
                maintainAspectRatio: false,
                scales: {
                    x: {
                        type: 'category',
                        labels: days,
                        grid: {
                            color: 'rgba(255, 255, 255, 0.1)'
                        }
                    },
                    y: {
                        type: 'category',
                        labels: times,
                        grid: {
                            color: 'rgba(255, 255, 255, 0.1)'
                        }
                    }
                },
                plugins: {
                    legend: {
                        display: false
                    },
                    tooltip: {
                        callbacks: {
                            label: function(context) {
                                const students = context.raw.r * 5;
                                return `${students} students at ${context.raw.y} on ${context.raw.x}`;
                            }
                        }
                    }
                }
            }
        });
    }

    // Update chart data
    updateChart(chartName, newData) {
        if (this.charts[chartName]) {
            this.charts[chartName].data = newData;
            this.charts[chartName].update('active');
        }
    }

    // Destroy all charts
    destroyCharts() {
        Object.values(this.charts).forEach(chart => {
            if (chart) chart.destroy();
        });
        this.charts = {};
    }

    // Export chart as image
    exportChart(chartName) {
        if (this.charts[chartName]) {
            const canvas = this.charts[chartName].canvas;
            const url = canvas.toDataURL('image/png');
            
            const link = document.createElement('a');
            link.download = `${chartName}_${new Date().toISOString().split('T')[0]}.png`;
            link.href = url;
            link.click();
        }
    }
}

// Initialize charts when DOM is ready
document.addEventListener('DOMContentLoaded', function() {
    if (typeof Chart !== 'undefined') {
        window.fleetCharts = new FleetCharts();
    }
});