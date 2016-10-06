# ImagX

## Setup

- Update Dockerfile to replace your AWS credentials
- `docker build -t imagx .`
- `docker run --publish 6060:8080 --name test --rm imagx`