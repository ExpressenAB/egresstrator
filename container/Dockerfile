FROM alpine:latest
ENV CONSUL_TEMPLATE_VER 0.14.0
ENV CONSUL_TEMPLATE_ARCH linux_amd64
RUN apk update && apk add bash curl iproute2 iptables vim \
&& curl -L -sSS https://releases.hashicorp.com/consul-template/${CONSUL_TEMPLATE_VER}/consul-template_${CONSUL_TEMPLATE_VER}_${CONSUL_TEMPLATE_ARCH}.zip \
    -o /consul-template.zip \
&& unzip /consul-template.zip \
&& rm -f /consul-template.zip \
&& mv /consul-template /usr/bin/ \
&& chmod +x /usr/bin/consul-template \
&& rm -fr /var/cache/apk
ADD . /
ENTRYPOINT ["/egresstrator.sh"]
