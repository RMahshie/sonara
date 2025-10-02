FROM node:18-alpine AS builder
WORKDIR /app

# Copy package files
COPY package*.json ./
RUN npm ci

# Copy web source
COPY . ./

ARG VITE_API_URL
ENV VITE_API_URL=$VITE_API_URL
RUN npm run build

# Serve with nginx
FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/nginx.conf

EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]