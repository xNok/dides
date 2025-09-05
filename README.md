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
    "batch_size": 2,
    "failure_threshold": 1,
  }
}
```

The system only accept one in flight deployment at the time.

## Deployment Process (after a trigger)

Once a deployment is trigger the coordinatoor update the desired state (`code_version`, `configuration_version`) for up to `max_inflight` instances and motior the progress of the deployment with the instances heartbeat.

Once 1 instance report `HEALTHY` and `code_version == target_code_version` then the deployment progress by updating the `target_code_version` for one of the remaining instances.

In the best case scenario all intances eventualled report `HEALTHY` and `code_version == target_code_version`. Then the eployment status is marked as completed.

Failure scenarios:
* 