## How to start

```bash
# Move to docker directory
$ cd docker

# Start redis and rest of necessary backend services
$ docker-compose up -d

# Move to base directory
$ cd ..

# Start the backend server
$ go run /cmd/main.go
```
