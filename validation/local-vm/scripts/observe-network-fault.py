#!/usr/bin/env python3
"""Observe a real host-forwarded SSH/HTTP outage on one QEMU guest.

This is a diagnostic, not a P1 gate verifier. QEMU user-mode NAT in the
current cluster does not supply a guest-to-guest mesh. The scheduled guest
timer heals the interface even if this process or its SSH connection dies.
"""

import argparse
import datetime as dt
import json
import os
from pathlib import Path
import re
import subprocess
import sys
import time


def utc():
    return dt.datetime.now(dt.timezone.utc).isoformat()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--node", required=True, help="Exact name in cluster.json")
    parser.add_argument("--workload-id", required=True)
    parser.add_argument("--http-port", type=int, required=True)
    parser.add_argument("--duration", type=int, default=20, help="Fault duration in seconds")
    parser.add_argument("--output", type=Path, required=True, help="New evidence directory")
    parser.add_argument("--inject", action="store_true", help="Authorize a timed tc fault")
    args = parser.parse_args()
    if not re.fullmatch(r"[A-Za-z0-9_-]{1,64}", args.workload_id):
        parser.error("workload ID must be an alphanumeric identifier")
    if not 1 <= args.http_port <= 65535 or not 5 <= args.duration <= 120:
        parser.error("HTTP port or fault duration out of range")
    if not args.inject:
        parser.error("pass --inject to execute the timed network fault")

    root = Path(__file__).resolve().parents[3]
    state = root / "validation/local-vm/state"
    cluster = json.loads((state / "cluster.json").read_text())
    nodes = [n for n in cluster["node_details"] if n["name"] == args.node]
    if len(nodes) != 1:
        parser.error("node is not unique in cluster.json")
    node = nodes[0]
    pid = int(Path(node["qemu_pid_file"]).read_text().strip())
    os.kill(pid, 0)
    sha = subprocess.check_output(["git", "-C", str(root), "rev-parse", "HEAD"], text=True).strip()
    if sha != cluster.get("source_sha"):
        parser.error("cluster source SHA differs from current HEAD")
    key = Path(cluster["ssh_key_path"])
    if not key.is_file():
        parser.error("SSH key absent")
    args.output.mkdir(parents=True, exist_ok=False)
    ssh_base = ["ssh", "-i", str(key), "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=no",
                "-o", "UserKnownHostsFile=/dev/null", "-o", "ConnectTimeout=3",
                "-p", str(node["ssh_port"]), "ubuntu@127.0.0.1"]
    events = args.output / "events.jsonl"
    traffic = args.output / "traffic.jsonl"

    def record(path, **data):
        with path.open("a", encoding="utf-8") as f:
            f.write(json.dumps({"timestamp": utc(), **data}, sort_keys=True) + "\n")
            f.flush()
            os.fsync(f.fileno())

    def remote(command, timeout=12):
        return subprocess.run(ssh_base + [command], text=True, capture_output=True, timeout=timeout)

    def probe(request_id):
        started = time.monotonic_ns()
        # The HTTP request runs inside the guest; the response crosses its
        # actual network interface and QEMU's forwarded SSH connection.
        command = ("python3 -c 'import json,urllib.request; "
                   f"r=urllib.request.urlopen(\"http://127.0.0.1:{args.http_port}/\",timeout=2); "
                   "print(json.dumps({\"status\":r.status,\"body\":r.read().decode()}))'")
        error = None
        status = None
        identity = None
        try:
            result = remote(command, timeout=6)
            if result.returncode:
                error = f"ssh_exit_{result.returncode}: {result.stderr[-300:]}"
            else:
                envelope = json.loads(result.stdout)
                status = envelope["status"]
                identity = json.loads(envelope["body"]).get("workload_id")
        except (subprocess.TimeoutExpired, ValueError, KeyError) as exc:
            error = str(exc)
        ok = status == 200 and identity == args.workload_id and error is None
        record(traffic, request_id=request_id, workload_id=args.workload_id,
               node_id=args.node, http_status=status, response_identity=identity,
               latency_ms=(time.monotonic_ns() - started) / 1e6, error=error,
               success=ok)
        return ok

    record(events, event="START", source_sha=sha, qemu_pid=pid,
           topology_scope="host-forwarded-ssh-to-guest", campaign="DIAGNOSTIC")
    # Read route and qdisc before touching the guest. Refuse an existing root
    # qdisc; deleting it during heal could otherwise destroy operator policy.
    pre = remote("ip -j route get 10.0.2.2; ip -j -s link; tc -j qdisc show", timeout=12)
    (args.output / "network-before.txt").write_text(pre.stdout + "\nSTDERR:\n" + pre.stderr)
    if pre.returncode:
        record(events, event="BLOCKED", reason="guest route or tc observation failed")
        return 2
    try:
        route = json.loads(pre.stdout.splitlines()[0])[0]
        interface = route["dev"]
    except (ValueError, IndexError, KeyError, TypeError):
        record(events, event="BLOCKED", reason="cannot identify guest egress interface")
        return 2
    if not re.fullmatch(r"[A-Za-z0-9_.-]{1,32}", interface):
        record(events, event="BLOCKED", reason="unsafe interface name")
        return 2
    qdisc = remote(f"tc -j qdisc show dev {interface}")
    (args.output / "qdisc-before.json").write_text(qdisc.stdout)
    if qdisc.returncode:
        record(events, event="BLOCKED", reason="tc unavailable in guest")
        return 2
    existing = json.loads(qdisc.stdout)
    root_qdiscs = [item for item in existing if item.get("root") is True]
    if any(item.get("kind") not in {"fq_codel", "pfifo_fast", "noqueue"} for item in root_qdiscs):
        record(events, event="BLOCKED", reason="guest has a nondefault root qdisc")
        return 2
    if not probe("pre-1") or not probe("pre-2"):
        record(events, event="BLOCKED", reason="HTTP identity precondition failed")
        return 2

    # systemd-run creates independent timers. Heal is scheduled first; the
    # injection command is scheduled two seconds later, without depending on
    # an SSH session that will be cut by the fault.
    unit_suffix = f"{os.getpid()}-{int(time.time())}"
    heal_unit = f"dh-fault-heal-{unit_suffix}"
    inject_unit = f"dh-fault-inject-{unit_suffix}"
    heal_delay = args.duration + 2
    heal = remote(f"sudo -n systemd-run --quiet --unit={heal_unit} --on-active={heal_delay}s "
                  f"/usr/sbin/tc qdisc del dev {interface} root")
    (args.output / "heal-schedule.txt").write_text(heal.stdout + heal.stderr)
    if heal.returncode:
        record(events, event="BLOCKED", reason="heal watchdog could not be scheduled")
        return 2
    inject = remote(f"sudo -n systemd-run --quiet --unit={inject_unit} --on-active=2s "
                    f"/usr/sbin/tc qdisc replace dev {interface} root netem loss 100%")
    (args.output / "inject-schedule.txt").write_text(inject.stdout + inject.stderr)
    if inject.returncode:
        record(events, event="BLOCKED", reason="injection timer could not be scheduled; heal timer remains")
        return 2
    record(events, event="TIMERS_SCHEDULED", interface=interface, heal_delay_seconds=heal_delay,
           inject_unit=inject_unit, heal_unit=heal_unit)
    first_failure = None
    first_restoration = None
    # The loop attempts HTTP throughout the fault window, including failures.
    deadline = time.monotonic() + heal_delay + 25
    request_id = 0
    while time.monotonic() < deadline:
        request_id += 1
        ok = probe(f"fault-{request_id}")
        if not ok and first_failure is None:
            first_failure = utc()
            record(events, event="FIRST_TRAFFIC_FAILURE", observed_at=first_failure)
        if ok and first_failure is not None:
            first_restoration = utc()
            record(events, event="FIRST_TRAFFIC_RESTORATION", observed_at=first_restoration)
            break
        time.sleep(0.25)
    post = remote(f"set -e; tc -j qdisc show dev {interface}; "
                  f"systemctl show {inject_unit}.service -p Result -p ExecMainStatus; "
                  f"systemctl show {heal_unit}.service -p Result -p ExecMainStatus")
    (args.output / "network-after.txt").write_text(post.stdout + "\nSTDERR:\n" + post.stderr)
    journal = remote("sudo -n journalctl --utc --no-pager -o short-iso-precise "
                     f"-u {inject_unit}.service -u {heal_unit}.service", timeout=12)
    (args.output / "guest-fault-journal.txt").write_text(journal.stdout + "\nSTDERR:\n" + journal.stderr)
    samples = [json.loads(line) for line in traffic.read_text().splitlines()]
    failure_count = sum(not sample["success"] for sample in samples)
    downtime = None
    if first_failure and first_restoration:
        downtime = (dt.datetime.fromisoformat(first_restoration) -
                    dt.datetime.fromisoformat(first_failure)).total_seconds()
    (args.output / "traffic-summary.json").write_text(json.dumps({
        "source_sha": sha, "campaign": "DIAGNOSTIC", "scope": "host-forwarded-ssh-to-guest",
        "total": len(samples), "success": len(samples) - failure_count,
        "failed": failure_count, "downtime_seconds": downtime,
        "production_detection": "UNKNOWN", "mesh_convergence": "UNKNOWN",
    }, indent=2) + "\n")
    service_results = re.findall(r"^Result=([^\n]+)$", post.stdout, re.MULTILINE)
    service_statuses = re.findall(r"^ExecMainStatus=([^\n]+)$", post.stdout, re.MULTILINE)
    services_succeeded = (service_results == ["success", "success"] and
                          service_statuses == ["0", "0"])
    qdisc_healed = '"netem"' not in post.stdout
    record(events, event="END", first_failure=first_failure, first_restoration=first_restoration,
           injection_mechanism="guest tc netem scheduled timer", detection="UNKNOWN",
           mesh_partition="UNKNOWN", guest_journal_exit=journal.returncode,
           post_observation_exit=post.returncode, services_succeeded=services_succeeded,
           qdisc_healed=qdisc_healed)
    # A zero exit means the diagnostic collected an outage and a restoration.
    # It does not certify the P1 detection, mesh, or reconciliation gates.
    return 0 if (first_failure and first_restoration and post.returncode == 0
                 and journal.returncode == 0 and services_succeeded and qdisc_healed) else 2


if __name__ == "__main__":
    try:
        sys.exit(main())
    except (OSError, ValueError, subprocess.SubprocessError) as exc:
        print(f"BLOCKED: {exc}", file=sys.stderr)
        sys.exit(2)
