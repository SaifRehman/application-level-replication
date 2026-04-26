FROM nginxinc/nginx-unprivileged:1.27-alpine

USER 0
RUN rm /etc/nginx/conf.d/default.conf

COPY nginx.conf /etc/nginx/templates/nginx.conf.template
COPY index.html style.css app.js config.js /usr/share/nginx/html/

RUN chown -R 101:0 /usr/share/nginx/html /etc/nginx/templates && \
    chmod -R g=u /usr/share/nginx/html /etc/nginx/templates

USER 101

ENV BACKEND_HOST=identity-backend \
    BACKEND_PORT=8080 \
    NGINX_ENVSUBST_OUTPUT_DIR=/etc/nginx \
    NGINX_ENVSUBST_FILTER="^(BACKEND_HOST|BACKEND_PORT)$"

EXPOSE 8080
