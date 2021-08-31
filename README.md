**文档内容**

- [10086 移动云容器灾备切换服务](#10086-移动云容器灾备切换服务)
  - [Getting Started](#getting-started)
    - [Prerequisites](#prerequisites)

<!-- END doctoc generated TOC please keep comment here to allow auto update -->

# 10086 移动云容器灾备切换服务

本仓库主要功能是在移动云 k8s 集群宕机的时候，从存储的数据中恢复 k8s 在生产中的服务，恢复的数据包括 k8s 中的各种资源类型，PV 中保存的数据以提供给 POD 使用。

## Getting Started

### Prerequisites

Velero 必须同时安装在生产和容灾环境，生产的 Velero 数据存储容灾环境可以访问
