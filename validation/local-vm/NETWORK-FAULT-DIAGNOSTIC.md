# Timed network fault diagnostic

`observe-network-fault.py` captures a real guest interface fault and HTTP
requests across the fault window. It is a **diagnostic**, not a decisive P1
verifier. The current QEMU cluster starts each VM with a separate user-mode NAT
network and a forwarded SSH port. This can test host-to-guest traffic loss;
it cannot establish a guest-to-guest mesh partition or convergence.

Run only against an already booted guest with a known, running HTTP workload
that returns JSON containing its `workload_id`:

```bash
python3 validation/local-vm/scripts/observe-network-fault.py \
  --node dh-node-2 --workload-id workload-api-02 --http-port 18082 \
  --duration 20 --output /tmp/p1-network-diagnostic-001 --inject
```

The script checks the exact cluster source SHA, live QEMU PID, SSH key, guest
route, `tc` qdisc, and two successful HTTP identities before scheduling the
fault. It schedules an independent systemd heal timer **before** the injection
timer. The guest applies 100% egress packet loss with `tc netem`, then removes
the qdisc after the bounded duration. The host attempts HTTP over forwarded
SSH throughout, recording each request, response identity, latency, and error
in `traffic.jsonl`. It records the guest service journal and interface state
after heal. An existing nondefault root qdisc blocks the diagnostic.

Exit zero means the diagnostic observed traffic loss and restoration and both
guest timer services reported success. It does not prove production heartbeat
detection, scheduler eligibility, reconciliation, or guest-to-guest mesh
behavior. Those remain `UNKNOWN` until separate runtime observations and the
authoritative evidence verifier establish them. Preserve this directory raw;
do not convert its diagnostic exit code into a P1 gate PASS.
