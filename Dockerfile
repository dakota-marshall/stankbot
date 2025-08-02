FROM alpine:3.14
LABEL maintainer="Dakota Marshall <me@dakotamarshall.net>"
ENV TZ="America/New_York"

RUN apk add ffmpeg
RUN mkdir /app
COPY ./stankbot /app/stankbot
WORKDIR /app
RUN chmod +x stankbot

CMD ["/app/stankbot"]

