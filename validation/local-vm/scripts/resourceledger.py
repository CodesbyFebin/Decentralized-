#!/usr/bin/env python3
"""Atomic local VM ResourceLedger transactions under a stable sidecar lock.

Model A: available = total - owner_reserve - reserved - allocated.
An allocation with a reservation ID consumes that exact active reservation,
moving its claim from RESERVED to ALLOCATED without changing availability.
"""

import argparse
from contextlib import contextmanager
from datetime import datetime, timezone
from decimal import Decimal, InvalidOperation
import fcntl
import json
import os
from pathlib import Path
import sys
import tempfile
import uuid


DIMENSIONS = (
    ("cpu", "cpu_cores", "cpu_owner_reserve", "cpu_reserved", "cpu_allocated"),
    ("memory_mb", "memory_mb", "memory_owner_reserve_mb", "memory_reserved_mb", "memory_allocated_mb"),
    ("disk_gb", "disk_gb", "disk_owner_reserve_gb", "disk_reserved_gb", "disk_allocated_gb"),
)


class LedgerError(Exception):
    pass


def number(value):
    if isinstance(value, bool):
        raise LedgerError("Boolean is not a resource quantity")
    try:
        value = Decimal(str(value))
    except InvalidOperation as exc:
        raise LedgerError("Invalid resource quantity") from exc
    if not value.is_finite() or value < 0:
        raise LedgerError("Resource quantity must be finite and nonnegative")
    return value


def requested(values):
    amounts = dict(zip((d[0] for d in DIMENSIONS), (number(v) for v in values)))
    if any(v <= 0 for v in amounts.values()):
        raise LedgerError("Requested resources must be positive")
    return amounts


def active_sum(node, collection, field):
    return sum((number(item[field]) for item in node.get(collection, [])
                if item["state"] == "ACTIVE"), Decimal(0))


def validate(ledger):
    nodes = ledger.get("node_capacity")
    if not isinstance(nodes, list) or not nodes:
        raise LedgerError("Missing node capacity")
    names = [node.get("node") for node in nodes]
    if len(set(names)) != len(names):
        raise LedgerError("Duplicate node names")
    ids = set()
    workloads = set()
    for node in nodes:
        for collection, key in (("reservations", "reservation_id"), ("allocations", "allocation_id")):
            for item in node.get(collection, []):
                identifier = item[key]
                if identifier in ids:
                    raise LedgerError(f"Duplicate record ID {identifier}")
                ids.add(identifier)
                if collection == "allocations" and item["state"] == "ACTIVE":
                    if item["workload_id"] in workloads:
                        raise LedgerError("Duplicate active workload")
                    workloads.add(item["workload_id"])
        for field, total, owner, reserved, allocated in DIMENSIONS:
            cap, own, res, alloc = (number(node[k]) for k in (total, owner, reserved, allocated))
            if res != active_sum(node, "reservations", field):
                raise LedgerError(f"{node['node']} {field}: reservation counter mismatch")
            if alloc != active_sum(node, "allocations", field):
                raise LedgerError(f"{node['node']} {field}: allocation counter mismatch")
            if cap - own - res - alloc < 0:
                raise LedgerError(f"{node['node']} {field}: negative availability")
    recorded = ledger.get("summary", {})
    for _, _, owner, reserved, allocated in DIMENSIONS:
        for field in (owner, reserved, allocated):
            if field in recorded and number(recorded[field]) != sum(
                    (number(node[field]) for node in nodes), Decimal(0)):
                raise LedgerError(f"Summary counter mismatch: {field}")


def summary(ledger):
    current = ledger.setdefault("summary", {})
    for _, _, owner, reserved, allocated in DIMENSIONS:
        for field in (owner, reserved, allocated):
            current[field] = sum((number(n[field]) for n in ledger["node_capacity"]), Decimal(0))


def serialize(value):
    if isinstance(value, Decimal):
        return int(value) if value == int(value) else float(value)
    raise TypeError(type(value).__name__)


@contextmanager
def transaction(path):
    if not path.is_file():
        raise LedgerError(f"Ledger absent: {path}")
    with open(str(path) + ".lock", "a+b") as lock:
        fcntl.flock(lock, fcntl.LOCK_EX)
        with path.open(encoding="utf-8") as stream:
            ledger = json.load(stream, parse_float=Decimal)
        validate(ledger)
        yield ledger
        summary(ledger)
        validate(ledger)
        descriptor, name = tempfile.mkstemp(prefix=path.name + ".tmp.", dir=path.parent)
        try:
            with os.fdopen(descriptor, "w", encoding="utf-8") as stream:
                json.dump(ledger, stream, default=serialize, indent=2)
                stream.write("\n")
                stream.flush()
                os.fsync(stream.fileno())
            os.replace(name, path)
            directory = os.open(path.parent, os.O_RDONLY)
            try:
                os.fsync(directory)
            finally:
                os.close(directory)
        finally:
            if os.path.exists(name):
                os.unlink(name)


def find_node(ledger, name):
    for node in ledger["node_capacity"]:
        if node["node"] == name:
            return node
    raise LedgerError(f"Unknown node {name}")


def find_record(ledger, collection, key, identifier):
    matches = [(node, record) for node in ledger["node_capacity"]
               for record in node.get(collection, []) if record[key] == identifier]
    if len(matches) != 1:
        raise LedgerError(f"Expected one {key}={identifier}, found {len(matches)}")
    return matches[0]


def stamp():
    return datetime.now(timezone.utc).isoformat()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--ledger", type=Path, default=Path(__file__).resolve().parents[1] / "state/resourceledger.json")
    commands = parser.add_subparsers(dest="command", required=True)
    for command in ("allocate", "reserve"):
        item = commands.add_parser(command)
        item.add_argument("node")
        item.add_argument("cpu")
        item.add_argument("memory_mb")
        item.add_argument("disk_gb")
        item.add_argument("workload_id")
        if command == "allocate":
            item.add_argument("--reservation-id")
    for command, identifier in (("deallocate", "allocation_id"), ("release-reservation", "reservation_id")):
        item = commands.add_parser(command)
        item.add_argument(identifier)
    item = commands.add_parser("owner-reserve")
    item.add_argument("node")
    item.add_argument("cpu")
    item.add_argument("memory_mb")
    item.add_argument("disk_gb")
    args = parser.parse_args()
    message = ""
    with transaction(args.ledger) as ledger:
        if args.command in ("allocate", "reserve", "owner-reserve"):
            node = find_node(ledger, args.node)
            amounts = requested((args.cpu, args.memory_mb, args.disk_gb)) if args.command != "owner-reserve" else dict(
                zip((d[0] for d in DIMENSIONS), (number(v) for v in (args.cpu, args.memory_mb, args.disk_gb))))
        if args.command == "owner-reserve":
            for field, _, owner, _, _ in DIMENSIONS:
                node[owner] = amounts[field]
            message = "STATUS: OWNER_RESERVE_SET"
        elif args.command == "reserve":
            for field, total, owner, reserved, allocated in DIMENSIONS:
                if number(node[total]) - number(node[owner]) - number(node[reserved]) - number(node[allocated]) < amounts[field]:
                    raise LedgerError(f"Insufficient {field} on {args.node}")
                node[reserved] = number(node[reserved]) + amounts[field]
            identifier = "res-" + uuid.uuid4().hex
            node.setdefault("reservations", []).append({"reservation_id": identifier, "workload_id": args.workload_id,
                **amounts, "state": "ACTIVE", "created_at": stamp()})
            message = "STATUS: RESERVED\nReservation ID: " + identifier
        elif args.command == "allocate":
            if any(record["state"] == "ACTIVE" and record["workload_id"] == args.workload_id
                   for n in ledger["node_capacity"] for record in n["allocations"]):
                raise LedgerError("Active allocation already exists for workload")
            if args.reservation_id:
                reserved_node, reservation = find_record(ledger, "reservations", "reservation_id", args.reservation_id)
                if reserved_node is not node or reservation["state"] != "ACTIVE" or reservation["workload_id"] != args.workload_id:
                    raise LedgerError("Reservation node, state, or workload mismatch")
                if any(number(reservation[field]) != amounts[field] for field, *_ in DIMENSIONS):
                    raise LedgerError("Allocation must consume the exact reservation")
                reservation["state"] = "CONSUMED"
            else:
                for field, total, owner, reserved, allocated in DIMENSIONS:
                    if number(node[total]) - number(node[owner]) - number(node[reserved]) - number(node[allocated]) < amounts[field]:
                        raise LedgerError(f"Insufficient {field} on {args.node}")
            for field, _, _, reserved, allocated in DIMENSIONS:
                if args.reservation_id:
                    node[reserved] = number(node[reserved]) - amounts[field]
                node[allocated] = number(node[allocated]) + amounts[field]
            identifier = "alloc-" + uuid.uuid4().hex
            node["allocations"].append({"allocation_id": identifier, "workload_id": args.workload_id,
                "reservation_id": args.reservation_id, **amounts, "state": "ACTIVE", "created_at": stamp()})
            message = "STATUS: ALLOCATED\nAllocation ID: " + identifier
        elif args.command in ("deallocate", "release-reservation"):
            is_alloc = args.command == "deallocate"
            collection = "allocations" if is_alloc else "reservations"
            key = "allocation_id" if is_alloc else "reservation_id"
            node, record = find_record(ledger, collection, key, getattr(args, key))
            if record["state"] != "ACTIVE":
                raise LedgerError(f"{key} is {record['state']}, not ACTIVE")
            record["state"] = "RELEASED"
            for field, _, _, reserved, allocated in DIMENSIONS:
                counter = allocated if is_alloc else reserved
                node[counter] = number(node[counter]) - number(record[field])
            message = "STATUS: DEALLOCATED" if is_alloc else "STATUS: RESERVATION_RELEASED"
    print(message)


if __name__ == "__main__":
    try:
        main()
    except (LedgerError, OSError, ValueError, KeyError, json.JSONDecodeError) as exc:
        print(f"ERROR: {exc}", file=sys.stderr)
        sys.exit(1)
