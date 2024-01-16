# YamlPartitioner

[![Latest Release](https://img.shields.io/github/release/asokolov365/YamlPartitioner.svg?style=flat-square)](https://github.com/asokolov365/YamlPartitioner/releases/latest)
[![GitHub license](https://img.shields.io/github/license/asokolov365/YamlPartitioner.svg)](https://github.com/asokolov365/YamlPartitioner/blob/master/LICENSE)
[![Go Report](https://goreportcard.com/badge/github.com/asokolov365/YamlPartitioner)](https://goreportcard.com/report/github.com/asokolov365/YamlPartitioner)
[![Build Status](https://github.com/asokolov365/YamlPartitioner/workflows/main/badge.svg)](https://github.com/asokolov365/YamlPartitioner/actions)
[![codecov](https://codecov.io/gh/asokolov365/YamlPartitioner/branch/master/graph/badge.svg)](https://codecov.io/gh/asokolov365/YamlPartitioner)

The YamlPartitioner is a small yet powerful Golang command-line application designed to transform originally non-clustered applications into efficiently functioning clusters. Leveraging a consistent hashing algorithm, this tool facilitates seamless partitioning of YAML configuration files across multiple instances of the application.
Notably, it supports partitioning at an arbitrary level on the YAML nodes tree, providing a high degree of flexibility in various areas of application.
This capability, combined with replication factor support, empowers applications to operate as highly available clusters, offering a horizontally scalable and organized solution for managing configurations with ease.

This tool is specifically crafted to streamline the management of setups dealing with a substantial number of items in configuration file(s), enabling the seamless organization, maintenance, and scaling.

The YamlPartitioner is available in [binary releases](https://github.com/asokolov365/YamlPartitioner/releases/latest) and [source code](https://github.com/asokolov365/YamlPartitioner).

The YamlPartitioner consists of a single small executable without external dependencies.


## Features

- **Arbitrary partitioning level (aka split-level):** Supports partitioning at an arbitrary level on the YAML nodes tree. For successful partitioning, the specified "split-level" node must be either a *Mapping* or *Sequence* node in the input YAML file(s).

- **Anchors and Aliases:** Supports YAML [*Anchors* and *Aliases*](https://yaml.org/spec/1.2.2/#3222-anchors-and-aliases). If the "split-level" node is an *Alias* node or contains a list/map of *Alias* nodes, the YamlPartitioner treats it as a corresponding *Anchor* node(s), ensuring logical consistency in the resulting file.

- **Consistent Hashing:** Utilizes a consistent hashing algorithm to ensure balanced and consistent partitioning of YAML configuration regardless of the number of runs or platform architecture.

- **Replication Factor:** Supports a replication factor setting, ensuring the same item appears in a specified number of shards for fault tolerance and redundancy.

- **Original YAML Structure:** Preserves the original YAML file structure, including comments and the sequence of YAML nodes.

- **Batch Partitioning:** Supports partitioning of multiple identical input files at once. This feature streamlines the process when dealing with multiple identical configurations, enabling efficient and consistent partitioning across them.

- **Preserve Folder Hierarchy:** In case of *Batch Partitioning* it supports retaining the original folder structure of input files. This feature ensures that the partitioned output maintains the same organizational hierarchy as the input files, facilitating clarity and ease of navigation.

- **Command-Line Interface:** Simple and intuitive command-line interface for ease of use.

- **Flexible Configuration:** Allows users to customize partitioning based on their specific needs and criteria. The YamlPartitioner supports config params as ENV vars as well as CLI args.


## Areas of application

Horizontal scaling is a common approach to handle increased workloads by adding more resources. However, scaling some systems that are not designed to be scaled horizontally introduces lots of toil and complexity in managing sharded configurations. The YamlPartitioner addresses this challenge by allowing users to maintain original YAML config files of such systems while partitioning them on deployment stage, ensuring efficient and reliable distribution across multiple instances.

Initially developed for clustering and horizontally scaling monitoring setups like [Blackbox Exporter](https://github.com/prometheus/blackbox_exporter/blob/master/CONFIGURATION.md), Prometheus [Alerting](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/), and [Recording](https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/) rules, the YamlPartitioner proves to be a valuable tool for simplifying the management of configurations in horizontally scaled setups. Leveraging consistent hashing and a replication factor setting, it ensures an efficient and fault-tolerant distribution of items, contributing to the scalability and reliability of your monitoring infrastructure. Incorporate the YamlPartitioner into your workflow to streamline configuration management in your horizontally scaled environments.

Happy scaling!

## Getting Started

### Installation

To quickly try the YamlPartitioner, just download the [YamlPartitioner](https://github.com/asokolov365/YamlPartitioner/releases/latest) executable for a desired platform and play with the YAML config file(s) that you want to partition. Use `-v` flag for an extensive report.


- **Install into your app Docker image** (linux-amd64 example):

```Dockerfile
# Install YamlPartitioner
ARG YP_VERSION=v0.1.0
# Get checksum at https://github.com/asokolov365/YamlPartitioner/releases/download/${YP_VERSION}/yp-linux-amd64-${YP_VERSION}_checksums.txt
RUN export YP_SHA256SUM=4c5e6e67e7c6ce4a1df56da816175e286389a9545466d55eba3c914f3c7d1a6d && \
    curl -Lso /tmp/YamlPartitioner.tar.gz \
        "https://github.com/asokolov365/YamlPartitioner/releases/download/${YP_VERSION}/yp-linux-amd64-${YP_VERSION}.tar.gz" && \
    echo "${YP_SHA256SUM}  /tmp/YamlPartitioner.tar.gz" | sha256sum -c && \
    tar -zxf /tmp/YamlPartitioner.tar.gz -C /bin && \
    rm -f /tmp/YamlPartitioner.tar.gz
```

### Usage

*For complete list of supported flags run `yp --help`.*

### Environment variables

The YamlPartitioner supports the following config params as Environment variables:

- `YP_SPLIT_POINT` represents the `--split-at` flag.
- `YP_SRC_PATH` represents the `--src` flag.
- `YP_DST_PATH` represents the `--dst` flag.
- `YP_SHARD_BASENAME` represents the `--shard-basename` flag.
- `YP_SHARDS_NUMBER` represents the `--shards-number` flag.
- `YP_SHARD_ID` represents the `--shard-id` flag.
- `YP_REPLICATION_FACTOR` represents the `--replication` flag.

Please note, CLI flags have precedence over Environment variables.

## Use cases

### Example 1 - all shards at once mode:

This is useful for testing or centralized preparation of configurations for all application instances with GitHub Actions. For example, uploading sharded configs to the Artifactory for future usage by the application instances.

This will partition the Prometheus [Alerting](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/) and [Recording](https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/) rules located in `/tmp/rules/` folder over `5` shards with replication factor `2`. The result will be consistently stored in `/tmp/test/instance.0`, `/tmp/test/instance.1`, `/tmp/test/instance.2`, `/tmp/test/instance.3`, and `/tmp/test/instance.4`.

```bash
yp --replication=2 --split-at="groups.*.rules" --src="/tmp/rules/**/*.{yml,yaml}" --dst=/tmp/test --shards-number=5
```

```
Partitioning of 52 yaml files finished in 75 ms
Shard "instance.0" got 199 items in total
Shard "instance.1" got 214 items in total
Shard "instance.2" got 249 items in total
Shard "instance.3" got 234 items in total
Shard "instance.4" got 218 items in total
```

Check the result:

```bash
ls -lR /tmp/test
```

### Example 2 - specified shard mode:

This is useful for *continuous* partitioning of the original configuration on the application instance side, for example in the application's main or sidecar docker container.

This will partition the same Prometheus [Alerting](https://prometheus.io/docs/prometheus/latest/configuration/alerting_rules/) and [Recording](https://prometheus.io/docs/prometheus/latest/configuration/recording_rules/) rules located in `/tmp/rules/` folder as in example 1. The result will be consistently stored in `/tmp/test/instance.2`.

Please note, the YamlPartitioner must run with the same set of CLI flags, like `replication`, `shards-number`, `split-at`, on all application instances. The only exception is `shard-id` flag, which represents the index of the particular instance in the list of shards.

```bash
yp --replication=2 --split-at="groups.*.rules" --src="/tmp/rules/**/*.{yml,yaml}" --dst=/tmp/test --shards-number=5 --shard-id=2
```

```
Partitioning of 52 yaml files finished in 27 ms
Shard "instance.2" got 249 items in total
```

Check the result:

```bash
ls -lR /tmp/test/instance.2
```

