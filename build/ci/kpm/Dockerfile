FROM  golang:1.20.1-alpine3.17 as  builder
# builder-golang-alpine
WORKDIR /app
ENV GOPROXY https://goproxy.cn
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
&& apk add --no-cache build-base \
&& apk add --no-cache upx \
&& apk add --no-cache git \
&& mkdir /lib64 \
&& ln -s /lib/ld-linux-x86-64.so.2 /lib64/ld-linux-x86-64.so.2
# building kpm
ARG COMPRESS='false'
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -trimpath -a -ldflags "-linkmode external -w -s -extldflags  -static " -gcflags "-l=4 -m=2 "  --tags netgo -o /dist/kpm github.com/KusionStack/kpm/cmd/kpm  \
    &&  if [  ${COMPRESS} = 'true' ]; then upx -9 /dist/kpm; fi && ls /dist



FROM alpine:3.17.2
# runtime-env-alpine-git
ENV TZ="Asia/Shanghai"
WORKDIR /app
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.aliyun.com/g' /etc/apk/repositories \
&& apk add --no-cache tzdata \
&& apk add --no-cache ca-certificates \
&& apk add --no-cache git \
&& mkdir /lib64 \
&& ln -s /lib/libc.musl-x86_64.so.1 /lib64/ld-linux-x86-64.so.2 \
&& ln -sf /usr/share/zoneinfo/$TZ /etc/localtime \
&& echo $TZ > /etc/timezone \
# add kpm and run it
COPY  --from=builder  /dist/kpm /bin/kpm
CMD ["/bin/sh"]

