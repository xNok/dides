# dides

distributed deployment system


## Register A New instance

We assume that the instance are managed by an agent and can self register to the coordinatoor provided an identity or a token.

```
POST /inventory/register

{
  "instance": {
    "ip": "192.168.1.100",
    "name": "web-server-01",
    "labels": {
      "environment": "production",
      "role": "web"
    }
  },
  "token": "your-registration-token"
}
```

## Instance heartbeat

We assume we recieve regular heartbit from the instances with an update with `code_version`, `configuration_version` and `status`

```
PATCH /inventory/instances/{instanceName}
{
  "code_version": "1.0.0",
  "configuration_version": "1.0.1"
  "status": 1
}
```

We use status code to represent the satus of an instance

```
UNKNOWN  => 0
HEALTHY  => 1
DEGRADED => 2
FAILED   => 3
```


## Deployment Trigger

You can deploy a new version of to a set of instance using the deploy endpoint, the body tells the coordinator the desired state, the target instances and configuration about deployment process itself.

```
POST /deploy
{
  "code_version": "1.0.0",
  "configuration_version": "1.0.1"
  "lables": {
    "env": "production"
  },
  "condiguration": {
    "max_inflight": 2
  }
}
```