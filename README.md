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

## Deployment

```

```