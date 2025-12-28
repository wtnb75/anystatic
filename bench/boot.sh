#! /bin/sh
# traefik+plugin: http://bench:8001/
# nginx: http://bench:8002/
# standalone: http://bench:8003/

nginx -c /etc/nginx/nginx.conf
traefik --log.level=DEBUG --api.insecure=true &
/anystatic -dir=/website -listen=:8003 &
wait
