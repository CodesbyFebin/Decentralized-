# P2-MULTIHOST-A01

Minimum: >=3 independently controlled physical hosts, explicit failure-domain declaration, distinct host/node identities, real routed/encrypted inter-node network and cross-host workload placement.

Record PHYSICAL_HOSTS, OS_INSTANCES, NODE_IDENTITIES, PHYSICAL_HOST_DOMAIN, OPERATOR_DOMAIN, POWER_DOMAIN, NETWORK_DOMAIN, HYPERVISOR_DOMAIN.

Do not infer operator/power independence from IP addresses. Test host loss, link loss, control-plane restart, replacement on another physical host, continuity and reboot identity persistence.
