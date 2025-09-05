# `SHA1 hashes`

```mermaid
`SHA1 hashes`
    title DIDES Application - Core Struct Relationships
    note "Inventory, Deployment, and Strategy"

    %% Inventory Models
    class Instance {
        +string IP
        +string Name
        +map~string,string~ Labels
        +Status Status
        +State CurrentState
        +State DesiredState
    }
    
    class State {
        +string CodeVersion
        +string ConfigurationVersion
    }
    
    class InventoryStore {
        <<interface>>
        +GetByLabels(labels) []*Instance
        +CountByLabels(labels) int
        +GetNeedingUpdate(labels, desiredState) []*Instance
        +CountInProgress(labels, desiredState) int
        +CountCompleted(labels, desiredState) int
        +CountFailed(labels, desiredState) int
        +Update(key, patch) *Instance
    }
    
    class StateService {
        -InventoryStore store
        +UpdateDesiredState(instanceKey, state)
        +GetInstancesByLabels(labels)
        +CountByLabels(labels)
        +GetNeedingUpdate(labels, desiredState)
    }

    %% Deployment Models
    class DeploymentRequest {
        +string CodeVersion
        +string ConfigurationVersion
        +map~string,string~ Labels
        +Configuration Configuration
    }
    
    class Configuration {
        +int BatchSize
        +int FailureThreshold
    }
    
    class DeploymentRecord {
        +string ID
        +DeploymentRequest Request
        +DeploymentStatus Status
        +DeploymentProgress Progress
    }
    
    class DeploymentProgress {
        +int TotalMatchingInstances
        +int InProgressInstances
        +int CompletedInstances
        +int FailedInstances
    }
    
    class DeploymentStore {
        <<interface>>
        +Save(record) error
        +Update(record) error
        +GetByStatus(status) []*DeploymentRecord
        +GetByLabelsAndStatus(labels, status) []*DeploymentRecord
    }

    %% Strategy Pattern
    class DeploymentStrategy {
        <<interface>>
        +StartDeployment(record) error
        +ProgressDeployment(record) *DeploymentRecord
        +ResetFailedInstances(labels) error
    }
    
    class RollingDeployment {
        -DeploymentStore store
        -StateService inventory
        +StartDeployment(record) error
        +ProgressDeployment(record) *DeploymentRecord
        +ResetFailedInstances(labels) error
    }

    %% Service Layer
    class TriggerService {
        -DeploymentStore store
        -DeploymentStrategy strategy
        +TriggerDeployment(request) error
        +ProgressDeployment() *DeploymentRecord
        +TriggerRollback(labels, config) error
    }

    %% Core relationships
    Instance *-- State : CurrentState
    Instance *-- State : DesiredState
    StateService *-- InventoryStore : uses
    
    DeploymentRequest *-- Configuration : contains
    DeploymentRecord *-- DeploymentRequest : contains
    DeploymentRecord *-- DeploymentProgress : tracks
    
    RollingDeployment ..|> DeploymentStrategy : implements
    RollingDeployment *-- DeploymentStore : uses
    RollingDeployment *-- StateService : uses
    
    TriggerService *-- DeploymentStore : uses
    TriggerService *-- DeploymentStrategy : uses

    %% Key cross-domain relationships
    RollingDeployment ..> Instance : "queries via StateService"
    RollingDeployment ..> State : "creates from DeploymentRequest"
    DeploymentRequest ..> State : "target version"

    %% Notes
    note for RollingDeployment "Core Strategy Implementation:\n1. Queries inventory for matching instances\n2. Updates instance DesiredState in batches\n3. Tracks progress until all instances updated"
    note for StateService "Inventory Interface:\nUsed by deployment strategies\nto query and update instances"
    note for DeploymentRequest "Contains target State:\nCodeVersion + ConfigurationVersion\nmatches Instance.State structure"
```
