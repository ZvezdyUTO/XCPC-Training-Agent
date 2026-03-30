FROM node:22-alpine AS builder

WORKDIR /frontend

COPY frontend/package.json ./
RUN npm install

COPY frontend ./
RUN npm run build

FROM nginx:1.27-alpine

COPY frontend/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /frontend/dist /usr/share/nginx/html

EXPOSE 80
