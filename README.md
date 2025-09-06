# DIDES

A distributed deployment system that safely rolls out updates across many service instances.

![](./dides.png)

## How to Run the System

```bash
go run ./cmd/controller/main.go
```

Note: Most of the end-to-end testing logic is in [main_test.go](./cmd/main_test.go) until a CLI is provided for testing.

```bash
go test ./cmd/controller/... -v
```

### Inventory Management
- `GET /inventory/instances` - List all instances
- `POST /inventory/instances/register` - Register new instance
- `PATCH /inventory/instances/{instanceID}` - Update instance status/state

### Deployment Management  
- `POST /deploy` - Trigger deployment
- `GET /deploy/status` - Get running deployments status
- `POST /deploy/progress` - Manually progress deployment
- `POST /deploy/rollback` - Manually trigger rollback

### Assumptions Made, Design Decisions, Notes, and Thoughts

* Decouple inventory and instance updates from deployment management
  * Applying updates can be slow or unreliable, so it is better to get the big picture each time we decide to progress the deployment
  * Instance updates can have high throughput while deployments need to progress at a steady pace
* Use `POST /deploy/progress` to make the reconciliation loop progress and have deterministic tests
  * The reconciliation loop would require a scheduling library to run periodically (for instance, every 10s)
* Introduce the concept of `DeploymentStrategy` to support different deployment strategies (canary, percentage rollout, etc.)
  * The `DeploymentTrigger` delegates the `desired_state` updates to the `DeploymentStrategy`
* Use semantic versioning instead of `SHA1 hashes` as it makes tests more readable while not changing the logic (i.e., `SHA1 hashes` can still be used)
* Added labels filtering on resource to only target a subset of instances to be updated.

## Implemented vs. Skipped

This application provides:
1. **State transitions** for both deployments and instances ✅
2. **Concurrency control** through locking mechanisms ✅  
3. **Rollback capabilities** for failed deployments ✅
4. **Progress tracking** through deployment progress counters ✅
5. **Batch processing** to control number of in-flight deployments ✅

The application is missing:
1. **Database Storage**: Replace in-memory storage with persistent database
2. **Background Processing**: Implement actual background reconciliation instead of manual progress calls
3. **Configuration Validation**: Enhance validation for deployment requests and instance registration
4. **Metrics and Monitoring**: Add deployment metrics and health monitoring
5. **Implement DEGRADED Status**: Add the missing status constant and update state transitions


## More Detailed Workflow

### Register a New Instance

We assume that the instances are managed by an agent and can self-register to the coordinator provided an identity or a token.

```
POST /inventory/instances/register

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

## Instance Heartbeat

We assume we receive regular heartbeats from the instances with updates containing `code_version`, `configuration_version`, and `status`

```
PATCH /inventory/instances/{instanceName}
{
  "code_version": "1.0.0",
  "configuration_version": "1.0.1",
  "status": 1
}
```

We use status codes to represent the status of an instance

```
UNKNOWN  => 0
HEALTHY  => 1
FAILED   => 2
```


## Deployment Trigger

You can deploy a new version to a set of instances using the deploy endpoint. The body tells the coordinator the desired state, the target instances, and configuration about the deployment process itself.

```
POST /deploy
{
  "code_version": "1.0.0",
  "configuration_version": "1.0.1",
  "labels": {
    "env": "production"
  },
  "configuration": {
    "batch_size": 2,
    "failure_threshold": 1
  }
}
```

The system only accepts one in-flight deployment at a time.

## Deployment Progress (After a Trigger)

Once a deployment is triggered, the coordinator updates the desired state (`code_version`, `configuration_version`) for up to `batch_size` instances and monitors the progress of the deployment through the instances' heartbeats.

Once instances report `HEALTHY` and `code_version == target_code_version`, the deployment progresses by updating the `target_code_version` for one of the remaining instances. The update process can be automated using a reconciliation interval. However, to provide a simple way to test the implementation, we can use the following endpoint.

```
POST /deploy/progress
{}
```

In the best-case scenario, all instances eventually report `HEALTHY` and `current_state == desired_state`. Then the deployment status is marked as completed.


## Which parts were LLM-written vs handcrafted

These packages are LLM generated to make tesing easier 
* [internal/infra/in-memory/](./internal/infra/in-memory/) 
* [pkg/simulator](./pkg/simulator/) 

Most of the documentation in the [docs](./docs/) folder is LLM generated including the diagrams.
