#!/bin/bash

# Somnia Stream Frontend Demo Script
# This script starts a simple HTTP server to serve the frontend files

echo "🚀 Starting Somnia Stream Frontend Demo"
echo "========================================"

# Check if we're in the frontend directory
if [ ! -f "index.html" ]; then
    echo "❌ Error: Please run this script from the frontend directory"
    echo "   cd frontend && ./demo.sh"
    exit 1
fi

# Default port
PORT=${1:-3000}

echo "📁 Serving files from: $(pwd)"
echo "🌐 Frontend will be available at: http://localhost:$PORT"
echo "🔌 Make sure Somnia Stream backend is running on port 8080"
echo ""

# Try different HTTP servers in order of preference
if command -v python3 &> /dev/null; then
    echo "🐍 Starting Python 3 HTTP server..."
    python3 -m http.server $PORT
elif command -v python &> /dev/null; then
    echo "🐍 Starting Python HTTP server..."
    python -m http.server $PORT
elif command -v php &> /dev/null; then
    echo "🐘 Starting PHP development server..."
    php -S localhost:$PORT
elif command -v npx &> /dev/null; then
    echo "📦 Starting Node.js serve..."
    npx serve -p $PORT
else
    echo "❌ No suitable HTTP server found!"
    echo "   Please install one of: python3, python, php, or node.js"
    echo ""
    echo "   Alternative: Open index.html directly in your browser"
    echo "   Note: Some features may not work due to CORS restrictions"
    exit 1
fi
