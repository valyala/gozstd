ARG builder_image
FROM $builder_image
RUN apk --update --no-cache add ca-certificates
RUN apk --no-cache add make git

ENTRYPOINT ["/bin/sh"]