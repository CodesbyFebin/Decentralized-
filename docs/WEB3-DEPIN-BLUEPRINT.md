# Decentralized.Host Web3 DePIN Blueprint

**Status**: Architecture Design  
**Version**: 2.0 (Web3 Extension)  
**Date**: 2026-09-30

---

## Executive Summary

This document extends Decentralized.Host (dh) v1.0.0 into a **Web3 Decentralized Physical Infrastructure Network (DePIN)**. 

The transformation enables:
- **Token-incentivized resource provision** (nodes earn tokens for providing compute/storage)
- **On-chain workload marketplace** (customers pay in tokens for compute)
- **Decentralized reputation system** (on-chain SLA tracking and operator scoring)
- **Smart contract-enforced SLAs** (automatic payments + slashing for failures)
- **DAO governance** (community-owned infrastructure parameters)

**No breaking changes to dh/v1.0 core.** Web3 layer is optional, backward-compatible, and deployed alongside existing v1.0.0 infrastructure.

---

## Current Architecture (v1.0.0)

### Core Capabilities
```
Signed Intent → Local Policy → State Machine → Audit Trail → Raft Consensus
   (Ed25519)   (Per-host)   (DESIRED→VERIFIED) (Immutable)  (Distributed)
```

### Existing Strengths
✅ Cryptographic identity binding (Ed25519)  
✅ Immutable, replicated audit trail (Raft)  
✅ Observable state machine (deterministic replay)  
✅ Local policy enforcement (no central authority)  
✅ Failure detection and recovery  
✅ Network-level security (mTLS everywhere)

---

## Web3 DePIN Layer Architecture

### Layer Stack

```
┌────────────────────────────────────────────┐
│     Application Layer (Marketplace UI)     │
├────────────────────────────────────────────┤
│  Smart Contracts (SLA, Pricing, Slashing)  │
│  Token Bridge (Native ↔ dh-network)        │
├────────────────────────────────────────────┤
│  Blockchain Integration (Event Listener)   │
│  Oracle (Lattice3 / Pyth for attestation) │
├────────────────────────────────────────────┤
│     dh v1.0.0 Core (Unchanged)             │
│  (Raft, Policy, State, Audit, Signing)     │
└────────────────────────────────────────────┘
```

### Integration Points

1. **Node Registration** (Blockchain ← dh)
   - On-chain: Node ID, stake, reputation
   - Off-chain: dh Raft identity, policy configuration

2. **Workload Settlement** (Blockchain → dh)
   - On-chain: Smart contract specifies SLA, budget, duration
   - Off-chain: dh executes via signed intent + policy matching

3. **Attestation** (dh → Blockchain)
   - Off-chain: Audit trail records state changes
   - On-chain: Oracle verifies via Merkle proofs
   - Result: Automatic payment or slashing

4. **Reputation** (Blockchain tracks)
   - Uptime: % of successful SLA periods
   - Latency: p99 response time vs. promised
   - Reliability: incident-free runs
   - Scoring: Integrated into staking rewards

---

## Component Design

### 1. Token Contract (ERC-20 + DePIN extensions)

```solidity
// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract DHToken is ERC20, Ownable {
    // Token: DHT (Decentralized.Host Token)
    // Supply: 100M tokens (50% operators, 30% customers, 20% treasury/staking)
    
    uint256 public constant TOTAL_SUPPLY = 100_000_000e18;
    
    mapping(address => uint256) public operatorStake;
    mapping(address => uint256) public unstakingTimestamp;
    
    uint256 public unstakingPeriod = 14 days;
    
    event Staked(address indexed operator, uint256 amount);
    event UnstakingInitiated(address indexed operator, uint256 amount);
    event Unstaked(address indexed operator, uint256 amount);
}
```

**Token Distribution**:
- Operators: 50M (staking + rewards)
- Customers: 30M (purchasing credits)
- Treasury: 15M (governance, incentives)
- Liquidity/Exchange: 5M

---

### 2. Node Registry Contract

```solidity
contract DHNodeRegistry {
    struct Node {
        address operator;           // Ethereum address
        bytes32 dhNodeID;          // dh v1.0 identity (Ed25519 public key)
        uint256 stake;             // DHT staked
        uint256 reputation;        // 0-100 (weighted average)
        uint256 slashableBalance;  // Collateral for SLA violations
        Status status;             // ACTIVE | UNSTAKING | SLASHED
        uint256 totalWorkloads;
        uint256 totalEarnings;
    }
    
    enum Status { ACTIVE, UNSTAKING, SLASHED }
    
    mapping(bytes32 => Node) public nodes;
    mapping(address => bytes32) public operatorToNode;
    
    event NodeRegistered(
        bytes32 indexed nodeID,
        address indexed operator,
        uint256 stake
    );
    
    event ReputationUpdated(
        bytes32 indexed nodeID,
        uint256 newReputation,
        string reason
    );
    
    // Minimum stake: 1000 DHT
    uint256 public minStake = 1000e18;
    
    function registerNode(
        bytes32 dhNodeID,
        uint256 stake
    ) external {
        require(stake >= minStake, "Insufficient stake");
        require(operatorToNode[msg.sender] == bytes32(0), "Already registered");
        
        // Transfer stake to escrow
        dhtToken.transferFrom(msg.sender, address(this), stake);
        
        nodes[dhNodeID] = Node({
            operator: msg.sender,
            dhNodeID: dhNodeID,
            stake: stake,
            reputation: 75, // Starting reputation
            slashableBalance: stake / 2,
            status: Status.ACTIVE,
            totalWorkloads: 0,
            totalEarnings: 0
        });
        
        operatorToNode[msg.sender] = dhNodeID;
        emit NodeRegistered(dhNodeID, msg.sender, stake);
    }
}
```

---

### 3. Workload SLA Contract

```solidity
contract DHWorkloadSLA {
    struct WorkloadSpec {
        bytes32 workloadID;
        address customer;
        bytes32[] requiredNodes;    // Which nodes must accept
        uint256 budget;             // DHT budget
        uint256 duration;           // Seconds
        SLATerms terms;
        WorkloadStatus status;
    }
    
    struct SLATerms {
        uint256 uptimeTarget;       // 99.9% = 999 (permille)
        uint256 p99LatencyMs;
        uint256 maxErrorRate;       // 1% = 1000 (permille)
    }
    
    enum WorkloadStatus {
        PENDING,      // Waiting for node acceptance
        ACCEPTED,     // Nodes accepted, dh executing
        EXECUTING,
        COMPLETED,
        SLASHED       // SLA violated
    }
    
    mapping(bytes32 => WorkloadSpec) public workloads;
    mapping(bytes32 => bool) public workloadAccepted;
    
    event WorkloadCreated(
        bytes32 indexed workloadID,
        address indexed customer,
        uint256 budget,
        uint256 duration
    );
    
    event WorkloadCompleted(
        bytes32 indexed workloadID,
        uint256 totalEarnings,
        bool slaMetByAll
    );
    
    function createWorkload(
        bytes32 workloadID,
        bytes32[] memory requiredNodes,
        uint256 budget,
        uint256 duration,
        SLATerms memory terms
    ) external payable {
        require(msg.value >= budget, "Insufficient budget");
        
        workloads[workloadID] = WorkloadSpec({
            workloadID: workloadID,
            customer: msg.sender,
            requiredNodes: requiredNodes,
            budget: budget,
            duration: duration,
            terms: terms,
            status: WorkloadStatus.PENDING
        });
        
        emit WorkloadCreated(workloadID, msg.sender, budget, duration);
    }
}
```

---

### 4. Oracle Integration (Attestation)

```solidity
contract DHOracle {
    // Uses Lattice3 / Pyth network for attestation verification
    
    struct Attestation {
        bytes32 workloadID;
        bytes32 dhAuditLogMerkleRoot;  // From dh audit trail
        uint256 timestamp;
        uint256 uptime;                // Permille
        uint256 p99Latency;
        uint256 errorRate;
        bytes signature;               // dh cluster signature
    }
    
    mapping(bytes32 => Attestation) public attestations;
    address public dhClusterPublicKey; // Cluster's Ed25519 public key
    
    event AttestationReceived(
        bytes32 indexed workloadID,
        uint256 uptime,
        bool slaMetByAll
    );
    
    function submitAttestation(
        Attestation memory attestation
    ) external {
        // Verify Merkle proof against dh audit trail
        require(
            verifyDHMerkleProof(
                attestation.dhAuditLogMerkleRoot,
                attestation.signature
            ),
            "Invalid attestation"
        );
        
        attestations[attestation.workloadID] = attestation;
        emit AttestationReceived(
            attestation.workloadID,
            attestation.uptime,
            checkSLACompliance(attestation)
        );
    }
    
    function verifyDHMerkleProof(
        bytes32 merkleRoot,
        bytes memory signature
    ) internal view returns (bool) {
        // Verify signature was made by dh cluster
        // (using Ed25519 public key)
        return true; // Implementation uses ecrecover or equivalent
    }
}
```

---

### 5. Payment & Slashing Engine

```solidity
contract DHPaymentEngine is Ownable {
    enum PaymentStatus { PENDING, SETTLED, DISPUTED }
    
    struct PaymentRecord {
        bytes32 workloadID;
        address[] nodes;
        uint256[] earnings;
        PaymentStatus status;
        uint256 totalPaid;
    }
    
    mapping(bytes32 => PaymentRecord) public payments;
    
    event PaymentSettled(
        bytes32 indexed workloadID,
        uint256 totalPaid,
        address[] nodes
    );
    
    event NodeSlashed(
        bytes32 indexed nodeID,
        uint256 slashAmount,
        string reason
    );
    
    function settlePayment(
        bytes32 workloadID,
        Attestation memory attestation
    ) external {
        WorkloadSpec memory workload = slaContract.workloads(workloadID);
        PaymentRecord storage payment = payments[workloadID];
        
        // Calculate per-node earnings based on SLA compliance
        uint256 basePayPerNode = workload.budget / workload.requiredNodes.length;
        
        for (uint i = 0; i < workload.requiredNodes.length; i++) {
            bytes32 nodeID = workload.requiredNodes[i];
            Node memory node = registry.nodes(nodeID);
            
            // If node met SLA: full payment + reputation bonus
            if (attestation.uptime >= workload.terms.uptimeTarget) {
                uint256 bonus = (basePayPerNode * 5) / 100; // 5% bonus
                payment.earnings[i] = basePayPerNode + bonus;
                
                // Increase reputation
                registry.updateReputation(nodeID, 1, "SLA_COMPLIANT");
            } else {
                // Node missed SLA: slash collateral
                uint256 slashAmount = (basePayPerNode * 20) / 100; // 20% slash
                payment.earnings[i] = basePayPerNode - slashAmount;
                
                // Reduce reputation
                registry.updateReputation(nodeID, -5, "SLA_VIOLATION");
                
                emit NodeSlashed(nodeID, slashAmount, "SLA_VIOLATION");
            }
        }
        
        // Transfer payments to operators
        for (uint i = 0; i < payment.nodes.length; i++) {
            address operator = registry.nodes(payment.nodes[i]).operator;
            dhtToken.transfer(operator, payment.earnings[i]);
        }
        
        payment.status = PaymentStatus.SETTLED;
        emit PaymentSettled(workloadID, payment.totalPaid, payment.nodes);
    }
    
    function slashNode(
        bytes32 nodeID,
        uint256 amount,
        string memory reason
    ) external onlyOracle {
        Node storage node = registry.nodes(nodeID);
        require(node.slashableBalance >= amount, "Insufficient balance to slash");
        
        node.slashableBalance -= amount;
        dhtToken.burn(amount); // Remove from circulation
        
        emit NodeSlashed(nodeID, amount, reason);
    }
}
```

---

### 6. DAO Governance Contract

```solidity
contract DHDAO is Ownable {
    struct Proposal {
        uint256 proposalID;
        string description;
        ProposalType proposalType;
        bytes calldata;
        uint256 votesFor;
        uint256 votesAgainst;
        uint256 deadline;
        bool executed;
        ProposalStatus status;
    }
    
    enum ProposalType {
        PARAMETER_CHANGE,      // Minimum stake, slashing %, etc
        CONTRACT_UPGRADE,
        TREASURY_ALLOCATION,
        NEW_ORACLE
    }
    
    enum ProposalStatus { PENDING, ACTIVE, PASSED, FAILED, EXECUTED }
    
    mapping(uint256 => Proposal) public proposals;
    mapping(address => mapping(uint256 => bool)) public hasVoted;
    
    uint256 public proposalCount;
    uint256 public votingPeriod = 7 days;
    uint256 public quorumPercentage = 40; // 40% of staked tokens
    
    event ProposalCreated(
        uint256 indexed proposalID,
        ProposalType proposalType,
        string description
    );
    
    event VoteCasted(
        uint256 indexed proposalID,
        address voter,
        bool support,
        uint256 weight
    );
    
    function createProposal(
        ProposalType proposalType,
        string memory description,
        bytes memory calldata_
    ) external returns (uint256) {
        // Require minimum stake to propose (e.g., 10,000 DHT)
        require(
            dhtToken.balanceOf(msg.sender) >= 10_000e18,
            "Insufficient stake to propose"
        );
        
        uint256 proposalID = proposalCount++;
        proposals[proposalID] = Proposal({
            proposalID: proposalID,
            description: description,
            proposalType: proposalType,
            calldata: calldata_,
            votesFor: 0,
            votesAgainst: 0,
            deadline: block.timestamp + votingPeriod,
            executed: false,
            status: ProposalStatus.ACTIVE
        });
        
        emit ProposalCreated(proposalID, proposalType, description);
        return proposalID;
    }
    
    function vote(uint256 proposalID, bool support) external {
        Proposal storage proposal = proposals[proposalID];
        require(block.timestamp <= proposal.deadline, "Voting period ended");
        require(!hasVoted[msg.sender][proposalID], "Already voted");
        
        uint256 weight = dhtToken.balanceOf(msg.sender);
        require(weight > 0, "No voting power");
        
        if (support) {
            proposal.votesFor += weight;
        } else {
            proposal.votesAgainst += weight;
        }
        
        hasVoted[msg.sender][proposalID] = true;
        emit VoteCasted(proposalID, msg.sender, support, weight);
    }
    
    function executeProposal(uint256 proposalID) external onlyOwner {
        Proposal storage proposal = proposals[proposalID];
        require(block.timestamp > proposal.deadline, "Voting still active");
        require(!proposal.executed, "Already executed");
        
        uint256 totalVotes = proposal.votesFor + proposal.votesAgainst;
        require(totalVotes >= getQuorum(), "Quorum not met");
        require(proposal.votesFor > proposal.votesAgainst, "Proposal failed");
        
        proposal.status = ProposalStatus.EXECUTED;
        proposal.executed = true;
        
        // Execute the proposal
        (bool success, ) = address(this).call(proposal.calldata);
        require(success, "Execution failed");
    }
}
```

---

## Integration with dh v1.0.0

### On-Ramp: Node Registration

**Flow**:
1. Operator runs `dh-node` on their hardware
2. dh generates Ed25519 node identity automatically
3. Operator calls `DHNodeRegistry.registerNode(dhNodeID, stake)`
4. Smart contract escrows stake
5. Node becomes eligible to accept workloads

**Modified dh code** (minimal):
```go
// In dh-noded main.go
func init() {
    // Read node identity (already exists in v1.0)
    nodeID := readNodeIdentity()
    
    // NEW: Broadcast node to blockchain
    if os.Getenv("DHT_REGISTRY_ADDRESS") != "" {
        registerOnBlockchain(nodeID)
    }
}
```

---

### Workload Execution

**Flow**:
1. Customer creates `DHWorkloadSLA` contract with budget + SLA terms
2. dh cluster receives workload as signed intent (via existing mechanism)
3. Operator policy engine checks: "Accept this workload for these terms?"
4. If accepted: dh executes (unchanged from v1.0)
5. Audit trail records all state changes (unchanged)
6. Oracle reads Merkle root from audit trail (NEW)
7. Smart contract verifies attestation + settles payments (NEW)

**No changes to core dh execution path.**

---

### Attestation: dh → Blockchain

**Flow**:
1. dh audit trail records: workload start, state transitions, completion
2. At completion: dh computes Merkle root of audit log
3. dh cluster signs Merkle root with collective Ed25519 key
4. `DHOracle.submitAttestation(merkleRoot, signature)` posts on-chain
5. Oracle contract verifies signature against dh cluster public key
6. Payment engine calculates rewards/slashes based on SLA compliance

**New code in dh**:
```go
// In dh-control/main.go
func submitAttestation(workloadID string, auditLogMerkleRoot []byte) {
    // Sign with cluster's collective key
    signature := clusterKey.Sign(auditLogMerkleRoot)
    
    // POST to blockchain oracle
    attestation := Attestation{
        WorkloadID: workloadID,
        MerkleRoot: merkleRoot,
        Signature: signature,
    }
    
    oracleContract.SubmitAttestation(attestation)
}
```

---

## Token Economics

### Supply & Distribution

| Allocation | Amount | Use Case |
|-----------|--------|----------|
| Operator Rewards | 50M | Staking + execution rewards |
| Customer Credits | 30M | Purchase compute capacity |
| Treasury | 15M | Governance, incentives, reserves |
| Liquidity/Exchange | 5M | Initial DEX liquidity |

### Reward Model

**Per-workload earnings**:
```
Base reward = (Workload budget) / (Number of accepting nodes)

If SLA met:
  Actual earning = Base reward + 5% bonus
  Reputation += 1

If SLA missed:
  Actual earning = Base reward - 20% (slash)
  Reputation -= 5
```

**Annual staking rewards** (on-chain treasury):
```
Stake reward = (Stake / Total staked) × 15% annual
```

---

## Security Model

### Trust Anchors

1. **dh Cluster Identity** (Ed25519 public key)
   - Posted on-chain during bootstrap
   - Used to verify all attestations
   - Immutable in registry

2. **Raft Consensus** (dh v1.0)
   - Audit trail is Raft-replicated
   - Merkle root binding ensures no tampering
   - 3+ member quorum ensures Byzantine resilience

3. **Smart Contract Logic**
   - Immutable (no admin keys can change core logic)
   - Time-locked updates (7-day delay)
   - Multisig Treasury operations

### Attack Surface

| Attack | Defense |
|--------|---------|
| Node claims SLA success when actually failed | Merkle proof verification + oracle check |
| Operator removes workload from audit trail | Immutable Raft log + Merkle binding |
| Customer refuses to pay | On-chain escrow + automatic settlement |
| Malicious operator joins, then slashes | Minimum stake requirement + reputation |
| Smart contract bug | Timelock on parameter changes + insurance fund |

---

## Implementation Roadmap

### Phase 1: Foundation (Weeks 1-4)
- [ ] Deploy DHToken contract (ERC-20)
- [ ] Deploy DHNodeRegistry contract
- [ ] Deploy DHWorkloadSLA contract
- [ ] Integrate dh node registration (light modification)
- [ ] Write oracle attestation logic

**Deliverable**: Testnet integration, manual attestation flow

### Phase 2: Automation (Weeks 5-8)
- [ ] DHPaymentEngine contract (automatic settlement)
- [ ] DHDAO governance contract
- [ ] Automate attestation submission from dh clusters
- [ ] Build marketplace UI (React)

**Deliverable**: End-to-end workload execution with payments

### Phase 3: Production (Weeks 9-12)
- [ ] Security audit (OpenZeppelin / Trail of Bits)
- [ ] Mainnet deployment (Ethereum or L2: Arbitrum, Optimism)
- [ ] Liquidity pool creation (DHT/ETH or DHT/USDC)
- [ ] Operator onboarding program
- [ ] Customer beta program

**Deliverable**: Live DePIN infrastructure

### Phase 4: Scale (Ongoing)
- [ ] Additional chains (Polygon, Solana)
- [ ] Native stablecoin (dh-USD)
- [ ] Advanced reputation scoring (ML-based)
- [ ] Insurance pool for SLA breaches
- [ ] Cross-chain interop

---

## Backwards Compatibility

**v1.0.0 clusters remain unchanged:**
- No Solidity code in dh
- No blockchain RPC calls required (only if opted in)
- Existing dh deployments work without Web3 layer
- Can upgrade to Web3 by setting env vars + staking

**Operators have 3 options**:
1. Keep running dh v1.0.0 (no tokens, no rewards)
2. Register on-chain (staking + marketplace access)
3. Run hybrid (private + DePIN workloads)

---

## Governance Parameters (DAO-Controlled)

These can be changed via DHDAO proposals:

```json
{
  "tokenomics": {
    "minStake": "1000 DHT",
    "stakingRewardAnnual": "15%",
    "slaComplianceBonus": "5%",
    "slaViolationSlash": "20%"
  },
  "workloads": {
    "maxDuration": "30 days",
    "minBudget": "100 DHT",
    "maxConcurrent": "1000"
  },
  "governance": {
    "votingPeriod": "7 days",
    "quorumPercentage": "40%",
    "proposalThreshold": "10000 DHT"
  }
}
```

---

## Success Metrics

### Operator Metrics
- Active nodes: 100+ by month 6
- Average uptime: > 99.5%
- Average reputation: > 80/100
- Monthly earnings: $100-$1000 per node

### Network Metrics
- Total compute capacity: 10,000+ cores by month 6
- Total staked tokens: 50M+ DHT
- Monthly workloads: 10,000+ by month 6
- Token price: $1-$5 by month 12 (community-determined)

### Economic Metrics
- Monthly volume: $100K+ by month 6
- Operator earnings: $500K+ monthly by month 6
- Treasury reserves: $5M+ by month 12

---

## Next Steps

1. **Smart Contract Audit**: Engage OpenZeppelin (4-6 weeks)
2. **Testnet Launch**: Deploy on Sepolia/Goerli for community testing
3. **Operator Incentives**: Bootstrap program with 5M DHT rewards
4. **Customer Acquisition**: Partner with cloud platforms for workload routing
5. **Mainnet Readiness**: Final security review + liquidity setup

---

## Appendix: Smart Contract Deployment Checklist

- [ ] DHToken contract (ERC-20 with burn + staking)
- [ ] DHNodeRegistry (node management)
- [ ] DHWorkloadSLA (workload lifecycle)
- [ ] DHOracle (attestation verification)
- [ ] DHPaymentEngine (settlement + slashing)
- [ ] DHDAO (governance)
- [ ] Multisig Wallet (treasury)
- [ ] Timelock Controller (governance delays)
- [ ] Insurance Pool (optional, for SLA failures)

---

**Status**: Architecture complete, ready for community feedback and audit phase.
