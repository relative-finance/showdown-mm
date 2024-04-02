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

## How to test websockets

```bash
# Install wscat
$ npm install -g wscat

# Connect to the websocket server
$ wscat -c ws://localhost:8080/ws
```

## How to connect to cs2 or dota2 queue

```bash
# For CS2
$ wscat -c ws://localhost:8080/ws/cs2queue/{steamId}

# For Dota2
$ wscat -c ws://localhost:8080/ws/d2queue/{steamId}
```
