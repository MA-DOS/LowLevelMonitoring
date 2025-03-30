#!/bin/bash

# if [ "$EUID" -ne 0 ]; then
#     echo "Please run as root"
#     exit
# fi

# Exit on error
set -e

# Trap errors and execute the cleanup function
trap cleanup ERR

# Define the dirs for the config files
USER=nfomin3
PROM_DIR=/home/$USER/dev/LowLevelMonitoring/prometheus
PROC_CONFIG=/home/$USER/dev/process-exporter/config/process-exporter-config.yml
SCAPH_EXPORTER=/home/$USER/dev/scaphandre/target/release
CGROUP_EXPORTER_SLURM=/home/$USER/dev/cgroups-exporter
CGROUP_EXPORTER=/home/$USER/dev/cgroups-exporter-mosquito
SLURM_EXPORTER=/home/$USER/dev/SlurmExporter/bin
PROC_EXPORTER=/usr/local/bin
CADVISOR_EXPORTER=/home/$USER/dev/cadvisor/_output
DOCKER_ACTIVITY=/home/$USER/dev/docker-activity/example

# Cleanup function
cleanup() {
     pkill -f process-exporter || true
     pkill -f cgroups_exporter || true
     pkill -f cgroups-exporter || true
     pkill -f cadvisor || true
     pkill -f scaphandre || true
     docker rm -f node-exporter > /dev/null 2>&1 || true
     pkill -f prometheus || true
     pkill -f slurm_exporter || true
     pkill -f ceems_exporter || true
     cd $DOCKER_ACTIVITY && docker-compose down
}

# Kill any running instances of the programs
 pkill -f process-exporter || true
 pkill -f cgroups_exporter || true
 pkill -f cgroups-exporter || true
 pkill -f cadvisor || true
 pkill -f scaphandre || true
 docker rm -f node-exporter > /dev/null 2>&1 || true
 pkill -f prometheus || true
 pkill -f slurm_exporter || true
 pkill -f ceems_exporter || true
 cd $DOCKER_ACTIVITY && docker-compose down

# Run the services
echo "Monitoring Stack is starting..."

# Run process exporter
echo "Navigating to $PROC_EXPORTER"
cd $PROC_EXPORTER
echo "Reading in configuration at $PROC_CONFIG"
sleep 2
echo "Running process-exporter"
sleep 2
./process-exporter --procfs /proc --config.path $PROC_CONFIG > /dev/null 2>&1 & 

# Run docker-activity via docker compose
echo "Navigating to $DOCKER_ACTIVITY"
cd $DOCKER_ACTIVITY
echo "Running docker-activity"
sleep 2
docker-compose up -d

# Run ceems exporter
echo "Running ceems exporter"
sleep 2
docker run -p 9010:9010 -d --privileged mahendrapaipuri/ceems:latest ceems_exporter

# Run cgroups exporter for slurm
# echo "Navigating to $CGROUP_EXPORTER_SLURM"
# cd $CGROUP_EXPORTER_SLURM
# echo "Running cgroups-exporter for slurm"
# sleep 2
# ./cgroups_exporter -method slurm > /dev/null 2>&1 &

# Run cgroups exporter
# echo "Navigating to $CGROUP_EXPORTER"
# cd $CGROUP_EXPORTER
# echo "Running cgroups-exporter mosquito"
# sleep 2
# python3 -m cgroups_exporter --cgroups-path "/sys/fs/cgroup/*/docker/*" > /dev/null 2>&1 &
# if [ $? -ne 0 ]; then
#     echo "Failed to start cgroups-exporter mosquito. Check cgroups_exporter_mosquito.log for details."
#     cleanup
#     exit 1
# fi

# Run cadvisor exporter
echo "Navigating to $CADVISOR_EXPORTER"
cd $CADVISOR_EXPORTER
echo "Running cadvisor"
sleep 2
./cadvisor --docker_only=true --port=8081 > /dev/null 2>&1 &

# Run slurm exporter
echo "Navigating to $SLURM_EXPORTER"
cd $SLURM_EXPORTER
echo "Running slurm-exporter"
sleep 2
./slurm_exporter > /dev/null 2>&1 &

# Run scaphandre
# echo "Navigating to $SCAPH_EXPORTER"
# cd $SCAPH_EXPORTER
# echo "Running scaphandre"
# sleep 2
# ./scaphandre prometheus --containers > /dev/null 2>&1 &

# Run node-exporter
echo "Running node-exporter in docker container"
sleep 2
docker run -d -p 9100:9100 --name=node-exporter --restart=always prom/node-exporter

# Run ebpf-energy-exporter

# Run prometheus
echo "Navigating to $PROM_DIR"
cd $PROM_DIR
echo "Running prometheus"
sleep 2
docker run -d \
    -p 9090:9090 \
    -v $(pwd)/prometheus.yml:/etc/prometheus/prometheus.yml \
    prom/prometheus

# Health Check
if [ $? -eq 0 ]; then
    echo "All services started successfully."
else
    echo "An error occurred while starting the services."
    cleanup
fi

echo "Monitoring stack running. Keeping the process alive."
tail -f /dev/null




