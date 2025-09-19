# Somnia Stream Frontend

A professional web-based dashboard for monitoring real-time blockchain data from the Somnia network.

## 🚀 Quick Start

1. **Start the Somnia Stream backend** (from the parent directory):
   ```bash
   cd ..
   ./somnia-stream
   ```

2. **Start the frontend**:
   ```bash
   # Option 1: Use the demo script (recommended)
   ./demo.sh
   
   # Option 2: Manual server setup
   python3 -m http.server 3000
   ```

3. **Open your browser**:
   Navigate to `http://localhost:3000`

## 📁 Files

- `index.html` - Main HTML structure with tabbed interface
- `app.js` - JavaScript application with SSE client
- `styles.css` - Modern CSS with responsive design
- `demo.sh` - Convenience script to start HTTP server
- `README.md` - This file

## 🎯 Features

### Real-time Data Streams
- **Blocks**: Complete block information with transactions
- **Pending Transactions**: Live mempool monitoring
- **Event Logs**: Smart contract events
- **Network Statistics**: Chain stats and peer info
- **Gas Prices**: Current gas price recommendations

### User Interface
- Clean, modern design with glassmorphism effects
- Responsive layout (desktop, tablet, mobile)
- Stream-specific color coding
- Live connection status indicator
- Auto-scroll and manual controls

### Technical Features
- Server-Sent Events (SSE) for real-time updates
- Automatic reconnection handling
- Memory management (limits stored messages)
- Cross-browser compatibility
- No external dependencies

## 🔧 Customization

### Server URL
Update the backend URL in `app.js`:
```javascript
this.serverUrl = 'http://your-server:8080';
```

### Styling
Modify colors, fonts, and layout in `styles.css`:
```css
/* Main gradient */
background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);

/* Stream colors */
.block-item { border-left: 4px solid #4CAF50; }
.pending-item { border-left: 4px solid #FF9800; }
```

### Data Display
Extend the JavaScript handlers in `app.js` to:
- Add new data fields
- Implement custom formatting
- Create data visualizations
- Add filtering options

## 🌐 Browser Support

- Chrome/Edge 60+
- Firefox 55+
- Safari 12+
- Mobile browsers with SSE support

## 📱 Mobile Experience

The frontend is fully responsive and optimized for mobile devices:
- Touch-friendly controls
- Compact layout for small screens
- Swipe-friendly tab navigation
- Optimized font sizes and spacing

## 🔍 Troubleshooting

### Connection Issues
- Ensure Somnia Stream backend is running on port 8080
- Check browser console for CORS errors
- Verify network connectivity

### Performance
- The frontend automatically limits stored messages to prevent memory issues
- Use the "Clear" button to reset data if needed
- Disable auto-scroll for better performance with high-frequency updates

### Development
- Use browser developer tools to inspect SSE connections
- Check the Network tab for failed requests
- Monitor console logs for JavaScript errors

## 🎨 Screenshots

The frontend provides a professional interface with:
- Header with logo and connection status
- Navigation tabs for different streams
- Statistics cards showing key metrics
- Scrollable feed of real-time data
- Footer with connection info and message count

## 🚀 Production Deployment

For production use:

1. **Serve with a proper web server** (nginx, Apache, etc.)
2. **Configure HTTPS** for secure connections
3. **Update CORS settings** in the backend for your domain
4. **Implement caching** for static assets
5. **Add monitoring** for frontend errors

### Nginx Example
```nginx
server {
    listen 80;
    server_name your-domain.com;
    
    location / {
        root /path/to/frontend;
        index index.html;
        try_files $uri $uri/ /index.html;
    }
    
    # Proxy API requests to backend
    location /sse/ {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_cache_bypass $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Upgrade $http_upgrade;
    }
}
```

## 📄 License

This frontend is part of the Somnia Stream project and follows the same MIT license.
