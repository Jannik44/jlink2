FROM scratch
# copy views and assets
COPY ./jlink2 /app/
COPY ./views /app/views
COPY ./assets /app/assets
WORKDIR /app
CMD [ "./jlink2" ]
# how to build go application
# CGO_ENABLED=0 go build .
# sudo docker build -t jlink2 .
# how to run the container
# sudo docker run -d -p80:3000 -v /path/to/your/data/jlink2:/app/data --name jlink2 jannik44/jlink2
# another example
# docker run -d --network mainnet --name jlink2-test -v /root/dockerdata/jlink2:/app/data jlink2
