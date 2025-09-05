# DIDES Application State Machine Diagram and Flow Chart

## Overview
DIDES (Distributed Instance Deployment System) is a rolling deployment orchestration system that manages the deployment of code and configuration updates across multiple instances using a state machine architecture.

## 1. Deployment State Machine (✅ IMPLEMENTED)

### Deployment States

```mermaid
stateDiagram-v2
    [*] --> Unknown
    Unknown --> Running : TriggerDeployment()
    Running --> Completed : All instances updated successfully
    Running --> Failed : Failure threshold exceeded
    Failed --> Running : TriggerRollback()
    Completed --> Running : New deployment triggered
    Failed --> [*]
    Completed --> [*]
```

### Deployment State Definitions
- **Unknown**: Initial state when deployment record is created (iota = 0)
- **Running**: Deployment is actively progressing through instances (iota = 1) 
- **Completed**: All instances successfully updated to desired state (iota = 2)
- **Failed**: Deployment failed due to threshold exceeded or errors (iota = 3)

## 2. Instance State Machine (⚠️ PARTIALLY IMPLEMENTED)

### Instance Status States (Actual Implementation)
```mermaid
stateDiagram-v2
    [*] --> UNKNOWN
    UNKNOWN --> HEALTHY : Instance reports healthy status
    UNKNOWN --> FAILED : Instance reports failed status
    HEALTHY --> FAILED : Critical failure
    FAILED --> UNKNOWN : Reset via ResetFailedInstances()
    FAILED --> HEALTHY : Instance recovers
```

**⚠️ MISSING IMPLEMENTATION**: The `DEGRADED` status is documented in the README and diagrams but **NOT IMPLEMENTED** in the code. Only three statuses exist:
- `UNKNOWN` (iota = 0) - Default when instance is registered/not used
- `HEALTHY` (iota = 1) - Instance is functioning normally
- `FAILED` (iota = 2) - Instance has failed ⚠️ **Note: No DEGRADED status in code**

### Instance Update States (✅ IMPLEMENTED)
```mermaid
stateDiagram-v2
    [*] --> NotNeeded : currentState == desiredState
    [*] --> NeedingUpdate : currentState != desiredState
    NeedingUpdate --> InProgress : UpdateDesiredState() called
    InProgress --> Completed : Instance reports currentState == desiredState && HEALTHY
    InProgress --> Failed : Instance reports FAILED status
    Completed --> NeedingUpdate : New deployment with different desiredState
    Failed --> InProgress : Reset and retry
    NotNeeded --> NeedingUpdate : New deployment changes desiredState
```

**Implementation Details**:
- `needsUpdate()`: Checks if `currentState != desiredState` (code or config version)
- `isInProgress()`: Instance has `desiredState` set but `currentState` hasn't caught up yet
- `isCompleted()`: `currentState == desiredState` AND `status == HEALTHY`
- `isFailed()`: Instance has `desiredState` set but `status == FAILED`

## 3. Rolling Deployment Flow Chart (✅ IMPLEMENTED)

### Main Deployment Flow
```mermaid
flowchart TD
    A[TriggerDeployment Request] --> B{Validate Request}
    B -->|Invalid| C[Return Error]
    B -->|Valid| D{Check if Rollout in Progress}
    D -->|Yes| E[Return ErrRolloutInProgress]
    D -->|No| F[Acquire Lock]
    F --> G[Save Deployment Record with Status=Running]
    G --> H[StartDeployment]
    
    H --> I[Calculate Total Instances Needing Update]
    I --> J{Any Instances Need Update?}
    J -->|No| K[Mark as Completed]
    J -->|Yes| L[Get First Batch of Instances]
    L --> M[Update DesiredState for Batch]
    M --> N[Update Progress: InProgressInstances++]
    N --> O[Release Lock]
    
    K --> P[End: Deployment Completed]
    O --> Q[End: Deployment Started]
```

### Progress Deployment Flow (✅ IMPLEMENTED)
```mermaid
flowchart TD
    A[ProgressDeployment Request] --> B[Acquire Lock]
    B --> C[Get Running Deployment]
    C --> D{Deployment Exists?}
    D -->|No| E[Return nil]
    D -->|Yes| F[Count Failed Instances]
    F --> G[Count Completed Instances]
    G --> H[Count InProgress Instances]
    H --> I[Update Progress Counters]
    
    I --> J{Failed >= FailureThreshold?}
    J -->|Yes| K[Mark as Failed]
    K --> L[Trigger Automatic Rollback]
    L --> M[Return with Rollback]
    
    J -->|No| N{All Instances Completed?}
    N -->|Yes| O[Mark as Completed]
    O --> P[End: Deployment Completed]
    
    N -->|No| Q{Batch Full?}
    Q -->|Yes| R[Wait - No New Instances]
    R --> S[Update Record & Release Lock]
    
    Q -->|No| T[Calculate Available Batch Slots]
    T --> U[Get Next Batch of Instances]
    U --> V[Update DesiredState for New Instances]
    V --> W[Update Progress Counters]
    W --> S
    
    S --> X[End: Progress Updated]
```

### Rollback Flow (✅ IMPLEMENTED)
```mermaid
flowchart TD
    A[TriggerRollback Request] --> B[Acquire Lock]
    B --> C{Running Deployment Exists?}
    C -->|Yes| D[Cancel Running Deployment]
    D --> E[Mark as Failed]
    E --> F[Reset Failed Instances to UNKNOWN]
    
    C -->|No| F
    F --> G[Find Previous Completed Deployment]
    G --> H{Previous Deployment Found?}
    H -->|No| I[Return ErrNoPreviousDeploymentFound]
    H -->|Yes| J[Create Rollback Request]
    J --> K[Save New Deployment Record]
    K --> L[StartDeployment with Previous Versions]
    L --> M[Release Lock]
    M --> N[End: Rollback Started]
```

## 4. Instance Update Lifecycle (✅ IMPLEMENTED)

### Instance Registration and Update Flow
```mermaid
flowchart TD
    A[Instance Starts] --> B[Register with Controller]
    B --> C[Send Heartbeat with CurrentState]
    C --> D[Controller Updates Instance Record]
    D --> E{DesiredState != CurrentState?}
    E -->|No| F[Continue Normal Operation]
    E -->|Yes| G[Instance Receives update_needed=true]
    
    G --> H[Instance Starts Update Process]
    H --> I[Update Code/Configuration]
    I --> J{Update Successful?}
    J -->|Yes| K[Report HEALTHY with new CurrentState]
    J -->|No| L[Report FAILED status]
    
    K --> M[Controller Marks as Completed]
    L --> N[Controller Marks as Failed]
    
    M --> F
    N --> O{Failures >= Threshold?}
    O -->|Yes| P[Trigger Automatic Rollback]
    O -->|No| Q[Continue Deployment]
    
    F --> C
    P --> R[Reset Failed Instances]
    Q --> C
```

**Implementation Note**: The actual heartbeat mechanism is implemented via PATCH `/inventory/instances/{instanceID}` endpoint which updates the instance's current state and status.

## 5. Concurrency and Locking (✅ IMPLEMENTED)

### Lock Management Flow
```mermaid
flowchart TD
    A[API Request] --> B[Acquire Deployment Lock]
    B --> C{Lock Acquired?}
    C -->|No| D[Return Error]
    C -->|Yes| E[Execute Deployment Operation]
    E --> F[Update Deployment State]
    F --> G[Release Lock]
    G --> H[Return Response]
    
    subgraph "Protected Operations"
        I[TriggerDeployment]
        J[ProgressDeployment]
        K[TriggerRollback]
    end
```

**Implementation**: Uses simple in-memory locking with key "deployment" to ensure only one deployment operation can run at a time.

## 6. State Transition Conditions (✅ IMPLEMENTED)

### Instance State Logic (Updated to Match Implementation)
| Current State | Desired State | Instance Status | Result State |
|---------------|---------------|-----------------|--------------|
| v1.0.0 | v1.0.0 | Any | Not Needing Update |
| v1.0.0 | v2.0.0 | UNKNOWN/HEALTHY | Needing Update |
| v1.0.0 | v2.0.0 | FAILED | Needing Update |
| v1.0.0 (desired: v2.0.0) | v2.0.0 | HEALTHY | In Progress → Completed |
| v1.0.0 (desired: v2.0.0) | v2.0.0 | FAILED | In Progress → Failed |

⚠️ **Note**: Original documentation mentioned DEGRADED status, but this is **not implemented** in the code.

### Deployment State Logic (✅ IMPLEMENTED)
| Current Status | Condition | Next Status | Action |
|----------------|-----------|-------------|---------|
| Running | All instances completed | Completed | Mark deployment successful |
| Running | Failed instances >= threshold | Failed | Trigger automatic rollback |
| Running | Batch in progress | Running | Wait for current batch |
| Running | Batch has capacity | Running | Start next batch |
| Failed | Rollback triggered | Running | Start rollback deployment |

## 7. Error Handling States (✅ IMPLEMENTED)

### Error Conditions and Recovery
```mermaid
flowchart TD
    A[Error Detected] --> B{Error Type}
    B -->|Instance Failure| C[Increment Failed Counter]
    B -->|Threshold Exceeded| D[Mark Deployment Failed]
    B -->|System Error| E[Rollback Current Transaction]
    
    C --> F{Failed Count >= Threshold?}
    F -->|Yes| D
    F -->|No| G[Continue Deployment]
    
    D --> H[Trigger Automatic Rollback]
    H --> I[Reset Failed Instances]
    I --> J[Start Rollback Deployment]
    
    E --> K[Return Error to Client]
    G --> L[Monitor Next Progress]
    J --> L
```

**Implementation Details**:
- `ErrFailureThresholdExceeded` is returned when failed instances >= threshold
- Automatic rollback is triggered in `ProgressDeployment()` when this error occurs
- `ResetFailedInstances()` sets all FAILED instances matching labels back to UNKNOWN

## 8. HTTP API Endpoints (✅ IMPLEMENTED)

### Inventory Management
- `GET /inventory/instances` - List all instances
- `POST /inventory/instances/register` - Register new instance
- `PATCH /inventory/instances/{instanceID}` - Update instance status/state

### Deployment Management  
- `POST /deploy/` - Trigger deployment
- `GET /deploy/status` - Get running deployments status
- `POST /deploy/progress` - Manually progress deployment (for testing)
- `POST /deploy/rollback` - Trigger rollback

## 9. Improvements and Recommendations

### High Priority Enhancements:
1. **Database Storage**: Replace in-memory storage with persistent database

### Medium Priority Improvements:
1. **Background Processing**: Implement actual background reconciliation instead of manual progress calls
2. **Metrics and Monitoring**: Add deployment metrics and health monitoring
3. **Configuration Validation**: Enhance validation for deployment requests and instance registration

### Low Priority Enhancements:
1. **Implement DEGRADED Status**: Add the missing status constant and update state transitions
2. **Complete CLI Tool**: Implement actual CLI commands for deployment operations
3. **Complete Simulator**: Implement actual instance simulation functionality
2. **Advanced Deployment Strategies**: Implement blue-green, canary deployments

