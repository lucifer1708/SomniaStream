class SomniaStreamClient {
    constructor() {
        this.serverUrl = 'http://localhost:8080';
        this.eventSources = new Map();
        this.currentStream = 'blocks';
        this.messageCount = 0;
        this.isConnected = false;
        this.maxMessages = 100; // Limit messages to prevent memory issues
        
        this.initializeEventListeners();
        this.updateServerUrl();
    }

    initializeEventListeners() {
        // Tab switching
        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.addEventListener('click', (e) => {
                this.switchTab(e.target.dataset.stream);
            });
        });

        // Control buttons
        document.getElementById('connect-btn').addEventListener('click', () => {
            this.connect();
        });

        document.getElementById('disconnect-btn').addEventListener('click', () => {
            this.disconnect();
        });

        document.getElementById('clear-btn').addEventListener('click', () => {
            this.clearData();
        });

        // Auto-scroll checkbox
        document.getElementById('auto-scroll').addEventListener('change', (e) => {
            this.autoScroll = e.target.checked;
        });

        this.autoScroll = true;
    }

    updateServerUrl() {
        document.getElementById('server-url').textContent = this.serverUrl;
    }

    switchTab(stream) {
        // Update active tab
        document.querySelectorAll('.tab-btn').forEach(btn => {
            btn.classList.remove('active');
        });
        document.querySelector(`[data-stream="${stream}"]`).classList.add('active');

        // Update active content
        document.querySelectorAll('.tab-content').forEach(content => {
            content.classList.remove('active');
        });
        document.getElementById(`${stream}-content`).classList.add('active');

        this.currentStream = stream;

        // If connected, switch to new stream
        if (this.isConnected) {
            this.disconnect();
            setTimeout(() => this.connect(), 100);
        }
    }

    connect() {
        if (this.isConnected) return;

        this.updateConnectionStatus('connecting', 'Connecting...');
        
        const streamUrl = `${this.serverUrl}/sse/${this.currentStream}`;
        console.log(`Connecting to: ${streamUrl}`);

        const eventSource = new EventSource(streamUrl);
        
        eventSource.onopen = () => {
            console.log(`Connected to ${this.currentStream} stream`);
            this.isConnected = true;
            this.updateConnectionStatus('connected', 'Connected');
            this.updateControls();
        };

        eventSource.onmessage = (event) => {
            this.handleMessage(event.data);
        };

        eventSource.onerror = (error) => {
            console.error('EventSource error:', error);
            this.updateConnectionStatus('error', 'Connection Error');
            this.isConnected = false;
            this.updateControls();
        };

        this.eventSources.set(this.currentStream, eventSource);
    }

    disconnect() {
        this.eventSources.forEach((eventSource, stream) => {
            eventSource.close();
        });
        this.eventSources.clear();
        this.isConnected = false;
        this.updateConnectionStatus('disconnected', 'Disconnected');
        this.updateControls();
    }

    updateConnectionStatus(status, text) {
        const statusElement = document.getElementById('connection-status');
        statusElement.className = `status ${status}`;
        statusElement.textContent = text;
    }

    updateControls() {
        document.getElementById('connect-btn').disabled = this.isConnected;
        document.getElementById('disconnect-btn').disabled = !this.isConnected;
    }

    handleMessage(data) {
        try {
            const parsedData = JSON.parse(data);
            this.messageCount++;
            document.getElementById('message-count').textContent = this.messageCount;

            switch (this.currentStream) {
                case 'blocks':
                    this.handleBlockData(parsedData);
                    break;
                case 'pending':
                    this.handlePendingData(parsedData);
                    break;
                case 'logs':
                    this.handleLogsData(parsedData);
                    break;
                case 'network':
                    this.handleNetworkData(parsedData);
                    break;
                case 'gasPrice':
                    this.handleGasPriceData(parsedData);
                    break;
            }
        } catch (error) {
            console.error('Error parsing message:', error);
        }
    }

    handleBlockData(data) {
        // Update stats
        document.getElementById('latest-block-number').textContent = 
            this.formatNumber(data.number);
        document.getElementById('block-time').textContent = 
            new Date(data.timestamp * 1000).toLocaleTimeString();
        document.getElementById('gas-used').textContent = 
            this.formatNumber(data.gasUsed);
        document.getElementById('tx-count').textContent = data.txCount || 0;

        // Add to feed
        const feed = document.getElementById('blocks-feed');
        const blockElement = this.createBlockElement(data);
        feed.insertBefore(blockElement, feed.firstChild);
        this.limitFeedSize(feed);
        this.scrollToTop(feed);
    }

    handlePendingData(data) {
        // Update stats
        document.getElementById('pending-count').textContent = data.count || 0;
        
        if (data.transactions && data.transactions.length > 0) {
            const avgGasPrice = data.transactions.reduce((sum, tx) => {
                return sum + parseInt(tx.gasPrice || '0', 16);
            }, 0) / data.transactions.length;
            
            document.getElementById('avg-gas-price').textContent = 
                this.formatGwei(avgGasPrice);
        }

        // Add to feed
        const feed = document.getElementById('pending-feed');
        const pendingElement = this.createPendingElement(data);
        feed.insertBefore(pendingElement, feed.firstChild);
        this.limitFeedSize(feed);
        this.scrollToTop(feed);
    }

    handleLogsData(data) {
        // Update stats
        document.getElementById('logs-count').textContent = data.count || 0;

        // Add to feed
        const feed = document.getElementById('logs-feed');
        const logsElement = this.createLogsElement(data);
        feed.insertBefore(logsElement, feed.firstChild);
        this.limitFeedSize(feed);
        this.scrollToTop(feed);
    }

    handleNetworkData(data) {
        // Update stats
        document.getElementById('chain-id').textContent = 
            data.chainId ? parseInt(data.chainId, 16) : '-';
        document.getElementById('peer-count').textContent = 
            data.peerCount ? parseInt(data.peerCount, 16) : '-';
        document.getElementById('sync-status').textContent = 
            data.syncing === false ? 'Synced' : 'Syncing';

        // Add to feed
        const feed = document.getElementById('network-feed');
        const networkElement = this.createNetworkElement(data);
        feed.insertBefore(networkElement, feed.firstChild);
        this.limitFeedSize(feed);
        this.scrollToTop(feed);
    }

    handleGasPriceData(data) {
        // Update stats
        document.getElementById('current-gas-price').textContent = 
            this.formatNumber(data.gasPrice);
        document.getElementById('gas-price-gwei').textContent = 
            `${data.gwei?.toFixed(2) || '0'} Gwei`;

        // Add to feed
        const feed = document.getElementById('gasPrice-feed');
        const gasPriceElement = this.createGasPriceElement(data);
        feed.insertBefore(gasPriceElement, feed.firstChild);
        this.limitFeedSize(feed);
        this.scrollToTop(feed);
    }

    createBlockElement(data) {
        const div = document.createElement('div');
        div.className = 'feed-item block-item';
        div.innerHTML = `
            <div class="feed-header">
                <span class="feed-title">Block #${this.formatNumber(data.number)}</span>
                <span class="feed-time">${new Date(data.timestamp * 1000).toLocaleString()}</span>
            </div>
            <div class="feed-content">
                <div class="feed-row">
                    <span class="label">Hash:</span>
                    <span class="value hash">${data.hash}</span>
                </div>
                <div class="feed-row">
                    <span class="label">Transactions:</span>
                    <span class="value">${data.txCount || 0}</span>
                </div>
                <div class="feed-row">
                    <span class="label">Gas Used:</span>
                    <span class="value">${this.formatNumber(data.gasUsed)} / ${this.formatNumber(data.gasLimit)}</span>
                </div>
                <div class="feed-row">
                    <span class="label">Size:</span>
                    <span class="value">${this.formatBytes(data.size)}</span>
                </div>
            </div>
        `;
        return div;
    }

    createPendingElement(data) {
        const div = document.createElement('div');
        div.className = 'feed-item pending-item';
        div.innerHTML = `
            <div class="feed-header">
                <span class="feed-title">Pending Transactions</span>
                <span class="feed-time">${new Date(data.timestamp * 1000).toLocaleString()}</span>
            </div>
            <div class="feed-content">
                <div class="feed-row">
                    <span class="label">Count:</span>
                    <span class="value">${data.count}</span>
                </div>
                ${data.transactions ? data.transactions.slice(0, 3).map(tx => `
                    <div class="tx-item">
                        <div class="feed-row">
                            <span class="label">Hash:</span>
                            <span class="value hash">${tx.hash}</span>
                        </div>
                        <div class="feed-row">
                            <span class="label">Value:</span>
                            <span class="value">${this.formatEther(tx.value)} ETH</span>
                        </div>
                    </div>
                `).join('') : ''}
                ${data.count > 3 ? `<div class="more-indicator">... and ${data.count - 3} more</div>` : ''}
            </div>
        `;
        return div;
    }

    createLogsElement(data) {
        const div = document.createElement('div');
        div.className = 'feed-item logs-item';
        div.innerHTML = `
            <div class="feed-header">
                <span class="feed-title">Event Logs</span>
                <span class="feed-time">${new Date(data.timestamp * 1000).toLocaleString()}</span>
            </div>
            <div class="feed-content">
                <div class="feed-row">
                    <span class="label">Count:</span>
                    <span class="value">${data.count}</span>
                </div>
                <div class="feed-row">
                    <span class="label">Block Range:</span>
                    <span class="value">${data.fromBlock} - ${data.toBlock}</span>
                </div>
            </div>
        `;
        return div;
    }

    createNetworkElement(data) {
        const div = document.createElement('div');
        div.className = 'feed-item network-item';
        div.innerHTML = `
            <div class="feed-header">
                <span class="feed-title">Network Status</span>
                <span class="feed-time">${new Date(data.timestamp * 1000).toLocaleString()}</span>
            </div>
            <div class="feed-content">
                <div class="feed-row">
                    <span class="label">Block Number:</span>
                    <span class="value">${data.blockNumber ? parseInt(data.blockNumber, 16) : '-'}</span>
                </div>
                <div class="feed-row">
                    <span class="label">Gas Price:</span>
                    <span class="value">${this.formatGwei(parseInt(data.gasPrice || '0', 16))}</span>
                </div>
                <div class="feed-row">
                    <span class="label">Peers:</span>
                    <span class="value">${data.peerCount ? parseInt(data.peerCount, 16) : '-'}</span>
                </div>
            </div>
        `;
        return div;
    }

    createGasPriceElement(data) {
        const div = document.createElement('div');
        div.className = 'feed-item gas-price-item';
        div.innerHTML = `
            <div class="feed-header">
                <span class="feed-title">Gas Price Update</span>
                <span class="feed-time">${new Date(data.timestamp * 1000).toLocaleString()}</span>
            </div>
            <div class="feed-content">
                <div class="feed-row">
                    <span class="label">Wei:</span>
                    <span class="value">${this.formatNumber(data.gasPrice)}</span>
                </div>
                <div class="feed-row">
                    <span class="label">Gwei:</span>
                    <span class="value">${data.gwei?.toFixed(4) || '0'}</span>
                </div>
            </div>
        `;
        return div;
    }

    limitFeedSize(feed) {
        while (feed.children.length > this.maxMessages) {
            feed.removeChild(feed.lastChild);
        }
    }

    scrollToTop(feed) {
        if (this.autoScroll) {
            feed.scrollTop = 0;
        }
    }

    clearData() {
        document.querySelectorAll('.data-feed').forEach(feed => {
            feed.innerHTML = '';
        });
        this.messageCount = 0;
        document.getElementById('message-count').textContent = '0';
    }

    // Utility functions
    formatNumber(num) {
        if (typeof num === 'string') {
            if (num.startsWith('0x')) {
                num = parseInt(num, 16);
            } else {
                num = parseInt(num);
            }
        }
        return num?.toLocaleString() || '0';
    }

    formatBytes(bytes) {
        if (!bytes) return '0 B';
        const k = 1024;
        const sizes = ['B', 'KB', 'MB', 'GB'];
        const i = Math.floor(Math.log(bytes) / Math.log(k));
        return `${parseFloat((bytes / Math.pow(k, i)).toFixed(2))} ${sizes[i]}`;
    }

    formatEther(wei) {
        if (!wei) return '0';
        if (typeof wei === 'string' && wei.startsWith('0x')) {
            wei = parseInt(wei, 16);
        }
        return (wei / 1e18).toFixed(6);
    }

    formatGwei(wei) {
        if (!wei) return '0 Gwei';
        return `${(wei / 1e9).toFixed(2)} Gwei`;
    }
}

// Initialize the application when DOM is loaded
document.addEventListener('DOMContentLoaded', () => {
    window.somniaClient = new SomniaStreamClient();
    console.log('Somnia Stream Client initialized');
});
