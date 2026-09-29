"""An unreachable guest must leave a useful SSH diagnostic and failed state."""

import json
import os
from pathlib import Path
import subprocess
import tempfile


source = Path(__file__).resolve().parents[1] / "scripts/bootstrap-nodes.sh"
with tempfile.TemporaryDirectory() as td:
    root = Path(td)
    scripts = root / "validation/local-vm/scripts"
    scripts.mkdir(parents=True)
    script = scripts / "bootstrap-nodes.sh"
    script.write_bytes(source.read_bytes())

    node = root / "validation/local-vm/state/node-1"
    node.mkdir(parents=True)
    (node / "qemu.pid").write_text(str(os.getpid()))
    (node / "seed.iso").write_text("seed")
    (node / "serial.log").write_text("boot output\n")
    state = root / "validation/local-vm/state/cluster.json"
    state.write_text(json.dumps({
        "nodes": 1,
        "status": "RUNNING",
        "virtualization_mode": "tcg",
        "node_details": [{
            "name": "dh-node-1",
            "ssh_port": 2201,
            "qemu_pid_file": str(node / "qemu.pid"),
            "serial_log": str(node / "serial.log"),
        }],
    }))
    home = root / "home"
    (home / ".ssh").mkdir(parents=True)
    (home / ".ssh/p1-local-vm").write_text("test placeholder")
    bin_dir = root / "bin"
    bin_dir.mkdir()
    ssh = bin_dir / "ssh"
    ssh.write_text("#!/bin/sh\necho 'Connection refused' >&2\nexit 255\n")
    ssh.chmod(0o755)
    env = dict(os.environ, HOME=str(home),
               PATH=f"{bin_dir}:{os.environ['PATH']}", P1_BOOT_TIMEOUT_SECONDS="1")
    result = subprocess.run(["bash", str(script)], env=env, capture_output=True,
                            text=True, timeout=15, check=False)
    diagnostic = (node / "ssh-readiness-diagnostic.log").read_text()
    assert result.returncode != 0 and "ready=0/1" in result.stdout
    assert "Connection refused" in diagnostic and "virtualization=tcg" in diagnostic
    assert json.loads(state.read_text())["status"] == "BOOTSTRAP_FAILED"
print("PASS: SSH failure preserves per-node diagnostics and failed state")
