# Static site (web/) served by nginx. Multi-arch base (nginx:alpine supports
# both amd64 and arm64), so this builds for whatever --platform you target.
FROM nginx:alpine

COPY nginx.conf /etc/nginx/nginx.conf
COPY web/ /usr/share/nginx/html/

EXPOSE 5294
