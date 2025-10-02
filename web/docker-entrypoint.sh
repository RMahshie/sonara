#!/bin/sh
set -e

# Use PORT environment variable from Railway, default to 80 if not set
PORT=${PORT:-80}

# Replace the port placeholder in nginx config
sed -i "s/listen 80;/listen $PORT;/g" /etc/nginx/nginx.conf

# Start nginx
exec nginx -g 'daemon off;'

