# Docker Demo Application

## Creating a Swarm Master

```shell
# Get a token from Docker's Discovery service
$ ID=$(docker run --rm swarm create)

# Create a swarm master using Docker Machine
$ MASTER_NAME=swarm-master
$ docker-machine create \
        -d virtualbox \
        --swarm \
        --swarm-master \
        --swarm-discovery token://${ID} \
        ${MASTER_NAME}

# Save the IP of the swarm master, we'll need it
$ MASTER_IP=$(docker-machine ip ${MASTER_NAME})

# Pull the DNS server image
$ docker $(docker-machine config ${MASTER_NAME}) pull tombee/rawdns

# Set up the daemon to use DNS we're going to set up
$ DOCKER0_IP=$(docker-machine ssh ${MASTER_NAME} ifconfig | grep -A 1 'docker0' | tail -1 | cut -d ':' -f 2 | cut -d ' ' -f 1)
$ docker-machine ssh ${MASTER_NAME} "sudo sed -i '/EXTRA_ARGS/a --dns ${DOCKER0_IP}' /var/lib/boot2docker/profile"
$ docker-machine restart ${MASTER_NAME}

# Create our DNS config
$ docker-machine ssh ${MASTER_NAME} sudo tee /etc/rawdns.json <<EOF
{
    "swarm.": {
        "type": "containers",
        "socket": "tcp://${MASTER_IP}:3376",
        "swarmnode": true,
        "tlsverify": true,
        "tlscacert": "/var/lib/boot2docker/ca.pem",
        "tlscert": "/var/lib/boot2docker/server.pem",
        "tlskey": "/var/lib/boot2docker/server-key.pem"
    },
    ".": {
        "type": "forwarding",
        "nameservers": [ "8.8.8.8", "8.8.4.4" ]
    }
}
EOF

# Run the DNS server
$ docker $(docker-machine config ${MASTER_NAME}) run \
    --name dns \
    -d -it \
    -p 53:53/udp \
    -v /var/lib/boot2docker:/var/lib/boot2docker \
    -v /var/run/docker.sock:/var/run/docker.sock \
    -v /etc/rawdns.json:/etc/rawdns.json:ro \
    tombee/rawdns rawdns /etc/rawdns.json

# Restart swarm containers now DNS is running
$ docker $(docker-machine config ${MASTER_NAME}) restart swarm-agent swarm-agent-master

# Ensure you get an answer section from
$ dig @${MASTER_IP} dns.swarm
```

## Creating Swarm Agents

```shell
$ MACHINE_NAME=swarm-agent-1
$ docker-machine create \
        -d virtualbox \
        --swarm \
        --swarm-discovery token://${ID} \
        ${MACHINE_NAME}

$ docker-machine ssh ${MACHINE_NAME} "sudo sed -i '/EXTRA_ARGS/a --dns ${MASTER_IP}' /var/lib/boot2docker/profile"
$ docker-machine restart ${MACHINE_NAME}
```

## Configure your Docker environment

```shell
$ eval $(docker-machine env --swarm swarm-master)
```

## Run the app

```shell
$ docker-compose up -f docker-compose.prod.yml
```
