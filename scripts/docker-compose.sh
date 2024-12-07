#!/bin/bash

case "$1" in
  "up")
    docker-compose up -d
    ;;
  "down")
    docker-compose down
    ;;
  "build")
    docker-compose build
    ;;
  "logs")
    docker-compose logs -f
    ;;
  "restart")
    docker-compose restart
    ;;
  *)
    echo "Usage: $0 {up|down|build|logs|restart}"
    exit 1
    ;;
esac
