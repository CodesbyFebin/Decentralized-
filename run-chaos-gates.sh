#!/bin/bash
# Wrapper to run chaos gates with proper environment

cd /home/user/Decentralized-
export DH_HOME="/tmp/dh-cluster/operator"
export DH_CLUSTER="dev"
export PATH="/home/user/Decentralized-/bin:$PATH"

./validation/gates-25-32-direct.sh
